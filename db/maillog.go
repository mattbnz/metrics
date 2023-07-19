package db

import (
	"time"
)

type MailLog struct {
	ID      uint `gorm:"primarykey"`
	When    time.Time
	Host    string
	Name    string
	Org     string
	Details string
	Msg     string
	IP      string
}

func init() {
	register(&MailLog{})
}
