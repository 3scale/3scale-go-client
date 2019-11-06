package threescale

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
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

	"github.com/3scale/3scale-go-client/fake"
)

func TestClient_Authorize(t *testing.T) {
	const svcID = "test"

	ctx := context.Background()
	ctx, _ = context.WithDeadline(ctx, time.Now())

	inputs := []struct {
		name           string
		auth           ClientAuth
		request        *Request
		expectErr      bool
		expectErrMsg   string
		expectResponse *AuthorizeResponse
		client         *Client
		injectClient   *http.Client
	}{
		{
			name:         "Test expect failure invalid Params no app auth provided",
			request:      &Request{},
			expectErr:    true,
			expectErrMsg: badReqErrText,
		},
		{
			name: "Test expect failure invalid ClientAuth unknown auth type",
			auth: ClientAuth{
				Type:  3,
				Value: "any",
			},
			request:      &Request{Params: Params{AppID: "any"}},
			expectErr:    true,
			expectErrMsg: badReqErrText,
		},
		{
			name: "Test expect failure invalid ClientAuth empty value",
			auth: ClientAuth{
				Type:  ProviderKey,
				Value: "",
			},
			request:      &Request{Params: Params{AppID: "any"}},
			expectErr:    true,
			expectErrMsg: badReqErrText,
		},
		{
			name:         "Test expect failure bad url passed",
			auth:         ClientAuth{Type: ProviderKey, Value: "any"},
			request:      &Request{Params: Params{AppID: "any"}},
			expectErr:    true,
			expectErrMsg: httpReqErrText,
			client: &Client{
				backendHost: "/some/invalid/value%_",
				baseURL:     "/some/invalid/value%_",
				httpClient:  http.DefaultClient,
			},
		},
		{
			name:         "Test expect failure simulated network error",
			auth:         ClientAuth{Type: ProviderKey, Value: "any"},
			request:      &Request{Params: Params{AppID: "any"}},
			expectErr:    true,
			expectErrMsg: "Timeout exceeded",
			client: &Client{
				baseURL: defaultBackendUrl,
				httpClient: &http.Client{
					Timeout: time.Nanosecond,
				},
			},
		},
		{
			name:         "Test expect failure simulated bad response from 3scale error",
			auth:         ClientAuth{Type: ProviderKey, Value: "any"},
			request:      &Request{Params: Params{AppID: "any"}},
			expectErr:    true,
			expectErrMsg: "EOF",
			injectClient: NewTestClient(func(req *http.Request) *http.Response {
				return &http.Response{
					StatusCode: 200,
					Body:       ioutil.NopCloser(bytes.NewBufferString("EOF")),
					Header:     make(http.Header),
				}
			}),
		},
		{
			name: "Test params formatting",
			auth: ClientAuth{
				Type:  ServiceToken,
				Value: "any",
			},
			request: &Request{
				Params: Params{
					AppID:  "any",
					AppKey: "key",
				},
				Metrics: Metrics{"hits": 1, "other": 2},
			},
			expectResponse: &AuthorizeResponse{
				Success:    true,
				StatusCode: 200,
			},
			injectClient: NewTestClient(func(req *http.Request) *http.Response {
				// decodes to app_id=any&app_key=key&service_id=test&service_token=any&usage[hits]=1&usage[other]=2
				expect := `app_id=any&app_key=key&service_id=test&service_token=any&usage%5Bhits%5D=1&usage%5Bother%5D=2`

				if req.URL.RawQuery != expect {
					t.Error("unexpected result in query string")
				}

				return &http.Response{
					StatusCode: 200,
					Body:       ioutil.NopCloser(bytes.NewBufferString(fake.GetAuthSuccess())),
					Header:     make(http.Header),
				}
			}),
		},
		{
			name: "Test extension formatting",
			auth: ClientAuth{
				Type:  ServiceToken,
				Value: "any",
			},
			request: &Request{
				Params: Params{
					AppID: "any",
				},
				extensions: getExtensions(t),
			},
			expectResponse: &AuthorizeResponse{
				Success:    true,
				StatusCode: 200,
			},
			injectClient: NewTestClient(func(req *http.Request) *http.Response {
				ok, errMsg := checkExtensions(t, req)
				if !ok {
					t.Errorf("error in extensions - %s", errMsg)
				}

				return &http.Response{
					StatusCode: 200,
					Body:       ioutil.NopCloser(bytes.NewBufferString(fake.GetAuthSuccess())),
					Header:     make(http.Header),
				}
			}),
		},
		{
			name:    "Test usage reports",
			auth:    ClientAuth{Type: ProviderKey, Value: "any"},
			request: &Request{Params: Params{AppID: "any"}},
			expectResponse: &AuthorizeResponse{
				Success:    true,
				StatusCode: 200,
				usageReports: UsageReports{
					"hits": UsageReport{
						Period:       Minute,
						PeriodStart:  1550845920,
						PeriodEnd:    1550845980,
						MaxValue:     4,
						CurrentValue: 1,
					},
					"test_metric": UsageReport{
						Period:       Week,
						PeriodStart:  1550448000,
						PeriodEnd:    1551052800,
						MaxValue:     6,
						CurrentValue: 0,
					},
				},
			},
			injectClient: NewTestClient(func(req *http.Request) *http.Response {
				equals(t, req.URL.Path, authzEndpoint)
				resp := getUsageReportXML(t)

				return &http.Response{
					StatusCode: 200,
					Body:       ioutil.NopCloser(bytes.NewBufferString(resp)),
					Header:     make(http.Header),
				}
			}),
		},
		{
			name: "Test hierarchy extension",
			auth: ClientAuth{Type: ProviderKey, Value: "any"},
			request: NewRequest(Params{AppID: "any"},
				WithExtensions(Extensions{HierarchyExtension: "1"})),
			expectResponse: &AuthorizeResponse{
				Success:    true,
				StatusCode: 200,
				hierarchy:  Hierarchy{"hits": []string{"example", "sample", "test"}},
			},
			injectClient: NewTestClient(func(req *http.Request) *http.Response {
				expectValSet := req.Header.Get("3scale-Options")
				if expectValSet != "hierarchy=1" {
					t.Error("expected hierarchy feature to have been enabled via header")
				}
				equals(t, req.URL.Path, authzEndpoint)
				resp := getHierarchyXML(t)

				return &http.Response{
					StatusCode: 200,
					Body:       ioutil.NopCloser(bytes.NewBufferString(resp)),
					Header:     make(http.Header),
				}
			}),
		},
		{
			name: "Test authorization extensions - rate limiting",
			auth: ClientAuth{Type: ProviderKey, Value: "any"},
			request: NewRequest(Params{AppID: "any"},
				WithExtensions(Extensions{LimitExtension: "1"})),
			expectResponse: &AuthorizeResponse{
				Success:    true,
				StatusCode: 200,
				RateLimits: &RateLimits{
					limitRemaining: 5,
					limitReset:     100,
				},
			},
			injectClient: NewTestClient(func(req *http.Request) *http.Response {
				expectValSet := req.Header.Get("3scale-Options")
				if expectValSet != "limit_headers=1" {
					t.Error("expected rate limiting feature to have been enabled via header")
				}
				equals(t, req.URL.Path, authzEndpoint)

				header := http.Header{}
				header.Add(limitRemainingHeaderKey, "5")
				header.Add(limitResetHeaderKey, "100")

				return &http.Response{
					StatusCode: 200,
					Body:       ioutil.NopCloser(bytes.NewBufferString(fake.GetAuthSuccess())),
					Header:     header,
				}
			}),
		},
		{
			name: "Test context is respected",
			auth: ClientAuth{Type: ProviderKey, Value: "any"},
			request: NewRequest(Params{AppID: "any"},
				WithContext(ctx)),
			expectErr:    true,
			expectErrMsg: "context deadline exceeded",
			client: &Client{
				baseURL: defaultBackendUrl,
				httpClient: &http.Client{
					Timeout: time.Nanosecond,
				},
			},
		},
	}

	for _, input := range inputs {
		t.Run(input.name, func(t *testing.T) {
			if input.injectClient == nil {
				// fallback client
				input.injectClient = NewTestClient(func(req *http.Request) *http.Response {
					equals(t, req.URL.Path, authzEndpoint)
					return &http.Response{StatusCode: 200, Body: ioutil.NopCloser(bytes.NewBufferString(fake.GetAuthSuccess()))}
				})
			}

			c := input.client
			if c == nil {
				c = threeScaleTestClient(t, input.injectClient)
			}

			resp, err := c.Authorize(svcID, input.auth, input.request)
			if err != nil {
				if !input.expectErr {
					t.Error("unexpected error")
				}
				// we expected an error so ensure our err conditions are met
				if !strings.Contains(err.Error(), input.expectErrMsg) {
					t.Errorf("expected our error message to contain substring %s", input.expectErrMsg)
				}
				return
			}
			equals(t, input.expectResponse, resp)
		})
	}
}

