package enum

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
)

// Basic represents a simple, integer-based enum value with a name and a centralized
// registry for managing all values of a given enum set. It is designed for
// straightforward enum definitions where integer values are automatically assigned
// (starting from 0) or customized via With. Unlike Value[T], Basic is not generic,
// offering better performance for integer-based enums by avoiding generics overhead.
// It supports JSON marshaling/unmarshaling and SQL driver integration for seamless
// use in serialization and database operations.
//
// Basic is thread-safe, using Generator[int] internally to manage value-to-name and
// name-to-value mappings. It is ideal for simple enums (e.g., status codes) where
// type safety across different enum sets is less critical than ease of use.
//
// Example:
//
//	b := NewBasic()
//	pending := b.Add("Pending") // value: 0
//	active := b.Add("Active")   // value: 1
//	fmt.Println(pending.Get(), pending.String()) // Output: 0, "Pending"
//	data, _ := active.MarshalJSON()
//	fmt.Println(string(data)) // Output: 1
type Basic struct {
	name  string          // Human-readable name of the enum value.
	value int             // Integer value of the enum.
	meta  *Generator[int] // Internal registry for value-to-name mappings.
}

// NewBasic creates a new enum registry for Basic values. It initializes a Generator[int]
// with automatic numbering starting from 0. Each call to Add on the returned Basic
// instance creates a new enum value with the next sequential integer.
//
// Returns a Basic instance ready to define enum values via Add or With.
//
// Example:
//
//	b := NewBasic()
//	pending := b.Add("Pending") // value: 0
//	active := b.Add("Active")   // value: 1
func NewBasic() *Basic {
	// Note: We use a pointer here for the initial instance so that the `meta`
	// field can be shared across all enum values created from it.
	return &Basic{
		meta: NewNumeric[int](0),
	}
}

// Add defines a new enum value with the given name, automatically assigning the next
// sequential integer value (starting from 0 for the first value). It updates the
// internal registry to map the value to the name and vice versa.
//
// Returns a new Basic instance representing the enum value.
//
// Panics if the name is already used in the registry.
//
// Example:
//
//	b := NewBasic()
//	pending := b.Add("Pending") // value: 0
//	active := b.Add("Active")   // value: 1
//	// b.Add("Pending") // Panics because the name is already used.
func (e *Basic) Add(name string) Basic {
	// The underlying Generator's Next() method is thread-safe.
	v := e.meta.Next(name)
	return Basic{
		name:  v.String(),
		value: v.Get(),
		meta:  e.meta,
	}
}

// With assigns a custom integer value to the enum, updating the internal registry.
// This operation is atomic and thread-safe. If the value is already used, it panics to
// prevent conflicts. If the Basic instance already has a value, it removes the old
// mappings before assigning the new value.
//
// Returns a new Basic instance with the custom value.
//
// Panics if the value is already used in the registry.
//
// Example:
//
//	b := NewBasic()
//	pending := b.Add("Pending") // value: 0
//	custom := pending.With(100) // Reassigns Pending to value 100
//	fmt.Println(custom.Get())    // Output: 100
func (e Basic) With(v int) Basic {
	e.meta.mu.Lock()
	defer e.meta.mu.Unlock()

	if existing, ok := e.meta.valueMap[v]; ok {
		panic(fmt.Sprintf("value %d already used for %q", v, existing))
	}

	// Remove old mappings if they exist. This check ensures we only remove
	// the value if it's still associated with the correct name, preventing
	// incorrect deletions in complex scenarios.
	if oldName, ok := e.meta.valueMap[e.value]; ok && oldName == e.name {
		delete(e.meta.valueMap, e.value)
		delete(e.meta.nameMap, e.name)
		// Note: We don't remove from the `e.meta.values` slice for simplicity
		// and performance, as it would require a linear scan. The lookup maps
		// are the source of truth for all critical operations.
	}

	// Add new mappings
	e.meta.valueMap[v] = e.name
	e.meta.nameMap[e.name] = v
	e.meta.values = append(e.meta.values, NewValue(v, e.name))

	return Basic{
		name:  e.name,
		value: v,
		meta:  e.meta,
	}
}

// String returns the human-readable name of the enum value.
//
// Implements fmt.Stringer.
//
// Example:
//
//	b := NewBasic()
//	pending := b.Add("Pending")
//	fmt.Println(pending.String()) // Output: "Pending"
func (e Basic) String() string {
	return e.name
}

// Get returns the integer value of the enum.
//
// Example:
//
//	b := NewBasic()
//	pending := b.Add("Pending")
//	fmt.Println(pending.Get()) // Output: 0
func (e Basic) Get() int {
	return e.value
}

