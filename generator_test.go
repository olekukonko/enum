package enum

import (
	"encoding/json"
	"reflect"
	"sync"
	"testing"
)

func TestGenerator(t *testing.T) {
	t.Run("NumericSequence", func(t *testing.T) {
		g := NewGenerator[int](WithStart(10))
		val1 := g.Next("Ten")
		val2 := g.Next("Eleven")
		val3 := g.Next("Twelve")

		if val1.Get() != 10 || val1.String() != "Ten" {
			t.Errorf("Expected 10/Ten, got %d/%s", val1.Get(), val1.String())
		}
		if val2.Get() != 11 || val2.String() != "Eleven" {
			t.Errorf("Expected 11/Eleven, got %d/%s", val2.Get(), val2.String())
		}
		if val3.Get() != 12 || val3.String() != "Twelve" {
			t.Errorf("Expected 12/Twelve, got %d/%s", val3.Get(), val3.String())
		}

		if name, ok := g.Name(11); !ok || name != "Eleven" {
			t.Errorf("Expected Eleven for value 11, got %s", name)
		}

		if val, ok := g.Get("Twelve"); !ok || val != 12 {
			t.Errorf("Expected 12 for name Twelve, got %d", val)
		}

		if !g.Contains(11) {
			t.Error("Expected generator to contain value 11")
		}

		names := g.Names()
		expectedNames := []string{"Ten", "Eleven", "Twelve"}
		if !reflect.DeepEqual(names, expectedNames) {
			t.Errorf("Expected names %v, got %v", expectedNames, names)
		}

		validVals := g.ValidValues()
		if len(validVals) != 3 {
			t.Errorf("Expected 3 valid values, got %d", len(validVals))
		}
	})

	t.Run("StringSequenceWithCarry", func(t *testing.T) {
		g := NewGenerator[string](WithStart("Y"))
		g.Next("Yankee")
		valZ := g.Next("Zulu")
		valAA := g.Next("AlphaAlpha")

		if valZ.Get() != "Z" {
			t.Errorf("Expected Z, got %s", valZ.Get())
		}
		if valAA.Get() != "AA" {
			t.Errorf("Expected AA after Z, got %s", valAA.Get())
		}
	})

	t.Run("BitFlags", func(t *testing.T) {
		g := NewBitFlagGenerator[uint](1)
		val1 := g.Next("Read")
		val2 := g.Next("Write")
		val3 := g.Next("Execute")

		if val1.Get() != 1 || val2.Get() != 2 || val3.Get() != 4 {
			t.Errorf("Expected 1, 2, 4, got %d, %d, %d", val1.Get(), val2.Get(), val3.Get())
		}
	})

	t.Run("Prefixed", func(t *testing.T) {
		g := NewPrefixed("item", 1)
		val1 := g.Next("FirstItem")
		val2 := g.Next("SecondItem")

		if val1.Get() != "item1" || val2.Get() != "item2" {
			t.Errorf(`Expected "item1", "item2", got %q, %q`, val1.Get(), val2.Get())
		}
	})

	t.Run("Cyclic", func(t *testing.T) {
		g := NewCyclic(3) // Cycles 0, 1, 2
		v0 := g.Next("Zero")
		v1 := g.Next("One")
		v2 := g.Next("Two")
		v3 := g.Next("Three") // should wrap to 0
		if v0.Get() != 0 || v1.Get() != 1 || v2.Get() != 2 || v3.Get() != 0 {
			t.Errorf("Expected 0, 1, 2, 0, got %d, %d, %d, %d", v0.Get(), v1.Get(), v2.Get(), v3.Get())
		}
	})

	t.Run("Mapped", func(t *testing.T) {
		g := NewMapped(map[string]int{
			"Low": 1, "Medium": 5, "High": 10,
		})
		if val, ok := g.Get("Medium"); !ok || val != 5 {
			t.Errorf("Expected 5 for Medium, got %d", val)
		}
		if name, ok := g.Name(10); !ok || name != "High" {
			t.Errorf("Expected High for 10, got %s", name)
		}
		if !reflect.DeepEqual(g.NameMap(), map[string]int{"Low": 1, "Medium": 5, "High": 10}) {
			t.Error("NameMap() returned incorrect data")
		}
		if !reflect.DeepEqual(g.ValueMap(), map[int]string{1: "Low", 5: "Medium", 10: "High"}) {
			t.Error("ValueMap() returned incorrect data")
		}
	})

	t.Run("WithCustomIncrementer", func(t *testing.T) {
		g := NewGenerator[int](
			WithStart(0),
			WithIncrementer(func(i int) int { return i + 5 }),
		)
		v1 := g.Next("Zero")
		v2 := g.Next("Five")
		if v1.Get() != 0 || v2.Get() != 5 {
			t.Errorf("Expected 0, 5, got %d, %d", v1.Get(), v2.Get())
		}
	})

	t.Run("NewCyclic with Zero Modulus", func(t *testing.T) {
		g := NewCyclic(0)
		v1 := g.Next("Zero")
		v2 := g.Next("One")
		if v1.Get() != 0 || v2.Get() != 0 {
			t.Errorf("Expected cyclic generator with modulus 0 to stay at 0, got %d, %d", v1.Get(), v2.Get())
		}
	})

	t.Run("NewMapped Empty", func(t *testing.T) {
		g := NewMapped(map[string]int{})
		if len(g.Values()) != 0 {
			t.Errorf("Expected empty mapped generator to have no values, got %d", len(g.Values()))
		}
		if _, ok := g.Get("Any"); ok {
			t.Error("Expected Get to return false for empty generator")
		}
	})
}

