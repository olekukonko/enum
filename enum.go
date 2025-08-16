// Package enum provides two flexible mechanisms for creating and managing
// enumerations in Go: a programmatic Generator and a reflection-based Maker.
//
// The Generator is ideal for creating dynamic enums, bit flags, or sequences
// with custom incrementing logic. It is type-safe and highly configurable.
//
// The Maker provides a declarative approach, creating enums from the fields
// of a struct at runtime. This can be convenient for simple, static enums
// but comes with the performance and type-safety trade-offs inherent to reflection.
package enum

import (
	"fmt"
	"strconv"
	"strings"
)

// Types defines all supported enum value types for the Generator.
type Types interface {
	~string | ~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 |
		~float32 | ~float64
}

// Entry defines the interface that all enum types should implement.
type Entry[T comparable] interface {
	String() string
	Value() T
}

// Value provides a reusable base for enum types.
type Value[T comparable] struct {
	value T
	name  string
}

// Value returns the underlying value of the enum entry.
func (e Value[T]) Value() T { return e.value }

// String returns the name of the enum entry.
func (e Value[T]) String() string { return e.name }

// New creates a new enum value.
func New[T comparable](value T, name string) Value[T] {
	return Value[T]{value: value, name: name}
}

// Generator provides flexible enum generation.
type Generator[T Types] struct {
	current     T
	incrementer func(T) T
	values      []Value[T]
	valueMap    map[T]string
	nameMap     map[string]T
}

// NewGenerator creates a new enum generator with optional configurations.
// By default, it starts with the zero value for the given type and increments by 1.
func NewGenerator[T Types](opts ...Option[T]) *Generator[T] {
	g := &Generator[T]{
		current:     *new(T), // zero value
		incrementer: defaultIncrementer[T],
		valueMap:    make(map[T]string),
		nameMap:     make(map[string]T),
	}

	for _, opt := range opts {
		opt(g)
	}
	return g
}

// Option is a function that configures a Generator.
type Option[T Types] func(*Generator[T])

// defaultIncrementer provides a sensible default increment logic for supported types.
// Integers and floats are incremented by 1.
// Strings are incremented alphabetically (A -> B, Z -> AA, AZ -> BA).
func defaultIncrementer[T Types](x T) T {
	switch v := any(x).(type) {
	case string:
		// Robust string incrementer (e.g., spreadsheet columns)
		runes := []rune(v)
		if len(runes) == 0 {
			return any("A").(T)
		}
		// Iterate from right to left
		for i := len(runes) - 1; i >= 0; i-- {
			if runes[i] < 'Z' {
				runes[i]++
				return any(string(runes)).(T)
			}
			// Carry over
			runes[i] = 'A'
		}
		// If we've carried over the whole way (e.g., ZZZ -> AAAA)
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
	// Return unchanged if type is not handled
	return x
}

// WithStart sets the starting value for the enum sequence.
func WithStart[T Types](start T) Option[T] {
	return func(g *Generator[T]) {
		g.current = start
	}
}

// WithIncrementer sets a custom increment function for the generator.
func WithIncrementer[T Types](inc func(T) T) Option[T] {
	return func(g *Generator[T]) {
		g.incrementer = inc
	}
}

// Next generates the next enum value in the sequence with the given name.
func (g *Generator[T]) Next(name string) Value[T] {
	val := g.current
	g.current = g.incrementer(g.current)
	entry := New(val, name)
	g.values = append(g.values, entry)
	g.valueMap[val] = name
	g.nameMap[name] = val
	return entry
}

// Name returns the name for a given value, if it exists.
func (g *Generator[T]) Name(value T) (string, bool) {
	name, ok := g.valueMap[value]
	return name, ok
}

// Value returns the value for a given name, if it exists.
func (g *Generator[T]) Value(name string) (T, bool) {
	val, ok := g.nameMap[name]
	return val, ok
}

// Values returns all generated values as a slice of Value[T].
func (g *Generator[T]) Values() []Value[T] {
	return g.values
}

// ValueMap returns a map of value-to-name pairs.
func (g *Generator[T]) ValueMap() map[T]string {
	return g.valueMap
}

// NameMap returns a map of name-to-value pairs.
func (g *Generator[T]) NameMap() map[string]T {
	return g.nameMap
}

// Contains checks if a value exists in the generated enum set.
func (g *Generator[T]) Contains(value T) bool {
	_, ok := g.valueMap[value]
	return ok
}

// Names returns all enum names as a slice of strings.
func (g *Generator[T]) Names() []string {
	names := make([]string, len(g.values))
	for i, val := range g.values {
		names[i] = val.String()
	}
	return names
}

// NewAlpha creates a generator for alphabetical string enums (A, B, C... Z, AA, etc.).
func NewAlpha() *Generator[string] {
	// This now uses the improved default incrementer for strings.
	return NewGenerator[string](WithStart[string]("A"))
}

// NewNumeric creates a generator for any numeric type, starting at the specified value.
func NewNumeric[T Types](start T) *Generator[T] {
	return NewGenerator[T](WithStart(start))
}

// NewBitFlagGenerator creates a generator for bit flags (1, 2, 4, 8...).
// The underlying type must be an integer.
func NewBitFlagGenerator[T Types](start T) *Generator[T] {
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
				// Return unchanged for non-integer types
				return x
			}
		}),
	)
}

// NewPrefixed creates a generator that increments a number
// at the end of a string prefix. e.g., "dog1", "dog2".
func NewPrefixed(prefix string, start int) *Generator[string] {
	// The starting value for the sequence.
	initialValue := fmt.Sprintf("%s%d", prefix, start)

	incrementer := func(s string) string {
		// Remove the prefix to isolate the number.
		numStr := strings.TrimPrefix(s, prefix)

		// Convert the numeric part to an integer.
		num, err := strconv.Atoi(numStr)
		if err != nil {
			// If it fails for some reason, just return the original string.
			return s
		}

		// Increment the number and re-attach the prefix.
		return fmt.Sprintf("%s%d", prefix, num+1)
	}

	return NewGenerator[string](
		WithStart(initialValue),
		WithIncrementer(incrementer),
	)
}

// NewCyclic creates a generator that cycles through numbers
// from 0 to modulus-1.
func NewCyclic(modulus int) *Generator[int] {
	if modulus <= 0 {
		modulus = 1 // Avoid division by zero
	}

	incrementer := func(i int) int {
		// Use the modulus operator to wrap the value
		return (i + 1) % modulus
	}

	return NewGenerator[int](
		// Start at 0 and provide the custom incrementer
		WithStart(0),
		WithIncrementer(incrementer),
	)
}

// NewMapped creates a generator that is pre-populated with values
// from a map of names to values (map[string]T).
// This is ideal for enums where the values are non-sequential or are defined
// statically. The returned Generator can be used for lookups (Name, Value),
// but calling Next() on it will have undefined behavior as it is not
// configured for sequential generation.
func NewMapped[T Types](nameToValueMap map[string]T) *Generator[T] {
	g := &Generator[T]{
		// The incrementer and current value are not relevant for a static map.
		incrementer: func(t T) T { return t },
		valueMap:    make(map[T]string),
		nameMap:     make(map[string]T),
		values:      make([]Value[T], 0, len(nameToValueMap)),
	}

	for name, value := range nameToValueMap {
		// Create the enum entry.
		entry := New(value, name)

		// Populate the internal structures.
		g.values = append(g.values, entry)
		g.nameMap[name] = value
		g.valueMap[value] = name
	}

	return g
}
