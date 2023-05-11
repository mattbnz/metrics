package db

import (
	"testing"
	"time"

	"mattb.nz/web/metrics/config"
	"mattb.nz/web/metrics/metrics"
)

func Test_DB(t *testing.T) {
	Init(config.Config{
		DatabaseUrl: "file::memory:?cache=shared",
	})

	event := EventLog{
		Host:     "test.com",
		When:     time.Now(),
		IP:       "10.10.10.10",
		RawEvent: metrics.JsonEvent{Event: metrics.EV_PAGEVIEW},
	}
	if err := Create(&event).Error; err != nil {
		t.Error("Expected no error, got", err)
	}

	back := EventLog{}
	if err := First(&back).Error; err != nil {
		t.Error("Expected no error, got", err)
	}
	if back.ID != event.ID {
		t.Error("Expected ", event.ID, " got", back.ID)
	}
	if back.Host != event.Host {
		t.Error("Expected ", event.Host, " got", back.Host)
	}
	if !back.When.Equal(event.When) {
		t.Error("Expected ", event.When, " got", back.When)
	}
	if back.RawEvent != event.RawEvent {
		t.Error("Expected ", event.RawEvent, " got", back.RawEvent)
	}
}

// Check the no-op methods don't cause issues when we have no DB.
func Test_NoDB(t *testing.T) {
	DB = nil

	if err := Create(&EventLog{}).Error; err != nil {
		t.Error("Expected no error, got", err)
	}
	e := EventLog{}
	if err := Find(&e).Error; err != nil {
		t.Error("Expected no error, got", err)
	}
	if err := First(&e).Error; err != nil {
		t.Error("Expected no error, got", err)
	}
}
