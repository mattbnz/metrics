package db

import (
	"testing"

	"mattb.nz/web/metrics/config"
)

func Test_GetUsrAgentID(t *testing.T) {
	Init(config.Config{
		DatabaseUrl: "file::memory:?cache=shared",
	})

	id := GetUserAgentID("test")
	if id == 0 {
		t.Error("Expected non-zero ID")
	}
	id2 := GetUserAgentID("test")
	if id != id2 {
		t.Error("Expected same ID")
	}
	id3 := GetUserAgentID("test2")
	if id3 == 0 {
		t.Error("Expected non-zero ID")
	}
	if id == id3 {
		t.Error("Expected different ID")
	}

	var c int64
	if err := DB.Model(&UserAgent{}).Count(&c).Error; err != nil {
		t.Error("Error counting user agents:", err)
	}
	if c != 2 {
		t.Error("Expected 2 user agents, got", c)
	}
}
