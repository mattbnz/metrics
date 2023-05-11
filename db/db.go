package db

import (
	"fmt"
	"log"
	"os"
	"reflect"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"mattb.nz/web/metrics/config"
)

var DB *gorm.DB
var models []interface{}

func Init(config config.Config) error {
	l := logger.New(log.New(os.Stdout, "\r\n", log.LstdFlags), logger.Config{
		SlowThreshold:             200 * time.Millisecond,
		LogLevel:                  logger.Warn,
		IgnoreRecordNotFoundError: true,
		Colorful:                  true,
	})
	db, err := gorm.Open(sqlite.Open(config.DatabaseUrl), &gorm.Config{Logger: l})
	if err != nil {
		return fmt.Errorf("could not open database: %w", err)
	}
	if err := automigrate(db); err != nil {
		return fmt.Errorf("could not automigrate: %w", err)
	}
	DB = db
	return nil
}

func register(model interface{}) {
	models = append(models, model)
}

func automigrate(db *gorm.DB) error {
	for _, model := range models {
		if err := db.AutoMigrate(model); err != nil {
			return fmt.Errorf("could not automigrate %s: %w", reflect.TypeOf(model), err)
		}
		log.Printf("Automigrated %s", reflect.TypeOf(model))
	}
	return nil
}

// Wrappers for gorm.DB methods to no-op if DB is not available
func Create(value interface{}) *gorm.DB {
	if DB == nil {
		return &gorm.DB{}
	}
	return DB.Create(value)
}

func Find(out interface{}, where ...interface{}) *gorm.DB {
	if DB == nil {
		return &gorm.DB{}
	}
	return DB.Find(out, where...)
}

func First(out interface{}, where ...interface{}) *gorm.DB {
	if DB == nil {
		return &gorm.DB{}
	}
	return DB.First(out, where...)
}
