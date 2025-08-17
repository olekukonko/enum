// Package enum provides a generic implementation of enumerated types (enums) in Go.
// The Generator type in this package enables flexible, thread-safe creation and management
// of enum values for types constrained by the TypesValue interface (strings, integers, or floats).
// Generators support sequential generation (e.g., incrementing numbers, alphabetical strings),
// custom increment logic, and static mappings for non-sequential enums.
//
// A Generator[T] maintains a collection of enum entries (Value[T]), mapping values to names
// and vice versa. It is thread-safe, using a read-write mutex to protect concurrent access.
// Specialized constructors like NewAlpha, NewNumeric, NewBitFlagGenerator, NewPrefixed,
// NewCyclic, and NewMapped provide convenient ways to create generators for common use cases.
//
// Example usage for a numeric enum:
//
//	g := enum.NewNumeric(1)
//	v1 := g.Next("One")   // Value[int]{value: 1, name: "One"}
//	v2 := g.Next("Two")   // Value[int]{value: 2, name: "Two"}
//	fmt.Println(g.Name(1)) // Output: One, true
//
// Example usage for a mapped enum:
//
//	m := map[string]int{"Small": 1, "Large": 100}
//	g := enum.NewMapped(m)
//	v, err := g.Parse("Small") // Value[int]{value: 1, name: "Small"}
//	fmt.Println(v, err)       // Output: {1 Small} <nil>
package enum

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
)

// Generator provides thread-safe generation and management of enum values for a given type T.
// It supports sequential generation with customizable increment logic or static mappings.
// The Generator maintains mappings of values to names and names to values, and provides
// methods for lookup, parsing, and validation. Use NewGenerator or specialized constructors
// (e.g., NewNumeric, NewAlpha) to create a Generator.
type Generator[T TypesValue] struct {
	mu          sync.RWMutex // Protects concurrent access to generator state.
	current     T            // Current value for the next enum entry.
	incrementer func(T) T    // Function to compute the next value in the sequence.
	values      []Value[T]   // Slice of all generated enum entries.
	valueMap    map[T]string // Maps values to their string names.
	nameMap     map[string]T // Maps names to their values.
}

// NewGenerator creates a new Generator for type T with optional configuration options.
// By default, it starts with the zero value of T and uses a default incrementer that adds 1
// for numeric types or increments alphabetically for strings (e.g., "A" -> "B", "Z" -> "AA").
// The generator is thread-safe for concurrent use.
//
// Options can be used to set the starting value or custom increment logic.
//
// Example:
//
//	g := NewGenerator[int](WithStart(10), WithIncrementer(func(x int) int { return x + 2 }))
//	v := g.Next("Ten") // Value[int]{value: 10, name: "Ten"}
//	fmt.Println(v.Get()) // Output: 10
func NewGenerator[T TypesValue](opts ...Option[T]) *Generator[T] {
	g := &Generator[T]{
		current:     *new(T),
		incrementer: defaultIncrementer[T],
		valueMap:    make(map[T]string),
		nameMap:     make(map[string]T),
	}
	for _, opt := range opts {
		opt(g)
	}
	return g
}

// Option is a function that configures a Generator[T].
type Option[T TypesValue] func(*Generator[T])

// WithStart sets the starting value for the Generator's sequence.
func WithStart[T TypesValue](start T) Option[T] {
	return func(g *Generator[T]) {
		g.current = start
	}
}

// WithIncrementer sets a custom increment function for the Generator.
// The function takes the current value and returns the next value in the sequence.
func WithIncrementer[T TypesValue](inc func(T) T) Option[T] {
	return func(g *Generator[T]) {
		g.incrementer = inc
	}
}

// NewAlpha creates a Generator for alphabetical string enums (e.g., "A", "B", ..., "Z", "AA").
// It starts at "A" and increments alphabetically using the default string incrementer.
// The generator is thread-safe.
//
// Example:
//
//	g := NewAlpha()
//	v1 := g.Next("First")  // Value[string]{value: "A", name: "First"}
//	v2 := g.Next("Second") // Value[string]{value: "B", name: "Second"}
func NewAlpha() *Generator[string] {
	return NewGenerator[string](WithStart[string]("A"))
}