// because auth and auth rep essentially follow the same pattern, we can minimise the test in this instance
// ensure our query param is correct and we are calling the correct endpoint
func TestClient_AuthRep(t *testing.T) {
	type input struct {
		name           string
		auth           ClientAuth
		request        *Request
		expectErr      bool
		expectErrMsg   string
		expectResponse *AuthorizeResponse
		client         *Client
		injectClient   *http.Client
	}

	fixture := input{
		name: "Test params formatting",
		auth: ClientAuth{
			Type:  ServiceToken,
			Value: "any",
		},
		request: &Request{
			Params: Params{
				AppID:  "any",
				AppKey: "key",
			},
			Metrics: Metrics{"hits": 1, "other": 2},
		},
		expectResponse: &AuthorizeResponse{
			Success:    true,
			StatusCode: 200,
		},
		injectClient: NewTestClient(func(req *http.Request) *http.Response {
			equals(t, req.URL.Path, authRepEndpoint)
			// decodes to app_id=any&app_key=key&service_id=test&service_token=any&usage[hits]=1&usage[other]=2
			expect := `app_id=any&app_key=key&service_id=test&service_token=any&usage%5Bhits%5D=1&usage%5Bother%5D=2`

			if req.URL.RawQuery != expect {
				t.Error("unexpected result in query string")
			}

			return &http.Response{
				StatusCode: 200,
				Body:       ioutil.NopCloser(bytes.NewBufferString(fake.GetAuthSuccess())),
				Header:     make(http.Header),
			}
		}),
	}
	const svcID = "test"
	c := threeScaleTestClient(t, fixture.injectClient)
	resp, err := c.AuthRep(svcID, fixture.auth, fixture.request)
	if err != nil {
		t.Error("unexpected error")
	}
	equals(t, fixture.expectResponse, resp)

}

