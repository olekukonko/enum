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

// Maker provides a reflection-based mechanism to create enums from struct fields.
// It populates exported fields of a struct (type T) with sequential values of type E
// (an integer type constrained by TypesMake) and maintains mappings of values to
// field names and vice versa. It is less performant than other enum approaches due to
// reflection but is convenient for simple, static enums.
//
// The Maker is not thread-safe, as it is designed for initialization and read-only access
// after creation. Use Make to create a Maker instance.
type Maker[T any, E TypesMake] struct {
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
func Make[T any, E TypesMake](construct *T) *Maker[T, E] {
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
		entries = append(entries, NewValue(value, field.Name))
	}

	return &Maker[T, E]{
		instance: construct,
		valueMap: valueMap,
		nameMap:  nameMap,
		entries:  entries,
	}
}

// MakeManual creates a Maker instance without reflection by using a user-provided
// initialization function to populate the struct and a Generator to manage enum values.
// The init function should use the provided Generator to create enum values and set them
// on the struct. The resulting Maker maintains value-to-name and name-to-value mappings
// consistent with the Generator's state.
//
// Example:
//
//	type Colors struct{ Red, Blue int }
//	var c Colors
//	m := MakeManual(&c, func(g *Generator[int]) *Colors {
//	    c.Red = g.Next("Red").Get()
//	    c.Blue = g.Next("Blue").Get()
//	    return &c
//	})
func MakeManual[T any, E TypesMake](construct *T, init func(*Generator[E]) *T) *Maker[T, E] {
	if construct == nil {
		panic("enum.MakeManual: construct must not be nil")
	}

	// Create a Generator with default settings
	g := NewGenerator[E]()

	// Initialize struct using user-provided function
	result := init(g)
	if result != construct {
		panic("enum.MakeManual: init function must return the same struct pointer as construct")
	}

	return &Maker[T, E]{
		instance: construct,
		valueMap: g.ValueMap(),
		nameMap:  g.NameMap(),
		entries:  g.Values(),
	}
}

// MakeManualWithBasic creates a Maker instance without reflection by using a user-provided
// initialization function to populate the struct with Basic values. The init function should
// use the provided Basic to create enum values and set them on the struct. The resulting
// Maker maintains value-to-name and name-to-value mappings consistent with the Basic's state.
//
// Example:
//
//	type Colors struct{ Red, Blue Basic }
//	var c Colors
//	b := NewBasic()
//	m := MakeManualWithBasic(&c, b, func(b *Basic) *Colors {
//	    c.Red = b.Add("Red")
//	    c.Blue = b.Add("Blue")
//	    return &c
//	})
func MakeManualWithBasic[T any](construct *T, b *Basic, init func(*Basic) *T) *Maker[T, int] {
	if construct == nil {
		panic("enum.MakeManualWithBasic: construct must not be nil")
	}
	if b == nil {
		panic("enum.MakeManualWithBasic: Basic instance must not be nil")
	}

	// Initialize the user's struct using their provided function.
	// This populates the internal state of the Generator within 'b'.
	result := init(b)
	if result != construct {
		panic("enum.MakeManualWithBasic: init function must return the same struct pointer as construct")
	}

	// Create the Maker by using the public, thread-safe methods of the
	// underlying Generator. This is safer and cleaner than accessing
	// internal fields directly.
	return &Maker[T, int]{
		instance: construct,
		valueMap: b.meta.ValueMap(),
		nameMap:  b.meta.NameMap(),
		entries:  b.meta.Values(),
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
	// Check if instance is initialized
	if e.instance == nil {
		return fmt.Errorf("cannot unmarshal: Maker instance is nil")
	}

	// Deserialize into temporary map
	var tempMap map[E]string
	if err := json.Unmarshal(data, &tempMap); err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	// Validate against struct fields
	elem := reflect.ValueOf(e.instance).Elem()
	if elem.Kind() != reflect.Struct {
		return fmt.Errorf("instance is not a struct")
	}
	rc := elem.Type()
	tempNameMap := make(map[string]E, len(tempMap))
	tempEntries := make([]Value[E], 0, len(tempMap))

	// Check that all struct fields are represented in tempMap
	for i := 0; i < rc.NumField(); i++ {
		field := rc.Field(i)
		if !elem.Field(i).CanSet() {
			continue // Skip unexported fields
		}
		value, ok := e.nameMap[field.Name]
		if !ok {
			return fmt.Errorf("field %q not found in nameMap", field.Name)
		}
		name, ok := tempMap[value]
		if !ok || name != field.Name {
			return fmt.Errorf("invalid value %v or name %q for field %q", value, name, field.Name)
		}
		tempNameMap[name] = value
		tempEntries = append(tempEntries, NewValue(value, name))
	}

	// Check for extra entries in tempMap
	for value, name := range tempMap {
		if _, ok := tempNameMap[name]; !ok {
			return fmt.Errorf("unexpected value %v with name %q in JSON", value, name)
		}
	}

	// Update state
	e.valueMap = tempMap
	e.nameMap = tempNameMap
	e.entries = tempEntries
	return nil
}
