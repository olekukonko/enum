package enum

import (
	"encoding/json"
	"testing"
)

func TestBasicEnum(t *testing.T) {
	t.Run("New registry", func(t *testing.T) {
		colors := NewBasic()
		if colors == nil {
			t.Error("NewBasic() returned nil")
		}
	})

	t.Run("Add values", func(t *testing.T) {
		status := NewBasic()
		pending := status.Add("Pending")
		active := status.Add("Active")

		if pending.String() != "Pending" {
			t.Errorf("Expected 'Pending', got %q", pending.String())
		}
		if active.String() != "Active" {
			t.Errorf("Expected 'Active', got %q", active.String())
		}
	})

	t.Run("Auto-increment values", func(t *testing.T) {
		colors := NewBasic()
		red := colors.Add("Red")
		blue := colors.Add("Blue")

		if red.Get() != 0 {
			t.Errorf("Expected Red value 0, got %d", red.Get())
		}
		if blue.Get() != 1 {
			t.Errorf("Expected Blue value 1, got %d", blue.Get())
		}
	})

	t.Run("With custom values", func(t *testing.T) {
		http := NewBasic()
		ok := http.Add("OK").With(200)
		notFound := http.Add("NotFound").With(404)

		if ok.Get() != 200 {
			t.Errorf("Expected OK value 200, got %d", ok.Get())
		}
		if notFound.Get() != 404 {
			t.Errorf("Expected NotFound value 404, got %d", notFound.Get())
		}
	})

	t.Run("Duplicate name panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic for duplicate name")
			}
		}()

		colors := NewBasic()
		colors.Add("Red")
		colors.Add("Red") // Should panic
	})

	t.Run("Duplicate value panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic for duplicate value")
			}
		}()

		http := NewBasic()
		http.Add("OK").With(200)
		http.Add("Success").With(200) // Should panic
	})

	t.Run("JSON marshaling", func(t *testing.T) {
		status := NewBasic()
		pending := status.Add("Pending")

		data, err := json.Marshal(pending)
		if err != nil {
			t.Fatalf("Marshal error: %v", err)
		}

		var result Basic
		result.meta = status.meta
		err = json.Unmarshal(data, &result)
		if err != nil {
			t.Fatalf("Unmarshal error: %v", err)
		}

		if result.String() != "Pending" {
			t.Errorf("Expected 'Pending' after roundtrip, got %q", result.String())
		}
		if result.Get() != 0 {
			t.Errorf("Expected value 0 after roundtrip, got %d", result.Get())
		}
	})

	t.Run("Database Value/Scan", func(t *testing.T) {
		status := NewBasic()
		active := status.Add("Active").With(1)

		// Test Value() for SQL
		val, err := active.Value()
		if err != nil {
			t.Fatalf("Value() error: %v", err)
		}
		if val.(int64) != 1 {
			t.Errorf("Expected Value() to return 1, got %v", val)
		}

		// Test Scan() from SQL
		var scanned Basic
		// FIX: Assign the meta registry before scanning.
		scanned.meta = status.meta
		err = scanned.Scan(int64(1))
		if err != nil {
			t.Fatalf("Scan() error: %v", err)
		}
		if scanned.String() != "Active" {
			t.Errorf("Expected scanned value 'Active', got %q", scanned.String())
		}
	})

	t.Run("Validate", func(t *testing.T) {
		colors := NewBasic()
		colors.Add("Red")
		valid := colors.Add("Green")
		// FIX: Assign the meta registry to the manually created invalid instance.
		invalid := Basic{value: 99, meta: colors.meta} // Not registered

		if err := valid.Validate(); err != nil {
			t.Errorf("Valid value returned error: %v", err)
		}
		if err := invalid.Validate(); err == nil {
			t.Error("Invalid value should return error")
		}
	})

	t.Run("Values list", func(t *testing.T) {
		colors := NewBasic()
		colors.Add("Red")
		colors.Add("Green")
		colors.Add("Blue")

		values := colors.Values()
		if len(values) != 3 {
			t.Fatalf("Expected 3 values, got %d", len(values))
		}
		if values[0].String() != "Red" || values[1].String() != "Green" || values[2].String() != "Blue" {
			t.Errorf("Values() returned incorrect order or content")
		}
	})
}