// NewNumeric creates a Generator for any numeric type, starting at the specified value.
// It uses the default incrementer (adds 1) unless customized via options.
// The generator is thread-safe.
//
// Example:
//
//	g := NewNumeric(100)
//	v := g.Next("Hundred") // Value[int]{value: 100, name: "Hundred"}
func NewNumeric[T TypesValue](start T) *Generator[T] {
	return NewGenerator[T](WithStart(start))
}

// NewBitFlagGenerator creates a Generator for bit flag enums (e.g., 1, 2, 4, 8, ...).
// It starts at the specified value and shifts left by 1 for each new value (e.g., x << 1).
// The underlying type T must be an integer type; non-integer types will not increment.
// The generator is thread-safe.
//
// Example:
//
//	g := NewBitFlagGenerator(1)
//	v1 := g.Next("Flag1") // Value[int]{value: 1, name: "Flag1"}
//	v2 := g.Next("Flag2") // Value[int]{value: 2, name: "Flag2"}
func NewBitFlagGenerator[T TypesValue](start T) *Generator[T] {
	return NewGenerator[T](
		WithStart(start),
		WithIncrementer(func(x T) T {
			switch v := any(x).(type) {
			case int:
				return any(v << 1).(T)
			case int8:
				return any(v << 1).(T)
			case int16:
				return any(v << 1).(T)
			case int32:
				return any(v << 1).(T)
			case int64:
				return any(v << 1).(T)
			case uint:
				return any(v << 1).(T)
			case uint8:
				return any(v << 1).(T)
			case uint16:
				return any(v << 1).(T)
			case uint32:
				return any(v << 1).(T)
			case uint64:
				return any(v << 1).(T)
			default:
				return x
			}
		}),
	)
}

// NewPrefixed creates a Generator for string enums with a fixed prefix and incrementing number
// (e.g., "dog1", "dog2", ...). It starts with the given prefix and number.
// The generator is thread-safe.
//
// Example:
//
//	g := NewPrefixed("dog", 1)
//	v1 := g.Next("Dog1") // Value[string]{value: "dog1", name: "Dog1"}
//	v2 := g.Next("Dog2") // Value[string]{value: "dog2", name: "Dog2"}
func NewPrefixed(prefix string, start int) *Generator[string] {
	initialValue := fmt.Sprintf("%s%d", prefix, start)
	incrementer := func(s string) string {
		numStr := strings.TrimPrefix(s, prefix)
		num, err := strconv.Atoi(numStr)
		if err != nil {
			return s // Fail gracefully
		}
		return fmt.Sprintf("%s%d", prefix, num+1)
	}
	return NewGenerator[string](
		WithStart(initialValue),
		WithIncrementer(incrementer),
	)
}

// NewCyclic creates a Generator for integers that cycle from 0 to modulus-1.
// If modulus <= 0, it defaults to 1 to avoid division by zero.
// The generator is thread-safe.
//
// Example:
//
//	g := NewCyclic(3)
//	v1 := g.Next("Zero")  // Value[int]{value: 0, name: "Zero"}
//	v2 := g.Next("One")   // Value[int]{value: 1, name: "One"}
//	v3 := g.Next("Two")   // Value[int]{value: 2, name: "Two"}
//	v4 := g.Next("Zero2") // Value[int]{value: 0, name: "Zero2"}
func NewCyclic(modulus int) *Generator[int] {
	if modulus <= 0 {
		modulus = 1 // Avoid division by zero
	}
	incrementer := func(i int) int {
		return (i + 1) % modulus
	}
	return NewGenerator[int](
		WithStart(0),
		WithIncrementer(incrementer),
	)
}

