package threescale

import "github.com/3scale/3scale-go-client/threescale/api"

// GetServiceID from Request
func (r Request) GetServiceID() api.Service {
	return r.Service
}

var ascendingPeriodSequence = []api.Period{api.Minute, api.Hour, api.Day, api.Week, api.Month, api.Eternity}

func GetAscendingPeriodSequence() []api.Period {
	return ascendingPeriodSequence
}
