package db

import (
	"log"
	"time"

	"mattb.nz/web/metrics/metrics"
)

var ua_cache = make(map[string]uint)

type UserAgent struct {
	ID        uint `gorm:"primarykey"`
	UserAgent string
}

func GetUserAgentID(userAgent string) uint {
	if id, ok := ua_cache[userAgent]; ok {
		return id
	}
	ua := UserAgent{}
	if err := DB.Where("user_agent = ?", userAgent).First(&ua).Error; err != nil {
		ua.UserAgent = userAgent
		if err := DB.Create(&ua).Error; err != nil {
			log.Printf("Could not create user agent: %v", err)
			return 0
		}
	}
	ua_cache[userAgent] = ua.ID
	return ua.ID
}

type EventLog struct {
	ID          uint `gorm:"primarykey"`
	When        time.Time
	Host        string
	Referer     string
	UserAgentID uint
	IP          string
	RawEvent    metrics.JsonEvent `gorm:"serializer:json"`
}

func init() {
	register(&EventLog{})
	register(&UserAgent{})
}