// NewMapped creates a Generator pre-populated with a static map of names to values.
// It is designed for non-sequential enums where values are defined upfront.
// The Generator supports lookups (Name, Get, Parse) but panics if Next is called,
// as it does not support sequential generation. The generator is thread-safe.
//
// Example:
//
//	m := map[string]int{"Small": 1, "Large": 100}
//	g := NewMapped(m)
//	v, err := g.Parse("Small") // Value[int]{value: 1, name: "Small"}
func NewMapped[T TypesValue](nameToValueMap map[string]T) *Generator[T] {
	g := &Generator[T]{
		incrementer: nil, // Prevent Next() usage
		valueMap:    make(map[T]string, len(nameToValueMap)),
		nameMap:     make(map[string]T, len(nameToValueMap)),
		values:      make([]Value[T], 0, len(nameToValueMap)),
	}
	for name, value := range nameToValueMap {
		entry := NewValue(value, name)
		g.values = append(g.values, entry)
		g.nameMap[name] = value
		g.valueMap[value] = name
	}
	return g
}

// Next generates the next enum value in the sequence with the given name.
// It updates the internal state (valueMap, nameMap, values) and advances the current value
// using the configured incrementer. It panics if called on a Generator created with NewMapped.
// The method is thread-safe, using a write lock to protect state modifications.
//
// Returns a Value[T] containing the generated value and name.
func (g *Generator[T]) Next(name string) Value[T] {
	if g.incrementer == nil {
		panic("enum: cannot call Next() on a Generator created with NewMapped")
	}
	g.mu.Lock()
	defer g.mu.Unlock()

	// FIX: Check for duplicate names before adding.
	if _, exists := g.nameMap[name]; exists {
		panic(fmt.Sprintf("enum: name %q already exists", name))
	}

	val := g.current
	g.current = g.incrementer(g.current)
	entry := NewValue(val, name)
	g.values = append(g.values, entry)
	g.valueMap[val] = name
	g.nameMap[name] = val
	return entry
}

// Name returns the name associated with a given value, if it exists.
// It is thread-safe, using a read lock for access.
//
// Returns the name and true if the value exists, or an empty string and false otherwise.
func (g *Generator[T]) Name(value T) (string, bool) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	name, ok := g.valueMap[value]
	return name, ok
}

// Get returns the value associated with a given name, if it exists.
// It is thread-safe, using a read lock for access.
//
// Returns the value and true if the name exists, or the zero value of T and false otherwise.
func (g *Generator[T]) Get(name string) (T, bool) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	val, ok := g.nameMap[name]
	return val, ok
}

// Values returns a copy of all generated enum entries as a slice of Value[T].
// It is thread-safe, using a read lock and returning a copy to prevent external modification.
func (g *Generator[T]) Values() []Value[T] {
	g.mu.RLock()
	defer g.mu.RUnlock()
	valsCopy := make([]Value[T], len(g.values))
	copy(valsCopy, g.values)
	return valsCopy
}

// ValueMap returns a copy of the map of values to names.
// It is thread-safe, using a read lock and returning a copy to prevent external modification.
func (g *Generator[T]) ValueMap() map[T]string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	mapCopy := make(map[T]string, len(g.valueMap))
	for k, v := range g.valueMap {
		mapCopy[k] = v
	}
	return mapCopy
}

// NameMap returns a copy of the map of names to values.
// It is thread-safe, using a read lock and returning a copy to prevent external modification.
func (g *Generator[T]) NameMap() map[string]T {
	g.mu.RLock()
	defer g.mu.RUnlock()
	mapCopy := make(map[string]T, len(g.nameMap))
	for k, v := range g.nameMap {
		mapCopy[k] = v
	}
	return mapCopy
}

// Contains checks if a value exists in the generated enum set.
// It is thread-safe, using a read lock for access.
func (g *Generator[T]) Contains(value T) bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	_, ok := g.valueMap[value]
	return ok
}

// Names returns a slice of all enum names.
// It is thread-safe, using a read lock for access.
func (g *Generator[T]) Names() []string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	names := make([]string, len(g.values))
	for i, val := range g.values {
		names[i] = val.String()
	}
	return names
}

// MarshalJSON implements json.Marshaler, serializing the Generator's value-to-name map.
// It is thread-safe, using a read lock for access.
func (g *Generator[T]) MarshalJSON() ([]byte, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return json.Marshal(g.valueMap)
}

