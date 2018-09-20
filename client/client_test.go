package client

import (
	"fmt"
	"net/http"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
	"time"
)

// Mocking objects for HTTP tests
type RoundTripFunc func(req *http.Request) *http.Response

func (f RoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}

// Get a test client with transport overridden for mocking
func NewTestClient(fn RoundTripFunc) *http.Client {
	return &http.Client{
		Transport: RoundTripFunc(fn),
	}
}

// Tests creation of a new 3scale backend
func TestNewBackend(t *testing.T) {
	_, err := NewBackend("", "test.com", 443)
	if err == nil {
		fmt.Println(err)
		t.Fail()
	}
	_, err = NewBackend("https", "test.com", 443)
	if err != nil {
		t.Fail()
	}
	_, err = NewBackend("ftp", "test.com", 443)
	if err == nil {
		t.Fail()
	}
}

// Asserts correct dependency injection into client overwrites defaults
func TestNewThreeScale(t *testing.T) {
	validBe, err := NewBackend("https", "test.com", 443)
	if err != nil {
		t.Fail()
	}
	threeScale := NewThreeScale(validBe, &http.Client{
		Timeout: time.Duration(5),
	})
	if threeScale.backend == DefaultBackend() {
		t.Fail()
	}
	if reflect.DeepEqual(threeScale.httpClient, http.DefaultClient) {
		t.Fail()
	}

	threeScaleTwo := NewThreeScale(nil, nil)
	if threeScaleTwo.httpClient != http.DefaultClient {
		t.Fail()
	}
	equals(t, threeScaleTwo.backend, DefaultBackend())
}

// Get default success response for authorize endpoint
func getAuthSuccess() string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<status>
  <authorized>true</authorized>
  <plan>Basic</plan>
</status>`
}

// Returns a default client for testing
func threeScaleTestClient(hc *http.Client) *ThreeScaleClient {
	client := NewThreeScale(DefaultBackend(), hc)
	return client
}

// equals fails the test if exp is not equal to act.
func equals(tb testing.TB, exp, act interface{}) {
	if !reflect.DeepEqual(exp, act) {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d:\n\n\texp: %#v\n\n\tgot: %#v\033[39m\n\n", filepath.Base(file), line, exp, act)
		tb.FailNow()
	}
}
