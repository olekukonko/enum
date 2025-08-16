package enum

// ValueTypes is a constraint that defines all supported underlying types for enums.
// It includes string, all standard integer types (signed and unsigned), and
// floating-point types (float32 and float64).
type ValueTypes interface {
	~string | ~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 |
		~float32 | ~float64
}

// MakeTypes is a type constraint for enum values used with Maker.
// It restricts the underlying type to integers (signed or unsigned), as Maker
// assigns sequential integer values to struct fields.
type MakeTypes interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64
}

// Entry defines the interface that all enum types must implement.
// It requires methods to retrieve the underlying value and its string representation.
type Entry[T comparable] interface {
	// Get returns the underlying value of the enum entry.
	Get() T
	// String returns the human-readable name of the enum entry.
	String() string
}
