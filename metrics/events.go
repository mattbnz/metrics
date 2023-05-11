package metrics

import "time"

type EventType string

const (
	EV_PAGEVIEW EventType = "pageview"
	EV_CLICK    EventType = "click"
	EV_ACTIVITY EventType = "activity"
)

type JsonEvent struct {
	Event     EventType
	SessionId string
	Target    string
	Value     string
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
	case EV_PAGEVIEW, EV_CLICK, EV_ACTIVITY:
		return true
	}
	return false
}
