// Package enum provides a generic implementation of enumerated types (enums) in Go.
// It supports creating type-safe enums with underlying types such as strings, integers,
// or floating-point numbers, and provides seamless integration with JSON serialization
// and database interactions through the standard library's encoding/json and database/sql
// packages.
//
// The package defines a generic Value[T] type that serves as a reusable base for enum
// entries, supporting operations like string representation, JSON marshaling/unmarshaling,
// and SQL scanning/valuering. Enums are defined by implementing the Entry[T] interface,
// typically by embedding Value[T] in a custom type and providing a Name method for
// mapping values to their string representations.
//
// Example usage:
//
//	type Color struct {
//	    enum.Value[string]
//	}
//	func (c Color) Name(v string) (string, bool) {
//	    switch v {
//	    case "red": return "Red", true
//	    case "blue": return "Blue", true
//	    default: return "", false
//	    }
//	}
//	var Red = Color{Value: enum.New("red", "Red")}
//	var Blue = Color{Value: enum.New("blue", "Blue")}
package enum

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
)

// Types is a constraint that defines all supported underlying types for enums.
// It includes string, all standard integer types (signed and unsigned), and
// floating-point types (float32 and float64).
type Types interface {
	~string | ~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 |
		~float32 | ~float64
}

// Entry defines the interface that all enum types must implement.
// It requires methods to retrieve the underlying value and its string representation.
type Entry[T comparable] interface {
	// Get returns the underlying value of the enum entry.
	Get() T
	// String returns the human-readable name of the enum entry.
	String() string
}

// Value is a generic struct that serves as a reusable base for enum types.
// It stores the underlying value and its string representation, and implements
// methods for JSON marshaling/unmarshaling and SQL scanning/valuering.
// Typically, Value[T] is embedded in a custom enum type that implements Entry[T].
type Value[T comparable] struct {
	value T
	name  string
}

// New creates a new enum value with the given underlying value and name.
// It is used to initialize enum entries.
//
// Example:
//
//	red := enum.New("red", "Red")
//	fmt.Println(red.String()) // Output: Red
//	fmt.Println(red.Get())    // Output: red
func New[T comparable](value T, name string) Value[T] {
	return Value[T]{value: value, name: name}
}

// Get returns the underlying value of the enum entry.
func (e Value[T]) Get() T {
	return e.value
}

// String returns the name of the enum entry, as set during creation.
func (e Value[T]) String() string {
	return e.name
}

// MarshalJSON implements json.Marshaler, serializing the enum's underlying value
// to JSON. The value is marshaled as its raw type (e.g., string, int, float).
func (e Value[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(e.value)
}

// UnmarshalJSON implements json.Unmarshaler, deserializing a JSON value into
// the enum's underlying value. After unmarshaling, it attempts to set the name
// field by calling the Name method on the embedding type, if available. This
// allows the enum to maintain its string representation based on the deserialized
// value.
//
// If the embedding type does not provide a Name method, the name field remains
// unchanged. Errors are returned if the JSON data cannot be unmarshaled to type T.
func (e *Value[T]) UnmarshalJSON(data []byte) error {
	var val T
	if err := json.Unmarshal(data, &val); err != nil {
		return err
	}
	e.value = val
	return nil
}

// Value implements driver.Valuer, returning the enum's underlying value for
// database storage. The value is returned as-is, compatible with SQL drivers.
func (e Value[T]) Value() (driver.Value, error) {
	// Convert numeric types to the standard driver types (int64, float64)
	// to ensure database compatibility.
	val := any(e.value)
	switch v := val.(type) {
	case int, int8, int16, int32, int64:
		return reflect.ValueOf(v).Int(), nil
	case uint, uint8, uint16, uint32, uint64:
		return int64(reflect.ValueOf(v).Uint()), nil
	case float32, float64:
		return reflect.ValueOf(v).Float(), nil
	}
	return e.value, nil
}

// Scan implements sql.Scanner, populating the enum from a database value.
// It supports scanning from int64, float64, string, and []byte types, converting
// them to the enum's underlying type T. After scanning, it attempts to set the
// name field by calling the Name method on the embedding type, if available.
//
// Errors are returned if the database value cannot be converted to type T or
// if the type is unsupported.
func (e *Value[T]) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	var val T
	var err error

	// Handle different database value types
	switch v := value.(type) {
	case int64:
		val, err = safeCast[T](v)
		if err != nil {
			return fmt.Errorf("failed to scan enum: %w", err)
		}
	case float64:
		val, err = safeCast[T](v)
		if err != nil {
			return fmt.Errorf("failed to scan enum: %w", err)
		}
	case []byte:
		val, err = parseStringToValue[T](string(v))
		if err != nil {
			return fmt.Errorf("failed to scan enum from bytes: %w", err)
		}
	case string:
		val, err = parseStringToValue[T](v)
		if err != nil {
			return fmt.Errorf("failed to scan enum from string: %w", err)
		}
	default:
		return fmt.Errorf("unsupported type for enum scan: %T", value)
	}

	e.value = val
	return nil
}

// safeCast converts a numeric value (int64 or float64) to type T, checking for
// out-of-range errors to prevent silent truncation or overflow. It ensures the
// conversion is safe by verifying that converting the value back to the original
// type yields the same value.
//
// Returns an error if the conversion is not possible or if the value is out of
// range for the target type.
func safeCast[T comparable, N int64 | float64](n N) (T, error) {
	var zero T
	targetType := reflect.TypeOf(zero)
	val := reflect.ValueOf(n)

	if !val.CanConvert(targetType) {
		return zero, fmt.Errorf("cannot convert %T to %T", n, zero)
	}

	converted := val.Convert(targetType)
	// Skip round-trip check for floating-point types to avoid precision issues.
	if targetType.Kind() == reflect.Float32 || targetType.Kind() == reflect.Float64 {
		return converted.Interface().(T), nil
	}

	// Check if the conversion resulted in an overflow by converting back.
	if converted.Type().ConvertibleTo(val.Type()) {
		roundTrip := converted.Convert(val.Type())
		if roundTrip.Interface() != val.Interface() {
			return zero, fmt.Errorf("value %v is out of range for type %T", n, zero)
		}
	} else {
		// This case is unlikely with standard numeric types but is a safeguard.
		return zero, fmt.Errorf("non-reciprocal conversion from %T to %T", zero, n)
	}
	return converted.Interface().(T), nil
}

// parseStringToValue converts a string to the enum's underlying type T.
// It supports string, integer, unsigned integer, and floating-point types.
// For numeric types, it parses the string using strconv and checks for
// out-of-range errors to prevent overflow or truncation.
//
// Returns an error if the string cannot be parsed or if the type is unsupported.
func parseStringToValue[T comparable](s string) (T, error) {
	var zero T
	switch any(zero).(type) {
	case string:
		return any(s).(T), nil
	case int, int8, int16, int32, int64:
		val, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return zero, err
		}
		return safeCast[T](val)
	case uint, uint8, uint16, uint32, uint64:
		val, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			return zero, err
		}

		var uzero T
		t := reflect.TypeOf(uzero)
		bitSize := t.Bits()

		if bitSize < 64 {
			maxValue := uint64(1)<<bitSize - 1
			if val > maxValue {
				return zero, fmt.Errorf("value %v is out of range for type %T", val, zero)
			}
		}

		return reflect.ValueOf(val).Convert(t).Interface().(T), nil
	case float32, float64:
		val, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return zero, err
		}
		return safeCast[T](val)
	default:
		return zero, fmt.Errorf("unsupported type for string parsing: %T", zero)
	}
}
