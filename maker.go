package enum

import "reflect"

// MakeEnumValue constraint for enum values used with the Maker.
// It is limited to integer types.
type MakeEnumValue interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64
}

// Maker provides a simple, reflection-based way to create enums from struct fields.
type Maker[T any, E MakeEnumValue] struct {
	values   *T
	valueMap map[E]string
	nameMap  map[string]E
	entries  []Value[E]
}

// Make creates an enum from a struct definition. It iterates through the fields
// of the provided struct, assigning sequential integer values of type E to each field.
// It panics if the number of fields exceeds the capacity of the underlying enum type E.
//
// Note: This function relies on reflection and is less performant and less type-safe
// than using the standard Go `const` and `iota` pattern.
func Make[T any, E MakeEnumValue](construct T) *Maker[T, E] {
	rc := reflect.TypeOf(construct)
	n := rc.NumField()

	// Check if the number of fields exceeds the capacity of the chosen enum type E.
	var e E
	typeE := reflect.TypeOf(e)
	// For unsigned integers, capacity is 1 << (bits). For signed, it's the same on the positive side.
	// We only need to check the smaller types, as `n` is an `int`.
	var maxFields int64
	switch typeE.Kind() {
	case reflect.Int8, reflect.Uint8:
		maxFields = 1 << 8
	case reflect.Int16, reflect.Uint16:
		maxFields = 1 << 16
	}

	if maxFields > 0 && int64(n) >= maxFields {
		panic("enum.Make: number of struct fields exceeds the capacity of the underlying enum type E")
	}

	valueMap := make(map[E]string, n)
	nameMap := make(map[string]E, n)
	entries := make([]Value[E], 0, n)
	constructPtr := reflect.ValueOf(&construct).Elem()

	for i := 0; i < n; i++ {
		field := rc.Field(i)
		// Assign the index `i` as the enum value.
		value := E(i)
		constructPtr.Field(i).Set(reflect.ValueOf(value).Convert(field.Type))

		valueMap[value] = field.Name
		nameMap[field.Name] = value
		entries = append(entries, New(value, field.Name))
	}

	return &Maker[T, E]{
		values:   &construct,
		valueMap: valueMap,
		nameMap:  nameMap,
		entries:  entries,
	}
}

// Get returns the enum value by its field name (e.g., "Red" -> 0).
// Returns false if no field with that name exists.
func (e *Maker[T, E]) Get(name string) (E, bool) {
	val, ok := e.nameMap[name]
	return val, ok
}

// Name returns the name of an enum value (e.g., 0 -> "Red").
// Returns false if the value is not defined in the enum.
func (e *Maker[T, E]) Name(value E) (string, bool) {
	name, ok := e.valueMap[value]
	return name, ok
}

// Values returns the complete map of enum values to their string names.
func (e *Maker[T, E]) Values() map[E]string {
	return e.valueMap
}

// Names returns all enum names as a slice of strings.
func (e *Maker[T, E]) Names() []string {
	names := make([]string, 0, len(e.entries))
	for _, entry := range e.entries {
		names = append(names, entry.String())
	}
	return names
}

// Entries returns all enum entries as a slice of Value[E].
func (e *Maker[T, E]) Entries() []Value[E] {
	return e.entries
}

// Contains checks if a value exists in the enum.
func (e *Maker[T, E]) Contains(value E) bool {
	_, ok := e.valueMap[value]
	return ok
}

// ContainsName checks if a name exists in the enum.
func (e *Maker[T, E]) ContainsName(name string) bool {
	_, ok := e.nameMap[name]
	return ok
}

// NameMap returns a map of name-to-value pairs.
func (e *Maker[T, E]) NameMap() map[string]E {
	return e.nameMap
}
