package client

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"testing"
)

func TestAuthRep(t *testing.T) {
	fakeAppId, fakeServiceToken, fakeServiceId := "appId12345", "servicetoken54321", "555000"
	authRepInputs := []struct {
		appId, svcToken, svcId string
		expectErr              bool
		expectSuccess          bool
		expectReason           string
		expectStatus           int
		expectParamLength      int
		buildParams            func() AuthRepParams
	}{
		{
			appId:             fakeAppId,
			svcId:             fakeServiceId,
			svcToken:          fakeServiceToken,
			expectSuccess:     true,
			expectStatus:      200,
			expectParamLength: 4,
			buildParams:       func() AuthRepParams { return NewAuthRepParams("example", "", "") },
		},
		{
			appId:             "failme",
			svcId:             fakeServiceId,
			svcToken:          fakeServiceToken,
			expectErr:         true,
			expectSuccess:     false,
			expectStatus:      200,
			expectParamLength: 3,
			buildParams:       func() AuthRepParams { return AuthRepParams{} },
		},
	}
	for _, input := range authRepInputs {
		httpClient := NewTestClient(func(req *http.Request) *http.Response {
			equals(t, req.URL.Path, authRepEndpoint)

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
				Body:       ioutil.NopCloser(bytes.NewBufferString(getAuthSuccess())),
				Header:     make(http.Header),
			}
		})
		c := threeScaleTestClient(httpClient)
		resp, err := c.AuthRep(input.appId, input.svcToken, input.svcId, input.buildParams())
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
