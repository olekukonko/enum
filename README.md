# 🎭 enum - The Type-Safe Enumeration Toolkit for Go

Go's `iota` is great for simple enumerations, but it has limitations. This package solves those problems:

- **Automatic string representation** - No need for manual `String()` methods
- **Value validation** - Easily check if a value is valid
- **Database integration** - Works seamlessly with SQL databases
- **JSON support** - Automatic marshaling/unmarshaling
- **Flexible values** - Not limited to sequential integers
- **Type safety** - Generics prevent mixing different enum types


## Real-World Comparison

### Traditional iota Approach
```go
package main

import "fmt"

// Without enum
const (
    StatusPending = iota  // 0
    StatusActive          // 1  
    StatusCompleted       // 2
    // Need to manually handle:
    // - String conversion
    // - Validation
    // - JSON/Database serialization
)

func main() {
    fmt.Println(StatusPending) // 0 (just a number)
    // No built-in way to get the name "Pending"
}
```

### With enum Package
```go
package main

import (
    "fmt"
    "github.com/olekukonko/enum"
)

// Create enum registry
var status = enum.NewBasic()

// Define values
var (
    pending   = status.Add("Pending")   // Value 0
    active    = status.Add("Active")    // Value 1
    completed = status.Add("Completed") // Value 2
)

func main() {
    fmt.Println(pending)         // "Pending" (auto String())
    fmt.Println(pending.Get())   // 0
    fmt.Println(pending.String())// "Pending"
    
    // Built-in validation
    if err := status.Validate(1); err == nil {
        fmt.Println("Valid status")
    }
    
    // Database/JSON ready
    jsonData, _ := pending.MarshalJSON()
    fmt.Println(string(jsonData)) // "0"
}
```

## Key Benefits

1. **Automatic String Conversion**
   ```go
   fmt.Println(pending) // "Pending" (no manual String() needed)
   ```

2. **Bidirectional Lookups**
   ```go
   // Value to name
   name, _ := status.meta.Name(1) // "Active"
   
   // Name to value  
   val, _ := status.meta.Get("Pending") // 0
   ```

3. **Built-in Validation**
   ```go
   err := status.Validate(99) // Error: invalid value
   ```

4. **Database/JSON Integration**
   ```go
   // JSON
   type Task struct {
       Status enum.Basic `json:"status"`
   }
   
   // Database
   var t Task
   db.QueryRow("SELECT status FROM tasks").Scan(&t.Status)
   ```

5. **Non-Sequential Values**
   ```go
   http := enum.NewBasic()
   ok := http.Add("OK").With(200)
   notFound := http.Add("NotFound").With(404)
   ```

## Feature Comparison Table

| Feature                     | Standard iota | enum package |
|-----------------------------|---------------|--------------|
| Automatic string conversion | ❌ Manual     | ✅ Automatic |
| Value validation            | ❌ Manual     | ✅ Built-in |
| Database support            | ❌ Manual     | ✅ Built-in |
| JSON support                | ❌ Manual     | ✅ Built-in |
| Non-sequential values       | ❌ Limited    | ✅ Supported |
| Thread-safe                 | ❌ N/A        | ✅ Yes |
| Bit flag support            | ✅ Possible   | ✅ Built-in |

## Getting Started

```bash
go get github.com/olekukonko/enum
```

```go
package main

import (
    "fmt"
    "github.com/olekukonko/enum"
)

func main() {
    colors := enum.NewBasic()
    red := colors.Add("Red")
    green := colors.Add("Green")
    
    fmt.Println(green) // "Green"
    fmt.Println(green.Get()) // 1
}
```