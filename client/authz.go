package client

import (
	"errors"
	"fmt"
	"net/url"
)

const authzEndpoint = "/transactions/authorize.xml"

//Authorize - Read-only operation to authorize an application in the App Id authentication pattern.
func (client *ThreeScaleClient) Authorize(appId string, serviceToken string, serviceId string, arp AuthorizeParams) (ApiResponse, error) {
	var authRepResp ApiResponse

	req, err := client.buildGetReq(authzEndpoint)
	if err != nil {
		return authRepResp, errors.New(httpReqError.Error() + " for Authorize")
	}

	values := parseQueries(arp, url.Values{}, arp.Metrics)
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

// Create valid AuthorizeParams
func NewAuthorizeParams(appKey string, referrer string, userId string) AuthorizeParams {
	return AuthorizeParams{
		AppKey:   appKey,
		Referrer: referrer,
		UserId:   userId,
		Metrics:  make(Metrics),
	}
}
