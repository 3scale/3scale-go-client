package client

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/3scale/3scale-go-client/fake"
)

func TestReportAppID(t *testing.T) {
	fakeServiceId := "555000"
	auth := TokenAuth{
		Type:  "service_token",
		Value: "servicetoken54321",
	}

	authInputs := []struct {
		svcId             string
		auth              TokenAuth
		extensions        map[string]string
		expectErr         bool
		expectSuccess     bool
		expectReason      string
		expectStatus      int
		expectParamLength int
		buildParams       func() ReportTransactions
	}{
		{
			svcId:             fakeServiceId,
			auth:              auth,
			expectSuccess:     true,
			expectStatus:      200,
			expectParamLength: 5,
			buildParams: func() ReportTransactions {
				p := NewTransactionAppID("valid", "", "", make(Metrics), nil)
				p.Metrics.Add("hits", 1)
				p.Metrics.Add("test", 1)
				return p
			},
		},
		{
			svcId: fakeServiceId,
			auth: TokenAuth{
				Type:  "service_token",
				Value: "servicetoken54321",
			},
			expectSuccess:     true,
			expectStatus:      200,
			expectParamLength: 3,
			buildParams:       func() ReportTransactions { return NewTransactionAppID("valid", "", "", make(Metrics), nil) },
		},
		{
			svcId: fakeServiceId,
			auth: TokenAuth{
				Type:  "service_token",
				Value: "servicetoken54321",
			},
			expectErr:         true,
			expectSuccess:     false,
			expectStatus:      200,
			expectParamLength: 3,
			buildParams:       func() ReportTransactions { return NewTransactionAppID("failme", "", "", make(Metrics), nil) },
		},
	}
	for _, input := range authInputs {
		httpClient := NewTestClient(func(req *http.Request) *http.Response {
			equals(t, req.URL.Path, reportEndpoint)
			params := req.URL.Query()
			if input.expectParamLength != len(params) {
				t.Fatalf("unexpected param length, expect %d got  %d", input.expectParamLength, len(params))
			}

			queryAppId := params["app_id"][0]

			if queryAppId == "failme" {
				return &http.Response{
					StatusCode: 200,
					Body:       ioutil.NopCloser(bytes.NewBufferString("Some invalid xml")),
					Header:     make(http.Header),
				}
			}

			return &http.Response{
				StatusCode: 200,
				Body:       ioutil.NopCloser(bytes.NewBufferString(fake.GetAuthSuccess())),
				Header:     make(http.Header),
			}
		})
		c := threeScaleTestClient(httpClient)
		resp, err := c.ReportAppID(input.auth, input.svcId, input.buildParams(), input.extensions)
		if input.expectErr && err != nil {
			continue
		}

		if err != nil {
			t.Fatal(err.Error())
		}
		if input.expectSuccess != resp.Success {
			t.Fatalf("unexpected auth response returned")
		}
		if input.expectStatus != resp.StatusCode {
			t.Fatalf("unexpected status code")
		}
		if !input.expectSuccess {
			if input.expectReason != resp.Reason {
				t.Fatalf("unexpected xml parsing")
			}
		}
	}
}
