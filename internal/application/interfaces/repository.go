package interfaces

import (
	"auth-system/internal/domain/entities"
	"context"
)

type UserRepository interface {
	Create(ctx context.Context, user *entities.User) error
	FindByID(ctx context.Context, id uint) (*entities.User, error)
	FindByEmail(ctx context.Context, email string) (*entities.User, error)
	Update(ctx context.Context, user *entities.User) error
	UpdateLastOnline(ctx context.Context, userID uint) error
	GetAll(ctx context.Context) ([]entities.User, error)
	BlockUser(ctx context.Context, userID uint) error
	UnblockUser(ctx context.Context, userID uint) error
	GetAdmins(ctx context.Context) ([]entities.User, error)
}

type EventRepository interface {
	Create(ctx context.Context, event *entities.Event) error
	FindByID(ctx context.Context, id uint) (*entities.Event, error)
	FindAll(ctx context.Context, filter map[string]interface{}) ([]entities.Event, error)
	Update(ctx context.Context, event *entities.Event) error
	Delete(ctx context.Context, id uint) error
	GetByCreator(ctx context.Context, creatorID uint) ([]entities.Event, error)
	GetParticipatedEvents(ctx context.Context, userID uint) ([]entities.Event, error)
	VerifyEvent(ctx context.Context, eventID uint) error
	RejectEvent(ctx context.Context, eventID uint, reason string) error
	GetPendingEvents(ctx context.Context) ([]entities.Event, error)
	GetStatistics(ctx context.Context) (map[string]interface{}, error)
	AddParticipant(ctx context.Context, eventID, userID uint) error
	RemoveParticipant(ctx context.Context, eventID, userID uint) error
	GetParticipantCount(ctx context.Context, eventID uint) (int64, error)
	IsParticipant(ctx context.Context, eventID, userID uint) (bool, error)
	AddTags(ctx context.Context, eventID uint, tags []string) error
	GetTopEvents(ctx context.Context, limit int) ([]map[string]interface{}, error)
}

type CommentRepository interface {
	Create(ctx context.Context, comment *entities.Comment) error
	FindByEventID(ctx context.Context, eventID uint) ([]entities.Comment, error)
	FindByID(ctx context.Context, id uint) (*entities.Comment, error)
	Update(ctx context.Context, comment *entities.Comment) error
	SoftDelete(ctx context.Context, id uint) error
	Vote(ctx context.Context, commentID, userID uint, voteType string) error
	GetVote(ctx context.Context, commentID, userID uint) (*entities.CommentVote, error)
	GetScore(ctx context.Context, commentID uint) (int, error)
	DeleteVote(ctx context.Context, commentID, userID uint) error
	UpdateVote(ctx context.Context, vote *entities.CommentVote) error
	CreateVote(ctx context.Context, vote *entities.CommentVote) error
}

type NotificationRepository interface {
	Create(ctx context.Context, notification *entities.Notification) error
	FindByUserID(ctx context.Context, userID uint) ([]entities.Notification, error)
	MarkAsRead(ctx context.Context, notificationID, userID uint) error
	MarkAllAsRead(ctx context.Context, userID uint) error
	GetUnreadCount(ctx context.Context, userID uint) (int64, error)
}

type AdminRepository interface {
	LogAction(ctx context.Context, action *entities.AdminAction) error
	GetActions(ctx context.Context) ([]entities.AdminAction, error)
	GetUserStats(ctx context.Context) (map[string]interface{}, error)
	GetEventStats(ctx context.Context) (map[string]interface{}, error)
	GetTopEvents(ctx context.Context, limit int) ([]map[string]interface{}, error)
}
