package threescale

import (
	"context"
	"net/http"
	"testing"
)

func TestNewClient(t *testing.T) {
	_, err := NewClient("ftp://invalid.com", http.DefaultClient)
	if err == nil {
		t.Error("expected error for invalid scheme")
	}

	c, err := NewClient(defaultBackendUrl, http.DefaultClient)
	if err != nil {
		t.Error("unexpected error when creating client")
	}

	if c.GetPeer() != "su1.3scale.net" {
		t.Error("unexpected hostname set via constructor")
	}
}

func TestNewDefaultClient(t *testing.T) {
	c, _ := NewDefaultClient()

	if c.baseURL != defaultBackendUrl {
		t.Error("unexpected setting in default client")
	}

	if c.httpClient.Timeout != defaultTimeout {
		t.Error("unexpected setting in default client")
	}
}

func TestNewTransaction(t *testing.T) {
	r := NewTransaction(
		Params{AppID: "any"},
		WithExtensions(Extensions{HierarchyExtension: "1", LimitExtension: "1"}),
		WithContext(context.TODO()))
	if r.context != context.TODO() {
		t.Error("expected context to be set")
	}

	if len(r.extensions) != 2 {
		t.Error("expected extensions to be set")
	}

}

func TestAuthorizeResponse_GetHierarchy(t *testing.T) {
	h := make(Hierarchy)
	h["test"] = []string{"example"}
	resp := &AuthorizeResponse{hierarchy: h}

	got := resp.GetHierarchy()
	if len(got) != 1 {
		t.Error("unexpected map len")
	}

	if _, ok := got["test"]; !ok {
		t.Error("expected key to exist")
	}
}
func TestAuthorizeResponse_GetUsageReports(t *testing.T) {
	ur := make(UsageReports)
	report := UsageReport{
		Period:       Eternity,
		MaxValue:     100,
		CurrentValue: 50,
	}

	ur["test"] = report
	resp := &AuthorizeResponse{usageReports: ur}
	equals(t, resp.GetUsageReports()["test"], report)
}

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
	if v, ok = original["test"]; v != 2 {
		t.Error("unexpect value in original after modifying clone")
	}
}

func TestRateLimits_GetLimitRemaining(t *testing.T) {
	rl := &RateLimits{
		limitRemaining: 100,
		limitReset:     500,
	}

	if rl.GetLimitRemaining() != 100 {
		t.Error("unexpected value")
	}
}

func TestRateLimits_GetLimitReset(t *testing.T) {
	rl := &RateLimits{
		limitRemaining: 100,
		limitReset:     500,
	}

	if rl.GetLimitReset() != 500 {
		t.Error("unexpected value")
	}
}
