package api

import "testing"

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
