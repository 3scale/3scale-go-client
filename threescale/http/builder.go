package http

import (
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strconv"

	"github.com/3scale/3scale-go-client/threescale"

	"github.com/3scale/3scale-go-client/threescale/api"
)

type requestBuilder struct {
}

func (rb requestBuilder) build(in threescale.Request, baseURL string, kind kind) (*http.Request, error) {
	req, err := rb.kindToHTTPRequest(baseURL, kind)
	if err != nil {
		return req, err
	}

	values := rb.setValues(in, kind)

	req.Header.Set("Accept", "application/xml")
	req.URL.RawQuery = values.Encode()

	if in.Extensions != nil {
		req.Header.Set(enableExtensions, rb.encodeExtensions(in.Extensions))
	}

	return req, nil
}

func (rb requestBuilder) setValues(in threescale.Request, kind kind) url.Values {
	values := rb.joinValues(make(url.Values), rb.serviceToValues(in.Service))
	values = rb.joinValues(values, rb.authToValues(in.Auth))

	if kind == report {
		for index, transaction := range in.Transactions {
			values = rb.joinValues(values, rb.transactionToValues(index, transaction))
		}
	} else {
		// the significance of the first entry here is important to call out, since
		// since auth, authrep only handle a single transaction, any others will be discarded
		values = rb.joinValues(values, rb.metricsToValues(in.Transactions[0].Metrics))
		values = rb.joinValues(values, rb.paramsToValues(in.Transactions[0].Params))
	}
	return values
}

func (rb requestBuilder) encodeExtensions(extensions api.Extensions) string {
	var exts string

	if extensions != nil {
		for k, v := range extensions {
			// the extensions mechanism requires escaping keys and values
			// we are using QueryEscape because it escapes characters that
			// PathEscape does not and are needed to disambiguate (ie. '=').
			k = url.QueryEscape(k)
			v = url.QueryEscape(v)

			// add separator if needed
			if exts != "" {
				exts = exts + "&"
			}

			exts = exts + fmt.Sprintf("%s=%s", k, v)
		}
	}

	return exts
}

func (rb requestBuilder) kindToHTTPRequest(baseURL string, kind kind) (*http.Request, error) {
	switch kind {
	case auth:
		return http.NewRequest(http.MethodGet, baseURL+authzEndpoint, nil)
	case authRep:
		return http.NewRequest(http.MethodGet, baseURL+authRepEndpoint, nil)
	case report:
		return http.NewRequest(http.MethodPost, baseURL+reportEndpoint, nil)
	default:
		return nil, fmt.Errorf("unknown api call kind provided")
	}
}

func (rb requestBuilder) authToValues(auth api.ClientAuth) url.Values {
	values := make(url.Values)
	values.Add(string(auth.Type), auth.Value)
	return values
}

func (rb requestBuilder) metricsToValues(m api.Metrics) url.Values {
	values := make(url.Values, len(m))

	for metricName, incrementBy := range m {
		key := fmt.Sprintf("usage[%s]", metricName)
		value := strconv.Itoa(incrementBy)
		values.Add(key, value)
	}
	return values
}

// paramsToValues inspects the Params (p) for non-empty values with json tags, and appends them to returned values
func (rb requestBuilder) paramsToValues(p api.Params) url.Values {
	values := make(url.Values)

	val := reflect.ValueOf(p)
	for i := 0; i < val.Type().NumField(); i++ {
		if tag, ok := val.Type().Field(i).Tag.Lookup("json"); ok {
			if valueToAdd := val.Field(i).String(); valueToAdd != "" {
				values.Add(tag, valueToAdd)
			}
		}
	}
	return values
}

func (rb requestBuilder) serviceToValues(s api.Service) url.Values {
	return url.Values{serviceIDKey: []string{string(s)}}
}

// transactionToValues formats the values correctly for batch reporting, this differs from the expected query for
// both auth endpoints so must be dealt with accordingly
func (rb requestBuilder) transactionToValues(index int, t api.Transaction) url.Values {
	values := make(url.Values)
	paramValues := rb.paramsToValues(t.Params)

	for k, v := range paramValues {
		values.Add(fmt.Sprintf("transactions[%d][%s]", index, k), v[0])
	}

	for k, v := range t.Metrics {
		values.Add(fmt.Sprintf("transactions[%d][usage][%s]", index, k), strconv.Itoa(v))
	}

	if t.Timestamp != 0 {
		values.Add(fmt.Sprintf("transactions[%d][timestamp]", index), strconv.FormatInt(t.Timestamp, 10))
	}
	return values
}

func (rb requestBuilder) joinValues(joinExisting url.Values, to url.Values) url.Values {
	for k, v := range joinExisting {
		to[k] = v
	}
	return to
}
