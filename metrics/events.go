package metrics

import "time"

type EventType string

const (
	EV_PAGEVIEW EventType = "pageview"
	EV_CLICK    EventType = "click"
	EV_ACTIVITY EventType = "activity"
	EV_CONTEXT  EventType = "context"
	EV_VITALS   EventType = "vitals"
	EV_EMAIL    EventType = "email"
)

type JsonEvent struct {
	Event     EventType `json:",omitempty"`
	JSVersion string    `json:",omitempty"`
	SessionId string    `json:",omitempty"`
	Page      string    `json:",omitempty"` // Page triggering the event
	Referer   string    `json:",omitempty"` // Who sent user to that page.
	LoadTime  float64   `json:",omitempty"`
	// Data for EV_CLICK style events
	Target string `json:",omitempty"`
	Value  string `json:",omitempty"`
	// Data for EV_ACTIVITY style events
	ScrollPerc string `json:",omitempty"`
	// Web vitals metrics for EV_VITALS style events
	LCP            float64 `json:",omitempty"`
	FID            float64 `json:",omitempty"`
	CLS            float64 `json:",omitempty"`
	NavigationType string  `json:",omitempty"`
}

type Event struct {
	JsonEvent

	Timestamp time.Time
}

// Live view of site metrics
type SiteData struct {
	EventCount map[EventType]uint // since program start
}

var Sites = make(map[string]*SiteData)

func GetSiteData(host string) *SiteData {
	if _, ok := Sites[host]; !ok {
		Sites[host] = &SiteData{EventCount: make(map[EventType]uint)}
	}
	return Sites[host]
}

func IsKnownEvent(event EventType) bool {
	switch event {
	case EV_PAGEVIEW, EV_CLICK, EV_ACTIVITY, EV_CONTEXT, EV_VITALS, EV_EMAIL:
		return true
	}
	return false
}