// UnmarshalJSON implements json.Unmarshaler, deserializing a JSON object into the
// Generator's value-to-name map. It clears existing state and populates valueMap, nameMap,
// and values. It is thread-safe, using a write lock for state modification.
//
// Note: This sets incrementer to nil, making the Generator behave like one created with NewMapped.
func (g *Generator[T]) UnmarshalJSON(data []byte) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.valueMap = make(map[T]string)
	g.nameMap = make(map[string]T)
	g.values = nil
	g.incrementer = nil
	return json.Unmarshal(data, &g.valueMap)
}

// Parse attempts to parse a string into an enum value.
// It first checks if the string matches a known name in nameMap. If not, it attempts to parse
// the string as a value literal using parseStringToValue and checks if the parsed value exists
// in valueMap. It is thread-safe, using a read lock for access.
//
// Returns a Value[T] if successful, or an error if no matching name or value is found.
func (g *Generator[T]) Parse(s string) (Value[T], error) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	if val, ok := g.nameMap[s]; ok {
		return NewValue(val, s), nil
	}
	parsedVal, err := parseStringToValue[T](s)
	if err != nil {
		return Value[T]{}, err
	}
	if name, ok := g.valueMap[parsedVal]; ok {
		return NewValue(parsedVal, name), nil
	}
	return Value[T]{}, fmt.Errorf("no matching enum value for %q", s)
}

// MustParse is like Parse but panics on error.
// It is thread-safe, using a read lock for access.
func (g *Generator[T]) MustParse(s string) Value[T] {
	val, err := g.Parse(s)
	if err != nil {
		panic(err)
	}
	return val
}

// Validate checks if a value is valid for this enum set.
// It is thread-safe, using a read lock for access.
//
// Returns nil if the value exists, or an error otherwise.
func (g *Generator[T]) Validate(value T) error {
	if !g.Contains(value) {
		return fmt.Errorf("invalid enum value: %v", value)
	}
	return nil
}

// ValidateName checks if a name is valid for this enum set.
// It is thread-safe, using a read lock for access.
//
// Returns nil if the name exists, or an error otherwise.
func (g *Generator[T]) ValidateName(name string) error {
	g.mu.RLock()
	defer g.mu.RUnlock()
	if _, ok := g.nameMap[name]; !ok {
		return fmt.Errorf("invalid enum name: %q", name)
	}
	return nil
}

// ValidValues returns a slice of all valid values in the enum set.
// It is thread-safe, using a read lock for access.
func (g *Generator[T]) ValidValues() []T {
	g.mu.RLock()
	defer g.mu.RUnlock()
	values := make([]T, 0, len(g.valueMap))
	for v := range g.valueMap {
		values = append(values, v)
	}
	return values
}

// defaultIncrementer provides default increment logic for supported types.
// For integers and floats, it adds 1. For strings, it increments alphabetically
// (e.g., "A" -> "B", "Z" -> "AA", "AZ" -> "BA"). It is used when no custom
// incrementer is provided to NewGenerator.
func defaultIncrementer[T TypesValue](x T) T {
	switch v := any(x).(type) {
	case string:
		runes := []rune(v)
		if len(runes) == 0 {
			return any("A").(T)
		}
		for i := len(runes) - 1; i >= 0; i-- {
			if runes[i] < 'Z' {
				runes[i]++
				return any(string(runes)).(T)
			}
			runes[i] = 'A' // Carry over
		}
		return any("A" + string(runes)).(T)
	case int:
		return any(v + 1).(T)
	case int8:
		return any(v + 1).(T)
	case int16:
		return any(v + 1).(T)
	case int32:
		return any(v + 1).(T)
	case int64:
		return any(v + 1).(T)
	case uint:
		return any(v + 1).(T)
	case uint8:
		return any(v + 1).(T)
	case uint16:
		return any(v + 1).(T)
	case uint32:
		return any(v + 1).(T)
	case uint64:
		return any(v + 1).(T)
	case float32:
		return any(v + 1).(T)
	case float64:
		return any(v + 1).(T)
	}
	return x
}
