package threescale

import (
	"testing"

	"github.com/3scale/3scale-go-client/threescale/api"
)

func TestRequest_GetServiceID(t *testing.T) {
	const expectService = "testing"

	r := Request{
		Auth:         api.ClientAuth{},
		Extensions:   nil,
		Service:      expectService,
		Transactions: nil,
	}

	if r.GetServiceID() != expectService {
		t.Errorf("expected %s but got %s", expectService, r.GetServiceID())
	}
}

func TestFormatTimestamp(t *testing.T) {
	const expect = "2020-03-10 11:31:31 +0000"
	timestamp := int64(1583839891)
	got := FormatTimestamp(timestamp)

	if expect != got {
		t.Errorf("failed to convert timestamp, wanted %s, but got %s", expect, got)
	}
}
