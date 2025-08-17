# enum - A Powerful Enumeration Library for Go

The `enum` package provides a comprehensive solution for working with enumerations in Go, offering features beyond what's possible with standard `iota` while maintaining type safety and performance.

## Why Use This Package?

Go's `iota` is great for simple enumerations, but it has limitations. This package solves those problems:

- **Automatic string representation** - No need for manual `String()` methods
- **Value validation** - Easily check if a value is valid
- **Database integration** - Works seamlessly with SQL databases
- **JSON support** - Automatic marshaling/unmarshaling
- **Flexible values** - Not limited to sequential integers
- **Type safety** - Generics prevent mixing different enum types

## Feature Comparison

| Feature                     | Standard iota | enum package |
|-----------------------------|---------------|--------------|
| Automatic string conversion | ❌ Manual     | ✅ Automatic |
| Non-sequential values       | ❌ Limited    | ✅ Supported |
| Database integration        | ❌ Manual     | ✅ Built-in |
| JSON support                | ❌ Manual     | ✅ Built-in |
| Value validation            | ❌ Manual     | ✅ Built-in |
| Bit flags support           | ✅ Possible   | ✅ Built-in |
| Custom value types          | ❌ Limited    | ✅ Any type |
| Reverse lookup (name→value) | ❌ Manual     | ✅ Built-in |
| Thread-safe generation      | ❌ N/A        | ✅ Supported |
| Cyclic values               | ❌ Manual     | ✅ Built-in |

## Installation

```bash
go get github.com/olekukonko/enum
```

## Basic Usage

### Simple Enumeration

```go
package main

import (
	"fmt"
	"github.com/olekukonko/enum"
)

func main() {
	// Create a new enum registry
	status := enum.NewBasic()
	
	// Add values (auto-incremented from 0)
	pending := status.Add("Pending")
	active := status.Add("Active")
	completed := status.Add("Completed")

	fmt.Println(pending.Int(), pending.String())  // 0 "Pending"
	fmt.Println(active.Int(), active.String())    // 1 "Active"
	fmt.Println(completed.Int(), completed.String()) // 2 "Completed"
}
```

### HTTP Status Codes Example

```go
func httpStatusExample() {
	http := enum.NewBasic()
	
	// Add with custom values
	ok := http.Add("OK").With(200)
	notFound := http.Add("NotFound").With(404)
	serverError := http.Add("ServerError").With(500)

	// JSON marshaling
	data, _ := json.Marshal(ok)
	fmt.Println(string(data)) // "200"

	// Database integration
	var fromDB enum.Basic
	fromDB.meta = http.meta
	fromDB.Scan(404)
	fmt.Println(fromDB.String()) // "NotFound"
}
```

### IP Data Example

```go
func ipProtocolExample() {
	// Create enum for IP protocols
	protocols := enum.NewBasic()
	
	tcp := protocols.Add("TCP").With(6)
	udp := protocols.Add("UDP").With(17)
	icmp := protocols.Add("ICMP").With(1)

	// Parse protocol number
	parsed, err := protocols.meta.Parse("17")
	if err != nil {
		panic(err)
	}
	fmt.Println(parsed.String()) // "UDP"

	// Validate input
	err = protocols.meta.Validate(22) // SSH (not in our enum)
	if err != nil {
		fmt.Println("Invalid protocol:", err)
	}
}
```

### Bit Flags Example

```go
func permissionExample() {
	// Create bit flag enum
	perms := enum.NewBitFlagGenerator[uint](1)
	
	read := perms.Next("Read")
	write := perms.Next("Write")
	execute := perms.Next("Execute")

	// Combine flags
	myPerms := read.Get() | write.Get()
	fmt.Printf("My permissions: %b\n", myPerms) // 011

	// Check permissions
	if myPerms&read.Get() != 0 {
		fmt.Println("Has read permission")
	}
}
```

## Advanced Features

### Using with Structs

```go
type Config struct {
	LogLevel enum.Basic
}

func structExample() {
	levels := enum.NewBasic()
	debug := levels.Add("Debug")
	info := levels.Add("Info")
	warn := levels.Add("Warn")

	cfg := Config{LogLevel: info}
	
	data, _ := json.Marshal(cfg)
	// {"LogLevel":1}
	
	var cfg2 Config
	cfg2.LogLevel.meta = levels.meta
	json.Unmarshal(data, &cfg2)
	fmt.Println(cfg2.LogLevel.String()) // "Info"
}
```

### Custom Value Types

```go
func customTypeExample() {
	// Create enum with string values
	g := enum.NewGenerator[string]()
	
	red := g.Next("Red") // value "A"
	green := g.Next("Green") // value "B"
	blue := g.Next("Blue") // value "C"

	fmt.Println(red.Get()) // "A"
	fmt.Println(green.String()) // "Green"
}
```

## Performance Considerations

The `enum` package is designed for performance:

1. **Basic enums** have minimal overhead over raw integers
2. **Thread-safe operations** use efficient locking
3. **Zero allocations** for most operations
4. **Fast lookups** using maps

For most use cases, the performance difference compared to `iota` is negligible.

## When to Use

Consider this package when you need:

- String representations of enum values
- Validation of enum values
- Database persistence
- JSON serialization
- Non-sequential values
- More flexibility than iota provides

For simple sequential enumerations where you don't need these features, `iota` may still be appropriate.

## Contributing

Contributions are welcome! Please open an issue or submit a pull request.

## License

MIT