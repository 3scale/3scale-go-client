package internal

import (
	"encoding/xml"
)

// AuthResponseXML formatted response from backend API for Authorize and AuthRep
type AuthResponseXML struct {
	Name         xml.Name     `xml:",any"`
	Authorized   bool         `xml:"authorized,omitempty"`
	Reason       string       `xml:"reason,omitempty"`
	Code         string       `xml:"code,attr,omitempty"`
	Hierarchy    HierarchyXML `xml:"hierarchy"`
	UsageReports struct {
		Reports []UsageReportXML `xml:"usage_report"`
	} `xml:"usage_reports"`
}

// HierarchyXML encapsulates the return value when using "hierarchy" extension
type HierarchyXML struct {
	Metric []struct {
		Name     string `xml:"name,attr"`
		Children string `xml:"children,attr"`
	} `xml:"metric"`
}

// UsageReportXML captures the XML response for rate limiting details
type UsageReportXML struct {
	Metric       string `xml:"metric,attr"`
	Period       string `xml:"period,attr"`
	PeriodStart  string `xml:"period_start"`
	PeriodEnd    string `xml:"period_end"`
	MaxValue     int    `xml:"max_value"`
	CurrentValue int    `xml:"current_value"`
}

// ReportErrorXML captures the XML response from Report endpoint when not status 202
type ReportErrorXML struct {
	XMLName xml.Name `xml:"error"`
	Text    string   `xml:",chardata"`
	Code    string   `xml:"code,attr"`
}
