package repo

import (
	"errors"

	"gorm.io/gorm"

	"go-gin-gorm-starter/internal/domain"
)

type UserRepo struct{ db *gorm.DB }

func NewUserRepo(db *gorm.DB) *UserRepo { return &UserRepo{db: db} }

func (r *UserRepo) Create(u *domain.User) error { return r.db.Create(u).Error }

func (r *UserRepo) FindByID(id string) (*domain.User, error) {
	var u domain.User
	err := r.db.First(&u, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &u, err
}

func (r *UserRepo) FindByEmail(email string) (*domain.User, error) {
	var u domain.User
	err := r.db.First(&u, "email = ?", email).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &u, err
}

func (r *UserRepo) List(offset, limit int) ([]domain.User, int64, error) {
	var users []domain.User
	tx := r.db.Model(&domain.User{})
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := tx.Offset(offset).Limit(limit).Order("created_at desc").Find(&users).Error; err != nil {
		return nil, 0, err
	}
	return users, total, nil
}

func (r *UserRepo) Update(u *domain.User) error { return r.db.Save(u).Error }

func (r *UserRepo) SoftDelete(id string) error {
	return r.db.Where("id = ?", id).Delete(&domain.User{}).Error
}
