package client

import (
	"errors"
	"fmt"
	"istio.io/istio/pkg/log"
	"net/url"
)

const reportEndpoint = "/transactions.xml"

//ReportAppID - Report for the Application Id authentication pattern with serviceToken
func (client *ThreeScaleClient) ReportAppID(auth TokenAuth, serviceId string, transactions ReportTransactions) (ApiResponse, error) {
	values := parseQueries(transactions, url.Values{}, transactions.Metrics, transactions.Log)

	err := auth.SetURLValues(&values)
	if err != nil {
		return ApiResponse{}, err
	}

	values.Add("service_id", serviceId)
	log.Errorf("%#v", values)

	return client.report(values)
}

//ReportUserKey - Report for the API Key authentication pattern with service token
func (client *ThreeScaleClient) ReportUserKey(auth TokenAuth, serviceId string, transactions ReportTransactions) (ApiResponse, error) {
	values := parseQueries(transactions, url.Values{}, transactions.Metrics, transactions.Log)

	err := auth.SetURLValues(&values)
	if err != nil {
		return ApiResponse{}, err
	}

	values.Add("service_id", serviceId)
	return client.report(values)
}

func (client *ThreeScaleClient) report(values url.Values) (ApiResponse, error) {
	var resp ApiResponse

	req, err := client.buildGetReq(reportEndpoint)
	if err != nil {
		return resp, errors.New(httpReqError.Error() + " for report")
	}

	req.URL.RawQuery = values.Encode()
	resp, err = client.doHttpReq(req)
	if err != nil {
		return resp, fmt.Errorf("error calling 3Scale API - %s", err.Error())
	}
	return resp, nil
}

func NewTransactionAppID(AppID string, Timestamp string, UserId string, metrics Metrics, log Log) ReportTransactions {
	return ReportTransactions{
		AppID:     AppID,
		UserKey:   "",
		UserId:    UserId,
		Timestamp: Timestamp,
		Metrics:   metrics,
		Log:       log,
	}
}

func NewTransactionUserKey(UserKey string, Timestamp string, UserId string, metrics Metrics, log Log) ReportTransactions {
	return ReportTransactions{
		AppID:     "",
		UserKey:   UserKey,
		UserId:    UserId,
		Timestamp: Timestamp,
		Metrics:   metrics,
		Log:       log,
	}
}
