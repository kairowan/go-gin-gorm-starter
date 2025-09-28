package user

import (
	"gorm.io/gorm"
	"time"
)

type UserModel struct {
	ID           string `gorm:"primaryKey;type:varchar(32)"`
	Email        string `gorm:"uniqueIndex;size:255;not null"`
	Name         string `gorm:"size:64;not null"`
	PasswordHash string `gorm:"size:100;not null"`
	Role         string `gorm:"size:16;not null;default:user"`

	CreatedAt time.Time      `gorm:"autoCreateTime"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (UserModel) TableName() string { return "users" }
