package client

import (
	"fmt"
	"math/rand"
	"net/http"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"
	"unsafe"
)

var ext_tested bool

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

// Tests the correct hostname is returned for a client remote end
func TestGetPeer(t *testing.T) {
	be, err := NewBackend("https", "www.test.com", 443)
	if err != nil {
		t.Fatalf("error creating client")
	}
	c := NewThreeScale(be, nil)
	host := c.GetPeer()
	if host != "www.test.com" {
		t.Fatalf("unexpected hostname")
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

func getExtensions(t *testing.T) map[string]string {
	t.Helper()

	// ensure we at least return the extensions the first time we get called
	if !ext_tested || rand.Intn(2) != 0 {
		ext_tested = true
		return map[string]string{
			"no_body":       "1",
			"asingle;field": "and;single;value",
			"many@@and==":   "should@@befine==",
			"a test&":       "&ok",
		}
	} else {
		return nil
	}
}

// returns a randomly-ordered list of strings for extensions with format "key=value"
func getExtensionsValue() []string {
	expected := map[string]string{
		"no_body":             "1",
		"asingle%3Bfield":     "and%3Bsingle%3Bvalue",
		"many%40%40and%3D%3D": "should%40%40befine%3D%3D",
		"a+test%26":           "%26ok",
	}

	exp := make([]string, 0, unsafe.Sizeof(expected))
	// golang's iteration over maps randomizes order of kv's
	for k, v := range expected {
		exp = append(exp, fmt.Sprintf("%s=%s", k, v))
	}

	return exp
}

func checkExtensions(t *testing.T, req *http.Request) (bool, string) {
	t.Helper()

	value := req.Header.Get("3scale-options")
	expected := getExtensionsValue()

	found := strings.Split(value, "&")

	if compareUnorderedStringLists(found, expected) {
		return true, ""
	} else {
		sort.Strings(expected)
		sort.Strings(found)

		return false, fmt.Sprintf("\nexpected extension header value %s\n"+
			"                      but found %s",
			strings.Join(expected, ", "), strings.Join(found, ", "))

	}
}

func compareUnorderedStringLists(one []string, other []string) bool {
	if len(one) != len(other) {
		return false
	}

	for _, x := range one {
		found := false

		for _, y := range other {
			if x == y {
				found = true
				break
			}
		}

		if !found {
			return false
		}
	}

	return true
}
