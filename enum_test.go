package enum

import (
	"testing"
)

func TestGenerator(t *testing.T) {
	t.Run("NumericSequence", func(t *testing.T) {
		g := NewGenerator[int](WithStart(10))
		val1 := g.Next("Ten")
		val2 := g.Next("Eleven")
		val3 := g.Next("Twelve")

		if val1.Value() != 10 || val1.String() != "Ten" {
			t.Errorf("Expected 10/Ten, got %d/%s", val1.Value(), val1.String())
		}
		if val2.Value() != 11 || val2.String() != "Eleven" {
			t.Errorf("Expected 11/Eleven, got %d/%s", val2.Value(), val2.String())
		}
		if val3.Value() != 12 || val3.String() != "Twelve" {
			t.Errorf("Expected 12/Twelve, got %d/%s", val3.Value(), val3.String())
		}

		if name, ok := g.Name(11); !ok || name != "Eleven" {
			t.Errorf("Expected Eleven for value 11, got %s", name)
		}

		if val, ok := g.Value("Twelve"); !ok || val != 12 {
			t.Errorf("Expected 12 for name Twelve, got %d", val)
		}

		if !g.Contains(11) {
			t.Error("Expected generator to contain value 11")
		}

		names := g.Names()
		expectedNames := []string{"Ten", "Eleven", "Twelve"}
		if len(names) != len(expectedNames) {
			t.Errorf("Expected %d names, got %d", len(expectedNames), len(names))
		}
		for i, name := range expectedNames {
			if names[i] != name {
				t.Errorf("Expected name %s at index %d, got %s", name, i, names[i])
			}
		}
	})

	t.Run("StringSequence", func(t *testing.T) {
		g := NewAlpha()
		val1 := g.Next("First")
		val2 := g.Next("Second")
		val3 := g.Next("Third")

		if val1.Value() != "A" || val1.String() != "First" {
			t.Errorf("Expected A/First, got %s/%s", val1.Value(), val1.String())
		}
		if val2.Value() != "B" || val2.String() != "Second" {
			t.Errorf("Expected B/Second, got %s/%s", val2.Value(), val2.String())
		}
		if val3.Value() != "C" || val3.String() != "Third" {
			t.Errorf("Expected C/Third, got %s/%s", val3.Value(), val3.String())
		}
	})

	t.Run("BitFlags", func(t *testing.T) {
		g := NewBitFlagGenerator[uint](1)
		val1 := g.Next("Read")
		val2 := g.Next("Write")
		val3 := g.Next("Execute")

		if val1.Value() != 1 || val1.String() != "Read" {
			t.Errorf("Expected 1/Read, got %d/%s", val1.Value(), val1.String())
		}
		if val2.Value() != 2 || val2.String() != "Write" {
			t.Errorf("Expected 2/Write, got %d/%s", val2.Value(), val2.String())
		}
		if val3.Value() != 4 || val3.String() != "Execute" {
			t.Errorf("Expected 4/Execute, got %d/%s", val3.Value(), val3.String())
		}
	})

	t.Run("Prefixed", func(t *testing.T) {
		g := NewPrefixed("item", 1)
		val1 := g.Next("FirstItem")
		val2 := g.Next("SecondItem")

		if val1.Value() != "item1" || val1.String() != "FirstItem" {
			t.Errorf("Expected item1/FirstItem, got %s/%s", val1.Value(), val1.String())
		}
		if val2.Value() != "item2" || val2.String() != "SecondItem" {
			t.Errorf("Expected item2/SecondItem, got %s/%s", val2.Value(), val2.String())
		}
	})

	t.Run("Mapped", func(t *testing.T) {
		g := NewMapped(map[string]int{
			"Low":    1,
			"Medium": 5,
			"High":   10,
		})

		if val, ok := g.Value("Medium"); !ok || val != 5 {
			t.Errorf("Expected 5 for Medium, got %d", val)
		}
		if name, ok := g.Name(10); !ok || name != "High" {
			t.Errorf("Expected High for 10, got %s", name)
		}
	})
}

func TestMaker(t *testing.T) {
	t.Run("BasicEnum", func(t *testing.T) {
		type Status int

		type StatusStruct struct {
			Pending Status
			Active  Status
			Closed  Status
		}
		statuses := Make[StatusStruct, Status](StatusStruct{})

		if val, ok := statuses.Get("Active"); !ok || val != 1 {
			t.Errorf("Expected Active to be 1, got %d", val)
		}

		if name, ok := statuses.Name(2); !ok || name != "Closed" {
			t.Errorf("Expected Closed for 2, got %s", name)
		}

		if !statuses.Contains(0) {
			t.Error("Expected enum to contain value 0")
		}

		if !statuses.ContainsName("Pending") {
			t.Error("Expected enum to contain name Pending")
		}

		names := statuses.Names()
		expectedNames := []string{"Pending", "Active", "Closed"}
		if len(names) != len(expectedNames) {
			t.Errorf("Expected %d names, got %d", len(expectedNames), len(names))
		}
		for i, name := range expectedNames {
			if names[i] != name {
				t.Errorf("Expected name %s at index %d, got %s", name, i, names[i])
			}
		}

		entries := statuses.Entries()
		if len(entries) != 3 {
			t.Errorf("Expected 3 entries, got %d", len(entries))
		}
		if entries[1].Value() != 1 || entries[1].String() != "Active" {
			t.Errorf("Expected entry 1 to be 1/Active, got %d/%s", entries[1].Value(), entries[1].String())
		}
	})

	t.Run("DifferentUnderlyingType", func(t *testing.T) {
		type SmallEnum uint8
		type SmallStruct struct {
			One   SmallEnum
			Two   SmallEnum
			Three SmallEnum
		}
		small := Make[SmallStruct, SmallEnum](SmallStruct{})

		if val, ok := small.Get("Two"); !ok || val != 1 {
			t.Errorf("Expected Two to be 1, got %d", val)
		}

		if name, ok := small.Name(2); !ok || name != "Three" {
			t.Errorf("Expected Three for 2, got %s", name)
		}
	})
}

func TestEdgeCases(t *testing.T) {
	t.Run("EmptyGenerator", func(t *testing.T) {
		g := NewGenerator[string]()
		if len(g.Values()) != 0 {
			t.Error("New generator should have no values")
		}
		if g.Contains("anything") {
			t.Error("Empty generator should contain nothing")
		}
	})

	t.Run("DuplicateNames", func(t *testing.T) {
		g := NewGenerator[int]()
		g.Next("Duplicate")
		g.Next("Duplicate") // Should allow duplicates

		if len(g.Values()) != 2 {
			t.Error("Should allow duplicate names")
		}
	})

	t.Run("MakerWithNoFields", func(t *testing.T) {
		type EmptyEnum int
		type EmptyStruct struct{}
		empty := Make[EmptyStruct, EmptyEnum](EmptyStruct{})

		if len(empty.Names()) != 0 {
			t.Error("Empty struct should produce no enum values")
		}
	})
}
