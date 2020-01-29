package api

import (
	"fmt"
)

// DeepCopy returns a clone of the original Metrics. It provides a deep copy
// of both the key and the value of the original Hierarchy.
func (h Hierarchy) DeepCopy() Hierarchy {
	clone := make(Hierarchy, len(h))
	for k, v := range h {
		var clonedV []string
		clonedV = append(clonedV, v...)
		clone[k] = clonedV
	}
	return clone
}

// AddHierarchyToMetrics takes the provided hierarchy structure, and uses it
// to determine how the metrics, m, are affected, incrementing parent metrics
// based on the value of the parents child/children metrics.
// Returns new Metrics, leaving metrics m in it's original state.
func (m Metrics) AddHierarchyToMetrics(hierarchy Hierarchy) Metrics {
	metrics := m.DeepCopy()

	for parent, children := range hierarchy {
		for metric, v := range metrics {
			if contains(metric, children) {
				if _, known := metrics[parent]; known {
					metrics.Add(parent, v)
				} else {
					metrics.Set(parent, v)
				}
			}
		}
	}
	return metrics
}

// Add takes a provided key and value and adds them to the Metric 'm'
// If the metric already existed in 'm', then the value will be added (if positive) or subtracted (if negative) from the existing value.
// If a subtraction leads to a negative value Add returns an error  and the change will be discarded.
// Returns the updated value (or current value in error cases) as well as the error.
func (m Metrics) Add(name string, value int) (int, error) {
	if currentValue, ok := m[name]; ok {
		newValue := currentValue + value
		if newValue < 0 {
			return currentValue, fmt.Errorf("invalid value for metric %s post computation. this will result in 403 from 3scale", name)
		}
		m[name] = newValue
		return newValue, nil
	}
	m[name] = value
	return value, nil
}

// Set takes a provided key and value and sets that value of the key in 'm', overwriting any value that exists previously.
func (m Metrics) Set(name string, value int) error {
	if value < 0 {
		return fmt.Errorf("invalid value for metric %s post computation. this will result in 403 from 3scale", name)
	}
	m[name] = value
	return nil
}

// Delete a metric m['name'] if present
func (m Metrics) Delete(name string) {
	delete(m, name)
}

// DeepCopy returns a clone of the original Metrics
func (m Metrics) DeepCopy() Metrics {
	clone := make(Metrics, len(m))
	for k, v := range m {
		clone[k] = v
	}
	return clone
}

func contains(key string, in []string) bool {
	for _, i := range in {
		if key == i {
			return true
		}
	}
	return false
}
