// maker_test.go
package enum

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestMaker(t *testing.T) {
	t.Run("BasicEnum", func(t *testing.T) {
		type Status int

		type StatusStruct struct {
			Pending Status
			Active  Status
			Closed  Status
		}
		statuses := Make[StatusStruct, Status](&StatusStruct{})

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
		if entries[1].Get() != 1 || entries[1].String() != "Active" {
			t.Errorf("Expected entry 1 to be 1/Active, got %d/%s", entries[1].Get(), entries[1].String())
		}
	})

	t.Run("DifferentUnderlyingType", func(t *testing.T) {
		type SmallEnum uint8
		type SmallStruct struct {
			One   SmallEnum
			Two   SmallEnum
			Three SmallEnum
		}
		small := Make[SmallStruct, SmallEnum](&SmallStruct{})

		if val, ok := small.Get("Two"); !ok || val != 1 {
			t.Errorf("Expected Two to be 1, got %d", val)
		}

		if name, ok := small.Name(2); !ok || name != "Three" {
			t.Errorf("Expected Three for 2, got %s", name)
		}
	})

	t.Run("Panic on Non-Struct Pointer", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic for non-struct pointer")
			}
		}()
		var i int
		Make[int, int](&i)
	})

	t.Run("Panic on Too Many Fields", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic for too many fields")
			}
		}()
		// An int8 can hold 128 non-negative values (0-127).
		// We define a struct with 129 fields to trigger the panic.
		type LargeStruct struct {
			F0, F1, F2, F3, F4, F5, F6, F7, F8, F9, F10, F11, F12, F13, F14, F15, F16, F17, F18, F19, F20, F21, F22, F23, F24, F25, F26, F27, F28, F29, F30, F31, F32, F33, F34, F35, F36, F37, F38, F39, F40, F41, F42, F43, F44, F45, F46, F47, F48, F49, F50, F51, F52, F53, F54, F55, F56, F57, F58, F59, F60, F61, F62, F63, F64, F65, F66, F67, F68, F69, F70, F71, F72, F73, F74, F75, F76, F77, F78, F79, F80, F81, F82, F83, F84, F85, F86, F87, F88, F89, F90, F91, F92, F93, F94, F95, F96, F97, F98, F99, F100, F101, F102, F103, F104, F105, F106, F107, F108, F109, F110, F111, F112, F113, F114, F115, F116, F117, F118, F119, F120, F121, F122, F123, F124, F125, F126, F127, F128 int8
		}
		Make[LargeStruct, int8](&LargeStruct{})
	})

	t.Run("Unexported Fields", func(t *testing.T) {
		type MixedStruct struct {
			Exported   int
			unexported int
		}
		m := Make[MixedStruct, int](&MixedStruct{})
		if len(m.Entries()) != 1 {
			t.Errorf("Expected 1 entry (exported field only), got %d", len(m.Entries()))
		}
		if name, ok := m.Name(0); !ok || name != "Exported" {
			t.Errorf("Expected Exported for 0, got %s", name)
		}
	})

	t.Run("JSON Marshal and Unmarshal", func(t *testing.T) {
		type Status struct {
			Pending int
			Active  int
		}
		m := Make[Status, int](&Status{})
		b, err := json.Marshal(m)
		if err != nil {
			t.Fatalf("MarshalJSON failed: %v", err)
		}
		expected1 := `{"0":"Pending","1":"Active"}`
		expected2 := `{"1":"Active","0":"Pending"}` // Order not guaranteed
		if s := string(b); s != expected1 && s != expected2 {
			t.Errorf("Expected JSON like %s, got %s", expected1, s)
		}

		var s2 Status
		newM := Make[Status, int](&s2)
		err = newM.UnmarshalJSON(b)
		if err != nil {
			t.Fatalf("UnmarshalJSON failed: %v", err)
		}
		if name, ok := newM.Name(1); !ok || name != "Active" {
			t.Errorf("Expected Active for 1 after unmarshal, got %s", name)
		}
	})
}
func TestMaker_Additional(t *testing.T) {
	// Existing tests remain unchanged.
	// Add new test case:
	t.Run("Empty Struct", func(t *testing.T) {
		type EmptyStruct struct{}
		m := Make[EmptyStruct, int](&EmptyStruct{})
		if len(m.Entries()) != 0 {
			t.Errorf("Expected 0 entries for empty struct, got %d", len(m.Entries()))
		}
		if _, ok := m.Get("Any"); ok {
			t.Error("Expected Get to return false for empty struct")
		}
	})

	t.Run("Only Unexported Fields", func(t *testing.T) {
		type UnexportedStruct struct {
			hidden int
		}
		m := Make[UnexportedStruct, int](&UnexportedStruct{})
		if len(m.Entries()) != 0 {
			t.Errorf("Expected 0 entries for struct with only unexported fields, got %d", len(m.Entries()))
		}
	})
}

func TestMaker_JSON_Marshal_and_Unmarshal(t *testing.T) {
	type Colors struct{ Red, Blue int }
	var c Colors
	m := Make[Colors, int](&c)
	jsonData, err := m.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}

	var c2 Colors
	m2 := Make[Colors, int](&c2)
	if err := m2.UnmarshalJSON(jsonData); err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(m.ValueMap(), m2.ValueMap()) {
		t.Errorf("expected valueMap %v, got %v", m.ValueMap(), m2.ValueMap())
	}
	// This check is valid because Make(&c2) sets the fields to 0 and 1.
	// UnmarshalJSON does not change them, so they should remain.
	if c2.Red != 0 || c2.Blue != 1 {
		t.Errorf("expected struct {Red:0, Blue:1}, got %+v", c2)
	}
}

func TestMakeManualWithBasic_JSON(t *testing.T) {
	type Colors struct{ Red, Blue Basic }
	var c Colors
	b := NewBasic()
	m := MakeManualWithBasic(&c, b, func(b *Basic) *Colors {
		c.Red = b.Add("Red")
		c.Blue = b.Add("Blue")
		return &c
	})

	// Test MarshalJSON
	data, err := m.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	expected1 := `{"0":"Red","1":"Blue"}`
	expected2 := `{"1":"Blue","0":"Red"}` // Order is not guaranteed
	if s := string(data); s != expected1 && s != expected2 {
		t.Errorf("expected JSON like %q, got %q", expected1, s)
	}

	// Test UnmarshalJSON
	var c2 Colors
	b2 := NewBasic()
	// We must define the same enum values for the new maker before unmarshaling.
	m2 := MakeManualWithBasic(&c2, b2, func(b *Basic) *Colors {
		c2.Red = b.Add("Red")
		c2.Blue = b.Add("Blue")
		return &c2
	})
	if err := m2.UnmarshalJSON(data); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(m.ValueMap(), m2.ValueMap()) {
		t.Errorf("expected valueMap %v, got %v", m.ValueMap(), m2.ValueMap())
	}
}
