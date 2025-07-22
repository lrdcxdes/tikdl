package models

import "gorm.io/gorm"

type GroupSettings struct {
	gorm.Model

	ChatID          int64 `gorm:"primaryKey"`
	NSFW            *bool
	Captions        *bool
	MediaGroupLimit int
	Silent          *bool
}
