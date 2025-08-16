// Package enum provides a generic implementation of enumerated types (enums) in Go.
// The Maker type in this package offers a reflection-based approach to create enums
// from struct fields, assigning sequential integer values to exported fields of a struct.
// It is designed for convenience in defining static enums where compile-time type safety
// is less critical than ease of use. Unlike the Generator type, Maker uses reflection,
// which is less performant and lacks the flexibility of programmatic enum generation.
//
// Maker[T, E] creates an enum by populating a struct’s fields with values of type E
// (constrained to integer types) and maintains mappings of values to field names and
// vice versa. It supports JSON serialization and provides methods for lookup and validation.
//
// Warning: Due to its use of reflection, Maker is less performant than using Go’s const/iota
// or the Generator type. It is best suited for simple, static enums where convenience
// outweighs performance concerns.
//
// Example usage:
//
//	type Colors struct {
//	    Red   int
//	    Blue  int
//	    Green int
//	}
//	var c Colors
//	m := enum.Make[Colors, int](&c)
//	fmt.Println(c.Red)         // Output: 0
//	fmt.Println(m.Name(1))     // Output: Blue, true
//	fmt.Println(m.Get("Green")) // Output: 2, true
package enum

import (
	"encoding/json"
	"fmt"
	"reflect"
)

// MakeEnumValue is a type constraint for enum values used with Maker.
// It restricts the underlying type to integers (signed or unsigned), as Maker
// assigns sequential integer values to struct fields.
type MakeEnumValue interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64
}

// Maker provides a reflection-based mechanism to create enums from struct fields.
// It populates exported fields of a struct (type T) with sequential values of type E
// (an integer type constrained by MakeEnumValue) and maintains mappings of values to
// field names and vice versa. It is less performant than other enum approaches due to
// reflection but is convenient for simple, static enums.
//
// The Maker is not thread-safe, as it is designed for initialization and read-only access
// after creation. Use Make to create a Maker instance.
type Maker[T any, E MakeEnumValue] struct {
	instance *T           // Pointer to the populated struct instance.
	valueMap map[E]string // Maps enum values to field names.
	nameMap  map[string]E // Maps field names to enum values.
	entries  []Value[E]   // Slice of all enum entries.
}

// Make creates a new Maker instance from a struct pointer, assigning sequential
// integer values of type E to its exported fields. It uses reflection to iterate
// through the struct’s fields, setting each exported field to a value (starting from 0)
// and building value-to-name and name-to-value mappings.
//
// Panics if:
// - The provided construct is not a pointer to a struct.
// - The number of fields exceeds the capacity of the underlying type E (e.g., 256 for int8).
// - The struct contains unexported fields that cannot be set (these are skipped silently).
//
// Warning: This function uses reflection, which is less performant and lacks the
// compile-time type safety of Go’s const/iota or the Generator type. Use it for
// simple, static enums where convenience is prioritized over performance.
//
// Example:
//
//	type Status struct {
//	    Pending int
//	    Active  int
//	    Done    int
//	}
//	var s Status
//	m := Make[Status, int](&s)
//	fmt.Println(s.Pending)     // Output: 0
//	fmt.Println(m.Name(1))     // Output: Active, true
//	fmt.Println(m.Get("Done")) // Output: 2, true
func Make[T any, E MakeEnumValue](construct *T) *Maker[T, E] {
	val := reflect.ValueOf(construct)
	if val.Kind() != reflect.Ptr || val.Elem().Kind() != reflect.Struct {
		panic("enum.Make: construct must be a pointer to a struct")
	}

	elem := val.Elem()
	rc := elem.Type()
	n := rc.NumField()

	var e E
	typeE := reflect.TypeOf(e)

	var capacity uint64
	kind := typeE.Kind()
	if kind >= reflect.Int && kind <= reflect.Int64 {
		// Signed integers can hold 2^(bits-1) non-negative values.
		capacity = uint64(1) << (typeE.Bits() - 1)
	} else if kind >= reflect.Uint && kind <= reflect.Uint64 {
		// Unsigned integers can hold 2^bits values.
		capacity = uint64(1) << typeE.Bits()
	}

	if capacity > 0 && uint64(n) >= capacity {
		panic(fmt.Sprintf("enum.Make: number of struct fields (%d) exceeds the capacity of the underlying enum type %s", n, typeE.Name()))
	}

	maxFields := 1 << (typeE.Bits() - 1)
	if n >= maxFields && maxFields > 0 {
		panic(fmt.Sprintf("enum.Make: number of struct fields (%d) exceeds the capacity of the underlying enum type %s", n, typeE.Name()))
	}

	valueMap := make(map[E]string, n)
	nameMap := make(map[string]E, n)
	entries := make([]Value[E], 0, n)

	for i := 0; i < n; i++ {
		field := rc.Field(i)
		fieldVal := elem.Field(i)

		if !fieldVal.CanSet() {
			continue // Skip unexported fields
		}

		value := E(i)
		fieldVal.Set(reflect.ValueOf(value).Convert(field.Type))

		valueMap[value] = field.Name
		nameMap[field.Name] = value
		entries = append(entries, New(value, field.Name))
	}

	return &Maker[T, E]{
		instance: construct,
		valueMap: valueMap,
		nameMap:  nameMap,
		entries:  entries,
	}
}

