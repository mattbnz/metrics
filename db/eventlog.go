package db

import (
	"time"

	"mattb.nz/web/metrics/metrics"
)

type EventLog struct {
	ID       uint `gorm:"primarykey"`
	When     time.Time
	Host     string
	Referer  string
	IP       string
	RawEvent metrics.JsonEvent `gorm:"serializer:json"`
}

func init() {
	register(&EventLog{})
}
