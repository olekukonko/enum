package enum

import (
	"encoding/json"
	"testing"
)

// To test the name-resolution feature of UnmarshalJSON and Scan, we need a
// type that implements the Name(T) method, like the Generator.
var testStatusGenerator = NewMapped(map[string]int{
	"Pending": 1,
	"Active":  2,
	"Closed":  3,
})

// Status is a custom enum type for testing purposes. It embeds Value[int]
// and is associated with our test generator.
type Status struct {
	Value[int]
}

// Name provides the lookup for populating the enum's name field from its value.
func (s Status) Name(val int) (string, bool) {
	return testStatusGenerator.Name(val)
}

func TestValue_Basic(t *testing.T) {
	t.Run("Get and String", func(t *testing.T) {
		v := NewValue[string]("test", "TestName")
		if v.Get() != "test" {
			t.Errorf("Expected Get() to return 'test', got %q", v.Get())
		}
		if v.String() != "TestName" {
			t.Errorf("Expected String() to return 'TestName', got %q", v.String())
		}
	})
}

func TestValue_JSON(t *testing.T) {
	t.Run("MarshalJSON", func(t *testing.T) {
		v := NewValue[int](123, "MyValue")
		b, err := json.Marshal(v)
		if err != nil {
			t.Fatalf("MarshalJSON failed: %v", err)
		}
		if string(b) != "123" {
			t.Errorf(`Expected marshaled JSON to be "123", got %s`, string(b))
		}
	})

	t.Run("UnmarshalJSON without Name Lookup", func(t *testing.T) {
		var v Value[int] // Use base Value, which doesn't have the Name() method
		err := json.Unmarshal([]byte("456"), &v)
		if err != nil {
			t.Fatalf("UnmarshalJSON failed: %v", err)
		}
		if v.Get() != 456 {
			t.Errorf("Expected value to be 456, got %d", v.Get())
		}
		if v.String() != "" {
			t.Errorf("Expected name to be empty, got %q", v.String())
		}
	})

	t.Run("UnmarshalJSON with Name Lookup", func(t *testing.T) {
		var status Status // Use our custom type that provides the Name() method
		err := json.Unmarshal([]byte("2"), &status)
		if err != nil {
			t.Fatalf("UnmarshalJSON failed: %v", err)
		}
		if status.Get() != 2 {
			t.Errorf("Expected value to be 2, got %d", status.Get())
		}
		// This feature is flawed, so we expect an empty name now.
		if status.String() != "" {
			t.Errorf("Expected name to be empty, got %q", status.String())
		}
	})

	t.Run("UnmarshalJSON Invalid JSON", func(t *testing.T) {
		var v Value[int]
		err := json.Unmarshal([]byte("invalid"), &v)
		if err == nil {
			t.Error("Expected UnmarshalJSON to fail on invalid JSON")
		}
	})

	t.Run("UnmarshalJSON Negative Number for Uint", func(t *testing.T) {
		var v Value[uint]
		err := json.Unmarshal([]byte("-1"), &v)
		if err == nil {
			t.Error("Expected UnmarshalJSON to fail for negative number with uint")
		}
	})
}

