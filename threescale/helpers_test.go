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