// ******
// Helpers

// equals fails the test if exp is not equal to act.
func equals(t *testing.T, exp, act interface{}) {
	t.Helper()
	if !reflect.DeepEqual(exp, act) {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d:\n\n\texp: %#v\n\n\tgot: %#v\033[39m\n\n", filepath.Base(file), line, exp, act)
		t.Error("unexpected result when calling equals")
	}
}

// Returns a default client for testing
func threeScaleTestClient(t *testing.T, hc *http.Client) *Client {
	client, err := NewClient(defaultBackendUrl, hc)
	if err != nil {
		t.Error("failed to build test client")
	}
	return client
}

func getUsageReportXML(t *testing.T) string {
	t.Helper()
	return `<?xml version="1.0" encoding="UTF-8"?>
<status>
   <authorized>true</authorized>
   <plan>Basic</plan>
   <usage_reports>
      <usage_report metric="hits" period="minute">
         <period_start>2019-02-22 14:32:00 +0000</period_start>
         <period_end>2019-02-22 14:33:00 +0000</period_end>
         <max_value>4</max_value>
         <current_value>1</current_value>
      </usage_report>
      <usage_report metric="test_metric" period="week">
         <period_start>2019-02-18 00:00:00 +0000</period_start>
         <period_end>2019-02-25 00:00:00 +0000</period_end>
         <max_value>6</max_value>
         <current_value>0</current_value>
      </usage_report>
   </usage_reports>
</status>`
}

func getHierarchyXML(t *testing.T) string {
	t.Helper()
	return `<?xml version="1.0" encoding="UTF-8"?>
<status>
   <authorized>true</authorized>
   <plan>Basic</plan>
   <hierarchy>
      <metric name="hits" children="example sample test test" />
      <metric name="test_metric" children="" />
   </hierarchy>
</status>`
}

var extTested bool

func getExtensions(t *testing.T) map[string]string {
	t.Helper()

	// ensure we at least return the extensions the first time we get called
	if !extTested || rand.Intn(2) != 0 {
		extTested = true
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
func getExtensionsValue(t *testing.T) []string {
	t.Helper()
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
	expected := getExtensionsValue(t)

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

// ******

// *****
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

// ******