func TestGenerator_JSON(t *testing.T) {
	g := NewMapped(map[string]int{"A": 1, "B": 2})
	b, err := json.Marshal(g)
	if err != nil {
		t.Fatalf("MarshalJSON failed: %v", err)
	}
	// Note: map marshaling order is not guaranteed
	expected1 := `{"1":"A","2":"B"}`
	expected2 := `{"2":"B","1":"A"}`
	if s := string(b); s != expected1 && s != expected2 {
		t.Errorf("Expected %s or %s, got %s", expected1, expected2, s)
	}

	var newG Generator[int]
	if err := json.Unmarshal(b, &newG); err != nil {
		t.Fatalf("UnmarshalJSON failed: %v", err)
	}
	if name, ok := newG.Name(2); !ok || name != "B" {
		t.Errorf("Expected unmarshaled generator to have mapping for 2->B")
	}

	t.Run("UnmarshalJSON Invalid JSON", func(t *testing.T) {
		var g Generator[int]
		err := json.Unmarshal([]byte("invalid"), &g)
		if err == nil {
			t.Error("Expected UnmarshalJSON to fail on invalid JSON")
		}
	})
}

func TestGenerator_Parse(t *testing.T) {
	g := NewGenerator[int]()
	g.Next("One") // 0
	g.Next("Two") // 1

	t.Run("ParseByName", func(t *testing.T) {
		v, err := g.Parse("Two")
		if err != nil || v.Get() != 1 {
			t.Errorf("Expected value 1 for name 'Two', got %d, err: %v", v.Get(), err)
		}
	})
	t.Run("ParseByValue", func(t *testing.T) {
		v, err := g.Parse("0")
		if err != nil || v.Get() != 0 || v.String() != "One" {
			t.Errorf("Expected value 0/One for string '0', got %d/%s, err: %v", v.Get(), v.String(), err)
		}
	})
	t.Run("ParseInvalid", func(t *testing.T) {
		_, err := g.Parse("Three")
		if err == nil {
			t.Error("Expected error for parsing unknown name, got nil")
		}
	})
	t.Run("MustParseSuccess", func(t *testing.T) {
		v := g.MustParse("Two")
		if v.Get() != 1 {
			t.Errorf("Expected 1, got %d", v.Get())
		}
	})
}

func TestGenerator_Validate(t *testing.T) {
	g := NewMapped(map[string]int{"OK": 200, "Error": 500})
	if err := g.Validate(200); err != nil {
		t.Errorf("Expected value 200 to be valid, got error: %v", err)
	}
	if err := g.Validate(404); err == nil {
		t.Error("Expected value 404 to be invalid, got nil error")
	}
	if err := g.ValidateName("OK"); err != nil {
		t.Errorf("Expected name 'OK' to be valid, got error: %v", err)
	}
	if err := g.ValidateName("NotFound"); err == nil {
		t.Error("Expected name 'NotFound' to be invalid, got nil error")
	}
}

func TestGenerator_Panics(t *testing.T) {
	t.Run("NextOnMapped", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected Next on mapped generator to panic")
			}
		}()
		g := NewMapped(map[string]int{"A": 1})
		g.Next("B")
	})
	t.Run("MustParseInvalid", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected MustParse on invalid input to panic")
			}
		}()
		g := NewGenerator[int]()
		g.MustParse("Invalid")
	})
}

func TestGenerator_Concurrency(t *testing.T) {
	g := NewGenerator[int]()
	var wg sync.WaitGroup
	numGoroutines := 100

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			g.Next("SomeName")
		}()
	}
	wg.Wait()

	if len(g.Values()) != numGoroutines {
		t.Errorf("Expected %d values after concurrent Next calls, got %d", numGoroutines, len(g.Values()))
	}
}

func TestGenerator_Additional(t *testing.T) {
	// Existing tests (NewCyclic with Zero Modulus, NewMapped Empty) remain unchanged.
	// Add new test case:
	t.Run("Invalid Incrementer", func(t *testing.T) {
		g := NewGenerator[int](
			WithStart(0),
			WithIncrementer(func(i int) int { return 0 }), // Always returns 0
		)
		v1 := g.Next("Zero")
		v2 := g.Next("ZeroAgain")
		if v1.Get() != 0 || v2.Get() != 0 {
			t.Errorf("Expected values to be 0, 0, got %d, %d", v1.Get(), v2.Get())
		}
		if len(g.Values()) != 2 {
			t.Errorf("Expected 2 values, got %d", len(g.Values()))
		}
	})
}
