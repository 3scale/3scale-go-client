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

func TestPeriodWindow_IsEqual(t *testing.T) {
	base := PeriodWindow{
		Period: Minute,
		Start:  100,
		End:    1000,
	}

	input := []struct {
		name         string
		compareTo    PeriodWindow
		expectResult bool
	}{
		{
			name: "Test false when period differs",
			compareTo: PeriodWindow{
				Period: Hour,
				Start:  100,
				End:    1000,
			},
			expectResult: false,
		},
		{
			name: "Test false when start differs",
			compareTo: PeriodWindow{
				Period: Minute,
				Start:  1000,
				End:    1000,
			},
			expectResult: false,
		},
		{
			name: "Test false when end differs",
			compareTo: PeriodWindow{
				Period: Minute,
				Start:  100,
				End:    100,
			},
			expectResult: false,
		},
		{
			name: "Test true when equal",
			compareTo: PeriodWindow{
				Period: Minute,
				Start:  100,
				End:    1000,
			},
			expectResult: true,
		},
	}

	for _, test := range input {
		t.Run(test.name, func(t *testing.T) {
			isEqual := base.IsEqual(test.compareTo)
			if isEqual != test.expectResult {
				t.Errorf("unexpected result during comparison, wanted %v but got %v",
					test.expectResult, isEqual)
			}
		})
	}
}

func TestUsageReport_IsSame(t *testing.T) {
	basePeriodWindow := PeriodWindow{
		Period: Minute,
		Start:  100,
		End:    1000,
	}

	baseUsageReport := UsageReport{
		PeriodWindow: basePeriodWindow,
		MaxValue:     100,
		CurrentValue: 10,
	}

	input := []struct {
		name         string
		compareTo    UsageReport
		expectResult bool
	}{
		{
			name: "Test false when period window differs",
			compareTo: UsageReport{
				PeriodWindow: PeriodWindow{},
				MaxValue:     100,
				CurrentValue: 10,
			},
			expectResult: false,
		},
		{
			name: "Test false when max differs",
			compareTo: UsageReport{
				PeriodWindow: basePeriodWindow,
				MaxValue:     10,
				CurrentValue: 10,
			},
			expectResult: false,
		},
		{
			name: "Test true even when CurrentValue differs",
			compareTo: UsageReport{
				PeriodWindow: basePeriodWindow,
				MaxValue:     100,
				CurrentValue: 5,
			},
			expectResult: true,
		},
		{
			name: "Test true when same",
			compareTo: UsageReport{
				PeriodWindow: basePeriodWindow,
				MaxValue:     100,
				CurrentValue: 10,
			},
			expectResult: true,
		},
	}

	for _, test := range input {
		t.Run(test.name, func(t *testing.T) {
			isSame := baseUsageReport.IsSame(test.compareTo)
			if isSame != test.expectResult {
				t.Errorf("unexpected result during comparison, wanted %v but got %v",
					test.expectResult, isSame)
			}
		})
	}
}
