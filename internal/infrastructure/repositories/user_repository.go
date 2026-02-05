package repositories

import (
	"context"
	"time"

	"auth-system/internal/application/interfaces"
	"auth-system/internal/domain/entities"

	"gorm.io/gorm"
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) interfaces.UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, user *entities.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *UserRepository) FindByID(ctx context.Context, id uint) (*entities.User, error) {
	var user entities.User
	err := r.db.WithContext(ctx).First(&user, id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*entities.User, error) {
	var user entities.User
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) Update(ctx context.Context, user *entities.User) error {
	return r.db.WithContext(ctx).Save(user).Error
}

func (r *UserRepository) UpdateLastOnline(ctx context.Context, userID uint) error {
	return r.db.WithContext(ctx).Model(&entities.User{}).
		Where("id = ?", userID).
		Update("last_online", time.Now()).Error
}

func (r *UserRepository) GetAll(ctx context.Context) ([]entities.User, error) {
	var users []entities.User
	err := r.db.WithContext(ctx).
		Select("id, username, email, role, created_at, is_blocked, last_online").
		Order("created_at DESC").
		Find(&users).Error
	return users, err
}

func (r *UserRepository) BlockUser(ctx context.Context, userID uint) error {
	return r.db.WithContext(ctx).Model(&entities.User{}).
		Where("id = ?", userID).
		Update("is_blocked", true).Error
}

func (r *UserRepository) UnblockUser(ctx context.Context, userID uint) error {
	return r.db.WithContext(ctx).Model(&entities.User{}).
		Where("id = ?", userID).
		Update("is_blocked", false).Error
}

func (r *UserRepository) GetAdmins(ctx context.Context) ([]entities.User, error) {
	var users []entities.User
	err := r.db.WithContext(ctx).
		Where("role = ?", "admin").
		Find(&users).Error
	return users, err
}
