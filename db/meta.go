package db

import (
	"errors"

	"gorm.io/gorm"
)

type Meta struct {
	ID    uint `gorm:"primarykey"`
	Key   string
	Value string
}

func GetMetadata(key string) (string, error) {
	m := Meta{}
	if err := DB.Where("key = ?", key).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil
		}
		return "", err
	}
	return m.Value, nil
}

func SetMetadata(key, value string) error {
	m := Meta{Key: key, Value: value}
	if err := DB.Create(&m).Error; err != nil {
		return err
	}
	return nil
}
