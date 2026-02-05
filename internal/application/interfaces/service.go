package interfaces

import (
	"auth-system/internal/application/dto"
	"context"
)

type AuthService interface {
	Register(ctx context.Context, req dto.RegisterRequest) (*dto.AuthResponse, error)
	Login(ctx context.Context, req dto.LoginRequest) (*dto.AuthResponse, error)
	GetProfile(ctx context.Context, userID uint) (*dto.UserResponse, error)
	UpdateLastOnline(ctx context.Context, userID uint) error
}

type EventService interface {
	CreateEvent(ctx context.Context, req dto.CreateEventRequest, userID uint) (*dto.EventResponse, error)
	GetEvents(ctx context.Context, filter dto.EventFilter) ([]dto.EventResponse, error)
	GetEventByID(ctx context.Context, id uint) (*dto.EventResponse, error)
	UpdateEvent(ctx context.Context, id uint, req dto.UpdateEventRequest, userID uint) (*dto.EventResponse, error)
	DeleteEvent(ctx context.Context, id uint, userID uint) error
	Participate(ctx context.Context, eventID, userID uint) error
	CancelParticipation(ctx context.Context, eventID, userID uint) error
	GetUserEvents(ctx context.Context, userID uint) ([]dto.EventResponse, error)
	GetParticipatedEvents(ctx context.Context, userID uint) ([]dto.EventResponse, error)
	FilterEvents(ctx context.Context, filter dto.EventFilter) ([]dto.EventResponse, error)
}

type CommentService interface {
	CreateComment(ctx context.Context, req dto.CreateCommentRequest, eventID, userID uint) (*dto.CommentResponse, error)
	GetComments(ctx context.Context, eventID uint) ([]dto.CommentResponse, error)
	UpdateComment(ctx context.Context, commentID uint, req dto.UpdateCommentRequest, userID uint) (*dto.CommentResponse, error)
	DeleteComment(ctx context.Context, commentID, userID uint) error
	VoteComment(ctx context.Context, commentID, userID uint, voteType string) (*dto.CommentResponse, error)
}

type NotificationService interface {
	GetNotifications(ctx context.Context, userID uint) ([]dto.NotificationResponse, error)
	MarkAsRead(ctx context.Context, notificationID, userID uint) error
	MarkAllAsRead(ctx context.Context, userID uint) error
	CreateNotification(ctx context.Context, userID uint, message, notificationType string) error
	GetUnreadCount(ctx context.Context, userID uint) (int64, error) // Добавили этот метод
}

type AdminService interface {
	GetStatistics(ctx context.Context) (*dto.StatisticsResponse, error)
	VerifyEvent(ctx context.Context, eventID, adminID uint) error
	RejectEvent(ctx context.Context, eventID, adminID uint, reason string) error
	DeleteEvent(ctx context.Context, eventID, adminID uint) error
	BlockUser(ctx context.Context, userID, adminID uint) error
	UnblockUser(ctx context.Context, userID, adminID uint) error
	DeleteComment(ctx context.Context, commentID, adminID uint) error
	GetAllEvents(ctx context.Context) ([]dto.EventResponse, error)
	GetAllUsers(ctx context.Context) ([]dto.UserResponse, error)
	GetPendingEvents(ctx context.Context) ([]dto.EventResponse, error)
}
