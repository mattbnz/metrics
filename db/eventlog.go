package db

import (
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"
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
	Page        string // The page that triggered this event.
	Referer     string // Who sent the user to the above page.
	UserAgentID uint
	IP          string
	RawEvent    metrics.JsonEvent `gorm:"serializer:json"`
}

func (e *EventLog) PostMigrate(db *gorm.DB) error {
	done, err := GetMetadata("EL_REFERER_TO_PAGE_DONE")
	if err != nil {
		return fmt.Errorf("failed to check EventLog referer migration status: %w", err)
	}
	if done == "completed" {
		return nil
	}
	// Need to migrate current contents of 'referer' into 'page'
	log.Printf("Migrating EventLog.referer to EventLog.page...")
	if err := db.Exec("UPDATE event_logs SET page=referer, referer=''").Error; err != nil {
		return fmt.Errorf("failed to migrate EventLog referer: %w", err)
	}
	if err := SetMetadata("EL_REFERER_TO_PAGE_DONE", "completed"); err != nil {
		return fmt.Errorf("EventLog referer migration completed, but status not set: %w", err)
	}
	log.Printf("Migration of EventLog.referer to EventLog.page completed.")
	return nil
}

func init() {
	register(&EventLog{})
	register(&UserAgent{})
}
