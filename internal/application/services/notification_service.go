package services

import (
	"context"
	"time"

	"auth-system/internal/application/dto"
	appInterfaces "auth-system/internal/application/interfaces"
	"auth-system/internal/domain/entities"
)

type NotificationService struct {
	notificationRepo appInterfaces.NotificationRepository
}

func NewNotificationService(notificationRepo appInterfaces.NotificationRepository) *NotificationService {
	return &NotificationService{notificationRepo: notificationRepo}
}

func (s *NotificationService) GetNotifications(ctx context.Context, userID uint) ([]dto.NotificationResponse, error) {
	notifications, err := s.notificationRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	response := make([]dto.NotificationResponse, len(notifications))
	for i, notification := range notifications {
		response[i] = dto.NotificationResponse{
			ID:        notification.ID,
			UserID:    notification.UserID,
			Message:   notification.Message,
			Type:      notification.Type,
			Read:      notification.Read,
			CreatedAt: notification.CreatedAt.Format(time.RFC3339),
		}
	}

	return response, nil
}

func (s *NotificationService) MarkAsRead(ctx context.Context, notificationID, userID uint) error {
	return s.notificationRepo.MarkAsRead(ctx, notificationID, userID)
}

func (s *NotificationService) MarkAllAsRead(ctx context.Context, userID uint) error {
	return s.notificationRepo.MarkAllAsRead(ctx, userID)
}

func (s *NotificationService) CreateNotification(ctx context.Context, userID uint, message, notificationType string) error {
	notification := &entities.Notification{
		UserID:    userID,
		Message:   message,
		Type:      notificationType,
		Read:      false,
		CreatedAt: time.Now(),
	}

	return s.notificationRepo.Create(ctx, notification)
}

func (s *NotificationService) GetUnreadCount(ctx context.Context, userID uint) (int64, error) {
	return s.notificationRepo.GetUnreadCount(ctx, userID)
}
