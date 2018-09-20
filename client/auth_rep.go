package client

//AuthRep is a 'one-shot' operation to authorize an application and report the associated transaction at the same time.
//The main difference between this call and the regular authorize call is that usage will be reported if the authorization is successful.
//AuthRep is the most convenient way to integrate your API with the 3scale's Service Management API.
//It does a 1:1 mapping between a request to your API, and a request to 3scale's API.
//AuthRep is not a read-only operation and will increment the values if the authorization step is a success.

import (
	"errors"
	"fmt"
	"net/url"
)

const authRepEndpoint = "/transactions/authrep.xml"

//AuthRep - Authorize & Report for the Application Id authentication pattern
func (client *ThreeScaleClient) AuthRep(appId string, serviceToken string, serviceId string, arp AuthRepParams) (ApiResponse, error) {
	var authRepResp ApiResponse

	req, err := client.buildGetReq(authRepEndpoint)
	if err != nil {
		return authRepResp, errors.New(httpReqError.Error() + " for AuthRep")
	}

	values := parseQueries(arp, url.Values{}, arp.Metrics, arp.Log)
	values.Add("app_id", appId)
	values.Add("service_token", serviceToken)
	values.Add("service_id", serviceId)

	req.URL.RawQuery = values.Encode()
	authRepRes, err := client.doHttpReq(req)
	if err != nil {
		return authRepResp, fmt.Errorf("error calling 3Scale API - %s", err.Error())
	}
	return authRepRes, nil
}

//AuthRep - Authorize & Report for the API Key authentication pattern
func (client *ThreeScaleClient) AuthRepKey(userKey string, serviceToken string, serviceId string, arp AuthRepKeyParams) (ApiResponse, error) {
	var resp ApiResponse

	req, err := client.buildGetReq(authRepEndpoint)
	if err != nil {
		return resp, errors.New(httpReqError.Error() + " for AuthRepKey")
	}

	values := parseQueries(arp, url.Values{}, arp.Metrics, arp.Log)
	values.Add("user_key", userKey)
	values.Add("service_token", serviceToken)
	values.Add("service_id", serviceId)

	req.URL.RawQuery = values.Encode()
	resp, err = client.doHttpReq(req)
	if err != nil {
		return resp, fmt.Errorf("error calling 3Scale API - %s", err.Error())
	}
	return resp, nil
}

// Create valid params for AuthRep
func NewAuthRepParams(key string, referrer string, userId string) AuthRepParams {
	return AuthRepParams{
		AuthorizeParams: AuthorizeParams{
			AppKey:   key,
			Referrer: referrer,
			UserId:   userId,
			Metrics:  make(Metrics),
		},
		Log: make(Log),
	}
}

// Create valid params for AuthRepKey
func NewAuthRepKeyParams(referrer string, userId string) AuthRepKeyParams {
	return AuthRepKeyParams{
		AuthorizeKeyParams: AuthorizeKeyParams{
			Referrer: referrer,
			UserId:   userId,
			Metrics:  make(Metrics),
		},
		Log: make(Log),
	}
}