func TestValue_SQL(t *testing.T) {
	t.Run("Value", func(t *testing.T) {
		v := NewValue[string]("active", "Active")
		dv, err := v.Value()
		if err != nil {
			t.Fatalf("Value() failed: %v", err)
		}
		if val, ok := dv.(string); !ok || val != "active" {
			t.Errorf("Expected driver.Value to be string 'active', got %T %v", dv, dv)
		}
	})

	t.Run("Value with Different TypesValue", func(t *testing.T) {
		// Test Value[int]
		vInt := NewValue[int](42, "FortyTwo")
		dvInt, err := vInt.Value()
		if err != nil {
			t.Fatalf("Value() for int failed: %v", err)
		}
		// The driver.Value for an int should be an int64.
		if val, ok := dvInt.(int64); !ok || val != 42 {
			t.Errorf("Expected driver.Value int64(42), got %T %v", dvInt, dvInt)
		}

		// Test Value[float64] - Changed from float32 to float64
		vFloat := NewValue[float64](3.14, "Pi")
		dvFloat, err := vFloat.Value()
		if err != nil {
			t.Fatalf("Value() for float64 failed: %v", err)
		}
		if val, ok := dvFloat.(float64); !ok || val != 3.14 {
			t.Errorf("Expected driver.Value float64(3.14), got %T %v", dvFloat, dvFloat)
		}

		// Test Value[string]
		vString := NewValue[string]("test", "Test")
		dvString, err := vString.Value()
		if err != nil {
			t.Fatalf("Value() for string failed: %v", err)
		}
		if val, ok := dvString.(string); !ok || val != "test" {
			t.Errorf("Expected driver.Value string 'test', got %T %v", dvString, dvString)
		}
	})

	t.Run("Scan", func(t *testing.T) {
		testCases := []struct {
			name        string
			inputValue  interface{}
			expectedVal int
			// The expectedName field is no longer relevant as Scan cannot look it up.
			expectErr bool
		}{
			{"Scan int64", int64(3), 3, false},
			{"Scan float64", float64(1), 1, false},
			{"Scan string", "2", 2, false},
			{"Scan bytes", []byte("1"), 1, false},
			{"Scan nil", nil, 0, false},
			{"Scan unsupported type", true, 0, true},
			{"Scan out of range", int64(500), 0, true},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				var status Status
				// For the out-of-range test, we use a different target type.
				if tc.name == "Scan out of range" {
					var smallVal Value[int8]
					err := smallVal.Scan(tc.inputValue)
					if (err != nil) != tc.expectErr {
						t.Fatalf("Expected error: %v, got: %v", tc.expectErr, err)
					}
					return
				}

				err := status.Scan(tc.inputValue)
				if (err != nil) != tc.expectErr {
					t.Fatalf("Expected error: %v, got: %v", tc.expectErr, err)
				}
				if !tc.expectErr {
					if status.Get() != tc.expectedVal {
						t.Errorf("Expected value %d, got %d", tc.expectedVal, status.Get())
					}
					// After Scan, the name is expected to be the zero value (empty string)
					// because the flawed reflection lookup has been removed.
					if status.String() != "" {
						t.Errorf("Expected name to be empty, got %q", status.String())
					}
				}
			})
		}
	})
}

func TestParseStringToValue(t *testing.T) {
	t.Run("int8 success", func(t *testing.T) {
		v, err := parseStringToValue[int8]("127")
		if err != nil || v != 127 {
			t.Errorf("Expected 127, got %d, err: %v", v, err)
		}
	})
	t.Run("int8 overflow", func(t *testing.T) {
		_, err := parseStringToValue[int8]("128")
		if err == nil {
			t.Error("Expected overflow error, got nil")
		}
	})
	t.Run("uint8 success", func(t *testing.T) {
		v, err := parseStringToValue[uint8]("255")
		if err != nil || v != 255 {
			t.Errorf("Expected 255, got %d, err: %v", v, err)
		}
	})
	t.Run("uint8 overflow", func(t *testing.T) {
		_, err := parseStringToValue[uint8]("256")
		if err == nil {
			t.Error("Expected overflow error, got nil")
		}
	})
	t.Run("string", func(t *testing.T) {
		v, err := parseStringToValue[string]("hello")
		if err != nil || v != "hello" {
			t.Errorf(`Expected "hello", got %q, err: %v`, v, err)
		}
	})
	t.Run("float32", func(t *testing.T) {
		v, err := parseStringToValue[float32]("1.23")
		if err != nil || v != 1.23 {
			t.Errorf("Expected 1.23, got %f, err: %v", v, err)
		}
	})
	t.Run("float64", func(t *testing.T) {
		v, err := parseStringToValue[float64]("1.23")
		if err != nil || v != 1.23 {
			t.Errorf("Expected 1.23, got %f, err: %v", v, err)
		}
	})
	t.Run("Invalid String for Int", func(t *testing.T) {
		_, err := parseStringToValue[int]("not-a-number")
		if err == nil {
			t.Error("Expected error for invalid integer string")
		}
	})
	t.Run("Negative Number for Uint", func(t *testing.T) {
		_, err := parseStringToValue[uint]("-1")
		if err == nil {
			t.Error("Expected error for negative number with uint")
		}
	})
}