// Validate checks if the enum value is valid by verifying its presence in the registry.
// Returns nil if the value exists, or an error if it does not.
//
// Example:
//
//	b := NewBasic()
//	pending := b.Add("Pending")
//	err := pending.Validate() // Returns nil
//	invalid := Basic{value: 999, meta: b.meta}
//	err = invalid.Validate()  // Returns error: "invalid enum value: 999"
func (e Basic) Validate() error {
	if _, ok := e.meta.Name(e.value); !ok {
		return fmt.Errorf("invalid enum value: %d", e.value)
	}
	return nil
}

// MarshalJSON implements json.Marshaler, serializing the enum value to its integer value.
//
// Example:
//
//	b := NewBasic()
//	pending := b.Add("Pending")
//	data, _ := pending.MarshalJSON()
//	fmt.Println(string(data)) // Output: 0
func (e Basic) MarshalJSON() ([]byte, error) {
	return json.Marshal(e.value)
}

// UnmarshalJSON implements json.Unmarshaler, deserializing an integer value from JSON
// and updating the Basic instance with the corresponding name from the registry.
// Returns an error if the value is not found in the registry, if the `meta` field is nil,
// or if JSON parsing fails.
//
// Example:
//
//	b := NewBasic()
//	b.Add("Pending")
//	data := []byte("0")
//	var e2 Basic
//	e2.meta = b.meta // IMPORTANT: The registry must be assigned before unmarshaling.
//	err := e2.UnmarshalJSON(data) // Sets e2 to {name: "Pending", value: 0}
func (e *Basic) UnmarshalJSON(data []byte) error {
	if e.meta == nil {
		return errors.New("cannot unmarshal into Basic enum with nil registry (meta)")
	}
	var val int
	if err := json.Unmarshal(data, &val); err != nil {
		return err
	}

	name, exists := e.meta.Name(val)
	if !exists {
		return fmt.Errorf("invalid enum value: %d", val)
	}
	e.value = val
	e.name = name
	return nil
}

// Value implements driver.Valuer, returning the enum value as an int64 for SQL storage.
//
// Example:
//
//	b := NewBasic()
//	pending := b.Add("Pending")
//	val, _ := pending.Value() // Returns int64(0)
func (e Basic) Value() (driver.Value, error) {
	return int64(e.value), nil
}

// Scan implements sql.Scanner, parsing an SQL value (int64, float64, string, or []byte)
// into the Basic instance. Updates the name and value based on the registry.
// Returns an error if the value is invalid, unsupported, or if the `meta` field is nil.
//
// Example:
//
//	b := NewBasic()
//	b.Add("Pending")
//	var e2 Basic
//	e2.meta = b.meta // IMPORTANT: The registry must be assigned before scanning.
//	err := e2.Scan(int64(0)) // Sets e2 to {name: "Pending", value: 0}
func (e *Basic) Scan(value interface{}) error {
	if e.meta == nil {
		return errors.New("cannot scan into Basic enum with nil registry (meta)")
	}
	if value == nil {
		// Set to zero value if DB is NULL
		e.value = 0
		e.name = ""
		return nil
	}
	var val int
	switch v := value.(type) {
	case int64:
		val = int(v)
	case float64:
		val = int(v)
	case []byte:
		var err error
		val, err = strconv.Atoi(string(v))
		if err != nil {
			return fmt.Errorf("invalid enum value: %s", string(v))
		}
	case string:
		var err error
		val, err = strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("invalid enum value: %s", v)
		}
	default:
		return fmt.Errorf("unsupported type for scan: %T", value)
	}

	name, exists := e.meta.Name(val)
	if !exists {
		return fmt.Errorf("invalid enum value: %d", val)
	}
	e.value = val
	e.name = name
	return nil
}

// Values returns a slice of all enum values in the registry.
// Each value is a Basic instance with the same meta registry.
//
// Example:
//
//	b := NewBasic()
//	b.Add("Pending")
//	b.Add("Active")
//	values := b.Values() // Returns [{name: "Pending", value: 0}, {name: "Active", value: 1}]
func (e *Basic) Values() []Basic {
	values := e.meta.Values()
	result := make([]Basic, len(values))
	for i, v := range values {
		result[i] = Basic{name: v.String(), value: v.Get(), meta: e.meta}
	}
	return result
}

// ToValue converts a Basic instance to a Value[int] for compatibility with generic
// enum operations in the package.
//
// Example:
//
//	b := NewBasic()
//	pending := b.Add("Pending")
//	v := pending.ToValue() // Returns Value[int]{value: 0, name: "Pending"}
func (e Basic) ToValue() Value[int] {
	return NewValue(e.value, e.name)
}

// FromValue creates a Basic instance from a Value[int], adding it to the registry.
// This is a convenience method that chains Add() and With().
//
// Panics if the value or name already exists in the registry.
//
// Example:
//
//	v := NewValue(10, "Pending")
//	b := NewBasic()
//	pending := b.FromValue(v) // Returns Basic{name: "Pending", value: 10}
func (e *Basic) FromValue(v Value[int]) Basic {
	// Add creates an entry with a temporary value, which With then corrects.
	return e.Add(v.String()).With(v.Get())
}
