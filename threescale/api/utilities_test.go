package api

import (
	"reflect"
	"testing"
)

func TestHierarchy_DeepCopy(t *testing.T) {
	h := make(Hierarchy)
	h["hits"] = []string{"x", "y", "z"}

	clone := h.DeepCopy()
	clone["hits"] = []string{"x"}

	if len(h["hits"]) != 3 {
		t.Error("expected changes in cloned value to not modify original")
	}
}

func TestMetrics_Add(t *testing.T) {
	m := make(Metrics)
	current, err := m.Add("test", 1)
	if err != nil {
		t.Error("unexpected error")
	}

	if m["test"] != 1 || current != 1 {
		t.Error("unexpected value")
	}

	current, err = m.Add("test", 2)
	if err != nil {
		t.Error("unexpected error")
	}
	if m["test"] != 3 || current != 3 {
		t.Error("unexpected value")
	}

	current, err = m.Add("test", -1)
	if err != nil {
		t.Error("unexpected error")
	}
	if m["test"] != 2 || current != 2 {
		t.Error("unexpected value")
	}

	current, err = m.Add("test", -100)
	if err == nil {
		t.Error("expected error but got none")
	}
	if m["test"] != 2 || current != 2 {
		t.Error("unexpected value")
	}
}

func TestMetrics_Set(t *testing.T) {
	m := make(Metrics)
	err := m.Set("test", 5)
	if err != nil {
		t.Error("unexpected error")
	}

	if m["test"] != 5 {
		t.Error("unexpected value")
	}

	err = m.Set("test", 1)
	if err != nil {
		t.Error("unexpected error")
	}

	if m["test"] != 1 {
		t.Error("unexpected value")
	}

	err = m.Set("test", -100)
	if err == nil {
		t.Error("expected error but got none")
	}
}

func TestMetrics_Delete(t *testing.T) {
	m := Metrics{"test": 1}
	if len(m) != 1 {
		t.Error("item was not added")
	}
	m.Delete("test")
	if len(m) != 0 {
		t.Error("item was not deleted")
	}
}

func TestMetrics_DeepCopy(t *testing.T) {
	original := Metrics{"hits": 1, "test": 2}
	clone := original.DeepCopy()

	v, ok := clone["test"]
	if !ok {
		t.Error("expected deep copy function to have copied 'test' key and value")
	}
	if v != 2 {
		t.Error("unexpected value for 'test'")
	}

	clone["test"] = 3
	if v, _ = original["test"]; v != 2 {
		t.Error("unexpect value in original after modifying clone")
	}
}

func TestMetrics_AddHierarchyToMetrics(t *testing.T) {
	inputs := []struct {
		name      string
		original  Metrics
		hierarchy Hierarchy
		expect    Metrics
	}{
		{
			name:      "Test empty hierarchy returns a copy",
			original:  Metrics{"hits": 10, "test": 5},
			hierarchy: Hierarchy{},
			expect:    Metrics{"hits": 10, "test": 5},
		},
		{
			name:     "Test childless parent unaffected",
			original: Metrics{"hits": 10, "orphan": 5},
			hierarchy: Hierarchy{
				"other": []string{"child_one", "child_two"},
			},
			expect: Metrics{"hits": 10, "orphan": 5},
		},
		{
			name:     "Test child metrics reflected onto known parent",
			original: Metrics{"hits": 10, "orphan": 5, "child_one": 3},
			hierarchy: Hierarchy{
				"hits": []string{"child_one", "child_two"},
			},
			expect: Metrics{"hits": 13, "orphan": 5, "child_one": 3},
		},
		{
			name:     "Test child metrics reflected onto unknown parent",
			original: Metrics{"child_one": 3},
			hierarchy: Hierarchy{
				"hits": []string{"child_one", "child_two"},
			},
			expect: Metrics{"hits": 3, "child_one": 3},
		},
	}

	for _, test := range inputs {
		t.Run(test.name, func(t *testing.T) {
			got := test.original.AddHierarchyToMetrics(test.hierarchy)
			if !reflect.DeepEqual(got, test.expect) {
				t.Errorf("unexpected metrics computed, expected %v, but got %v", test.expect, got)
			}
		})
	}
}

func TestMetrics_SubtractHierarchyFromMetrics(t *testing.T) {
	inputs := []struct {
		name      string
		original  Metrics
		hierarchy Hierarchy
		expect    Metrics
	}{
		{
			name:      "Test empty hierarchy returns a copy",
			original:  Metrics{"hits": 10, "test": 5},
			hierarchy: Hierarchy{},
			expect:    Metrics{"hits": 10, "test": 5},
		},
		{
			name:     "Test childless parent unaffected",
			original: Metrics{"hits": 10, "orphan": 5},
			hierarchy: Hierarchy{
				"other": []string{"child_one", "child_two"},
			},
			expect: Metrics{"hits": 10, "orphan": 5},
		},
		{
			name:     "Test child metrics reflected onto known parent",
			original: Metrics{"hits": 10, "orphan": 5, "child_one": 3},
			hierarchy: Hierarchy{
				"hits": []string{"child_one", "child_two"},
			},
			expect: Metrics{"hits": 7, "orphan": 5, "child_one": 3},
		},
		{
			name:     "Test child metrics reflected onto unknown parent",
			original: Metrics{"child_one": 3},
			hierarchy: Hierarchy{
				"hits": []string{"child_one", "child_two"},
			},
			expect: Metrics{"child_one": 3},
		},
		{
			name:     "Test child metrics remove parent if negative values occur",
			original: Metrics{"hits": 4, "child_one": 5},
			hierarchy: Hierarchy{
				"hits": []string{"child_one", "child_two"},
			},
			expect: Metrics{"child_one": 5},
		},
	}

	for _, test := range inputs {
		t.Run(test.name, func(t *testing.T) {
			got := test.original.SubtractHierarchyFromMetrics(test.hierarchy)
			if !reflect.DeepEqual(got, test.expect) {
				t.Errorf("unexpected metrics computed, expected %v, but got %v", test.expect, got)
			}
		})
	}
}
