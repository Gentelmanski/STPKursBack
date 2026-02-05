package repositories

import (
	"auth-system/internal/application/interfaces"

	"gorm.io/gorm"
)

// Repositories объединяет все репозитории как интерфейсы
type Repositories struct {
	User         interfaces.UserRepository
	Event        interfaces.EventRepository
	Comment      interfaces.CommentRepository
	Notification interfaces.NotificationRepository
	Admin        interfaces.AdminRepository
}

// Factory functions для создания репозиториев
func NewRepositories(db *gorm.DB) *Repositories {
	return &Repositories{
		User:         NewUserRepository(db),
		Event:        NewEventRepository(db),
		Comment:      NewCommentRepository(db),
		Notification: NewNotificationRepository(db),
		Admin:        NewAdminRepository(db),
	}
}
