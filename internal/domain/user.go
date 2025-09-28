package domain

import "time"

type User struct {
	ID           string     `gorm:"primaryKey;size:36" json:"id"`
	Email        string     `gorm:"uniqueIndex;size:191" json:"email"`
	Name         string     `gorm:"size:64" json:"name"`
	PasswordHash string     `gorm:"size:191" json:"-"`
	Role         string     `gorm:"size:16" json:"role"` // "user"/"admin"
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
	DeletedAt    *time.Time `gorm:"index" json:"-"`
}

type UserRepository interface {
	Create(u *User) error
	FindByID(id string) (*User, error)
	FindByEmail(email string) (*User, error)
	List(offset, limit int) ([]User, int64, error)
	Update(u *User) error
	SoftDelete(id string) error
}
