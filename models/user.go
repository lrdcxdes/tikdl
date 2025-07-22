package models

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	gorm.Model

	UserID   int64     `gorm:"primaryKey"`
	LastUsed time.Time `gorm:"autoCreateTime"`
}