// Struct returns the pointer to the populated struct instance.
// The struct’s exported fields contain the assigned enum values.
//
// Example:
//
//	type Colors struct {
//	    Red   int
//	    Blue  int
//	}
//	var c Colors
//	m := Make[Colors, int](&c)
//	fmt.Println(m.Struct().Red)  // Output: 0
//	fmt.Println(m.Struct().Blue) // Output: 1
func (e *Maker[T, E]) Struct() *T {
	return e.instance
}

// Get returns the enum value associated with a given field name.
// Returns the value and true if the name exists, or the zero value of E and false otherwise.
//
// Example:
//
//	m := Make[Colors, int](&Colors{})
//	val, ok := m.Get("Red") // Returns 0, true
func (e *Maker[T, E]) Get(name string) (E, bool) {
	val, ok := e.nameMap[name]
	return val, ok
}

// Name returns the field name associated with a given enum value.
// Returns the name and true if the value exists, or an empty string and false otherwise.
//
// Example:
//
//	m := Make[Colors, int](&Colors{})
//	name, ok := m.Name(1) // Returns "Blue", true
func (e *Maker[T, E]) Name(value E) (string, bool) {
	name, ok := e.valueMap[value]
	return name, ok
}

// Data is deprecated in favor of ValueMap.
// It returns the map of enum values to their field names.
// Use ValueMap for clarity in new code.
func (e *Maker[T, E]) Data() map[E]string {
	return e.valueMap
}

// Names returns a slice of all enum field names.
//
// Example:
//
//	m := Make[Colors, int](&Colors{})
//	names := m.Names() // Returns ["Red", "Blue"]
func (e *Maker[T, E]) Names() []string {
	names := make([]string, 0, len(e.entries))
	for _, entry := range e.entries {
		names = append(names, entry.String())
	}
	return names
}

// Entries returns a slice of all enum entries as Value[E].
//
// Example:
//
//	m := Make[Colors, int](&Colors{})
//	entries := m.Entries() // Returns [{0 Red}, {1 Blue}]
func (e *Maker[T, E]) Entries() []Value[E] {
	return e.entries
}

// Contains checks if a value exists in the enum set.
//
// Example:
//
//	m := Make[Colors, int](&Colors{})
//	ok := m.Contains(0) // Returns true
//	ok = m.Contains(99) // Returns false
func (e *Maker[T, E]) Contains(value E) bool {
	_, ok := e.valueMap[value]
	return ok
}

// ContainsName checks if a field name exists in the enum set.
//
// Example:
//
//	m := Make[Colors, int](&Colors{})
//	ok := m.ContainsName("Red")   // Returns true
//	ok := m.ContainsName("Green") // Returns false
func (e *Maker[T, E]) ContainsName(name string) bool {
	_, ok := e.nameMap[name]
	return ok
}

// ValueMap returns the map of enum values to their field names.
//
// Example:
//
//	m := Make[Colors, int](&Colors{})
//	vm := m.ValueMap() // Returns map[int]string{0: "Red", 1: "Blue"}
func (e *Maker[T, E]) ValueMap() map[E]string {
	return e.valueMap
}

// NameMap returns the map of field names to enum values.
//
// Example:
//
//	m := Make[Colors, int](&Colors{})
//	nm := m.NameMap() // Returns map[string]int{"Red": 0, "Blue": 1}
func (e *Maker[T, E]) NameMap() map[string]E {
	return e.nameMap
}

// MarshalJSON implements json.Marshaler, serializing the value-to-name map to JSON.
func (e *Maker[T, E]) MarshalJSON() ([]byte, error) {
	return json.Marshal(e.valueMap)
}

// UnmarshalJSON implements json.Unmarshaler, deserializing a JSON object into the
// value-to-name map. It updates valueMap but does not modify the struct instance or
// entries, as the struct’s fields are set during Make and cannot be safely updated
// via reflection after initialization.
//
// Note: This may leave the Maker in an inconsistent state if the deserialized map
// does not match the struct’s fields.
func (e *Maker[T, E]) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &e.valueMap)
}
