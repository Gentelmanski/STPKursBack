package services

import (
	"context"
	"errors"
	"time"

	"auth-system/internal/application/dto"
	appInterfaces "auth-system/internal/application/interfaces"
	"auth-system/internal/domain/entities"
)

type AdminService struct {
	adminRepo        appInterfaces.AdminRepository
	eventRepo        appInterfaces.EventRepository
	userRepo         appInterfaces.UserRepository
	commentRepo      appInterfaces.CommentRepository
	notificationRepo appInterfaces.NotificationRepository
}

func NewAdminService(
	adminRepo appInterfaces.AdminRepository,
	eventRepo appInterfaces.EventRepository,
	userRepo appInterfaces.UserRepository,
	commentRepo appInterfaces.CommentRepository,
	notificationRepo appInterfaces.NotificationRepository,
) *AdminService {
	return &AdminService{
		adminRepo:        adminRepo,
		eventRepo:        eventRepo,
		userRepo:         userRepo,
		commentRepo:      commentRepo,
		notificationRepo: notificationRepo,
	}
}

func (s *AdminService) GetStatistics(ctx context.Context) (*dto.StatisticsResponse, error) {
	// Получаем статистику пользователей
	userStats, err := s.adminRepo.GetUserStats(ctx)
	if err != nil {
		return nil, err
	}

	// Получаем статистику мероприятий
	eventStats, err := s.adminRepo.GetEventStats(ctx)
	if err != nil {
		return nil, err
	}

	// Получаем топ мероприятий
	topEvents, err := s.adminRepo.GetTopEvents(ctx, 10)
	if err != nil {
		return nil, err
	}

	// Получаем мероприятия на верификацию
	pendingEvents, err := s.eventRepo.GetPendingEvents(ctx)
	if err != nil {
		return nil, err
	}

	// Преобразуем pendingEvents в DTO
	pendingEventsDTO := make([]dto.EventResponse, len(pendingEvents))
	for i, event := range pendingEvents {
		pendingEventsDTO[i] = *s.eventToDTO(&event)
	}

	// Преобразуем topEvents
	topEventsDTO := make([]dto.TopEventResponse, len(topEvents))
	for i, topEvent := range topEvents {
		topEventsDTO[i] = dto.TopEventResponse{
			EventID:      uint(topEvent["event_id"].(int64)),
			Title:        topEvent["title"].(string),
			Participants: topEvent["participants"].(int64),
		}
	}

	// Создаем общую статистику
	totalUsers := userStats["total_users"].(int64)
	totalEvents := eventStats["total_events"].(int64)
	activeEvents := eventStats["active_events"].(int64)
	verifiedEvents := eventStats["verified_events"].(int64)
	totalComments := eventStats["total_comments"].(int64)
	todayRegistrations := userStats["today_registrations"].(int64)
	onlineUsers := userStats["online_users"].(int64)

	return &dto.StatisticsResponse{
		TotalUsers:         totalUsers,
		TotalEvents:        totalEvents,
		ActiveEvents:       activeEvents,
		VerifiedEvents:     verifiedEvents,
		TotalComments:      totalComments,
		TodayRegistrations: todayRegistrations,
		OnlineUsers:        onlineUsers,
		TopEvents:          topEventsDTO,
		PendingEvents:      pendingEventsDTO,
	}, nil
}

func (s *AdminService) VerifyEvent(ctx context.Context, eventID, adminID uint) error {
	event, err := s.eventRepo.FindByID(ctx, eventID)
	if err != nil {
		return errors.New("event not found")
	}

	if !event.IsActive {
		return errors.New("event is not active")
	}

	event.IsVerified = true
	if err := s.eventRepo.Update(ctx, event); err != nil {
		return err
	}

	// Логируем действие
	action := &entities.AdminAction{
		AdminID:     adminID,
		ActionType:  "verify_event",
		TargetID:    eventID,
		TargetType:  "event",
		PerformedAt: time.Now(),
	}
	s.adminRepo.LogAction(ctx, action)

	// Уведомляем создателя мероприятия
	message := "Ваше мероприятие \"" + event.Title + "\" было верифицировано администратором"
	s.notificationRepo.Create(ctx, &entities.Notification{
		UserID:    event.CreatorID,
		Message:   message,
		Type:      "event_verified",
		Read:      false,
		CreatedAt: time.Now(),
	})

	return nil
}

func (s *AdminService) RejectEvent(ctx context.Context, eventID, adminID uint, reason string) error {
	event, err := s.eventRepo.FindByID(ctx, eventID)
	if err != nil {
		return errors.New("event not found")
	}

	if !event.IsActive {
		return errors.New("event is not active")
	}

	event.IsActive = false
	if err := s.eventRepo.Update(ctx, event); err != nil {
		return err
	}

	// Логируем действие
	action := &entities.AdminAction{
		AdminID:     adminID,
		ActionType:  "reject_event",
		TargetID:    eventID,
		TargetType:  "event",
		Reason:      reason,
		PerformedAt: time.Now(),
	}
	s.adminRepo.LogAction(ctx, action)

	// Уведомляем создателя мероприятия
	message := "Ваше мероприятие \"" + event.Title + "\" было отклонено администратором. Причина: " + reason
	s.notificationRepo.Create(ctx, &entities.Notification{
		UserID:    event.CreatorID,
		Message:   message,
		Type:      "event_rejected",
		Read:      false,
		CreatedAt: time.Now(),
	})

	return nil
}

func (s *AdminService) DeleteEvent(ctx context.Context, eventID, adminID uint) error {
	event, err := s.eventRepo.FindByID(ctx, eventID)
	if err != nil {
		return errors.New("event not found")
	}

	event.IsActive = false
	if err := s.eventRepo.Update(ctx, event); err != nil {
		return err
	}

	// Логируем действие
	action := &entities.AdminAction{
		AdminID:     adminID,
		ActionType:  "delete_event",
		TargetID:    eventID,
		TargetType:  "event",
		PerformedAt: time.Now(),
	}
	s.adminRepo.LogAction(ctx, action)

	// Уведомляем создателя мероприятия
	message := "Ваше мероприятие \"" + event.Title + "\" было удалено администратором"
	s.notificationRepo.Create(ctx, &entities.Notification{
		UserID:    event.CreatorID,
		Message:   message,
		Type:      "event_deleted",
		Read:      false,
		CreatedAt: time.Now(),
	})

	return nil
}

func (s *AdminService) BlockUser(ctx context.Context, userID, adminID uint) error {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return errors.New("user not found")
	}

	if user.IsBlocked {
		return errors.New("user is already blocked")
	}

	user.IsBlocked = true
	if err := s.userRepo.Update(ctx, user); err != nil {
		return err
	}

	// Логируем действие
	action := &entities.AdminAction{
		AdminID:     adminID,
		ActionType:  "block_user",
		TargetID:    userID,
		TargetType:  "user",
		PerformedAt: time.Now(),
	}
	s.adminRepo.LogAction(ctx, action)

	// Уведомляем пользователя
	message := "Ваш аккаунт был заблокирован администратором"
	s.notificationRepo.Create(ctx, &entities.Notification{
		UserID:    userID,
		Message:   message,
		Type:      "system",
		Read:      false,
		CreatedAt: time.Now(),
	})

	return nil
}

func (s *AdminService) UnblockUser(ctx context.Context, userID, adminID uint) error {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return errors.New("user not found")
	}

	if !user.IsBlocked {
		return errors.New("user is not blocked")
	}

	user.IsBlocked = false
	if err := s.userRepo.Update(ctx, user); err != nil {
		return err
	}

	// Логируем действие
	action := &entities.AdminAction{
		AdminID:     adminID,
		ActionType:  "unblock_user",
		TargetID:    userID,
		TargetType:  "user",
		PerformedAt: time.Now(),
	}
	s.adminRepo.LogAction(ctx, action)

	return nil
}

func (s *AdminService) DeleteComment(ctx context.Context, commentID, adminID uint) error {
	comment, err := s.commentRepo.FindByID(ctx, commentID)
	if err != nil {
		return errors.New("comment not found")
	}

	if comment.IsDeleted {
		return errors.New("comment is already deleted")
	}

	comment.IsDeleted = true
	comment.UpdatedAt = time.Now()
	if err := s.commentRepo.Update(ctx, comment); err != nil {
		return err
	}

	// Логируем действие
	action := &entities.AdminAction{
		AdminID:     adminID,
		ActionType:  "delete_comment",
		TargetID:    commentID,
		TargetType:  "comment",
		PerformedAt: time.Now(),
	}
	s.adminRepo.LogAction(ctx, action)

	// Уведомляем автора комментария
	message := "Ваш комментарий был удален администратором"
	s.notificationRepo.Create(ctx, &entities.Notification{
		UserID:    comment.UserID,
		Message:   message,
		Type:      "system",
		Read:      false,
		CreatedAt: time.Now(),
	})

	return nil
}

func (s *AdminService) GetAllEvents(ctx context.Context) ([]dto.EventResponse, error) {
	events, err := s.eventRepo.FindAll(ctx, map[string]interface{}{})
	if err != nil {
		return nil, err
	}

	response := make([]dto.EventResponse, len(events))
	for i, event := range events {
		response[i] = *s.eventToDTO(&event)
	}

	return response, nil
}

func (s *AdminService) GetAllUsers(ctx context.Context) ([]dto.UserResponse, error) {
	users, err := s.userRepo.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	response := make([]dto.UserResponse, len(users))
	for i, user := range users {
		response[i] = dto.UserResponse{
			ID:         user.ID,
			Username:   user.Username,
			Email:      user.Email,
			Role:       user.Role,
			AvatarURL:  user.AvatarURL,
			IsBlocked:  user.IsBlocked,
			LastOnline: user.LastOnline.Format(time.RFC3339),
			CreatedAt:  user.CreatedAt.Format(time.RFC3339),
		}
	}

	return response, nil
}

func (s *AdminService) GetPendingEvents(ctx context.Context) ([]dto.EventResponse, error) {
	events, err := s.eventRepo.GetPendingEvents(ctx)
	if err != nil {
		return nil, err
	}

	response := make([]dto.EventResponse, len(events))
	for i, event := range events {
		response[i] = *s.eventToDTO(&event)
	}

	return response, nil
}

func (s *AdminService) eventToDTO(event *entities.Event) *dto.EventResponse {
	// Преобразуем теги
	tags := make([]dto.Tag, len(event.Tags))
	for i, tag := range event.Tags {
		tags[i] = dto.Tag{
			ID:   tag.ID,
			Name: tag.Name,
			Slug: tag.Slug,
		}
	}

	// Преобразуем медиа
	media := make([]dto.Media, len(event.Media))
	for i, m := range event.Media {
		media[i] = dto.Media{
			ID:         m.ID,
			FileURL:    m.FileURL,
			FileType:   m.FileType,
			OrderIndex: m.OrderIndex,
		}
	}

	return &dto.EventResponse{
		ID:              event.ID,
		Title:           event.Title,
		Description:     event.Description,
		EventDate:       event.EventDate,
		Latitude:        event.Latitude,
		Longitude:       event.Longitude,
		Type:            event.Type,
		MaxParticipants: event.MaxParticipants,
		Price:           event.Price,
		Address:         event.Address,
		IsVerified:      event.IsVerified,
		IsActive:        event.IsActive,
		CreatorID:       event.CreatorID,
		Creator: dto.UserShort{
			ID:       event.Creator.ID,
			Username: event.Creator.Username,
			Email:    event.Creator.Email,
			Role:     event.Creator.Role,
		},
		ParticipantsCount: event.ParticipantsCount,
		CreatedAt:         event.CreatedAt.Format(time.RFC3339),
		UpdatedAt:         event.UpdatedAt.Format(time.RFC3339),
		Tags:              tags,
		Media:             media,
	}
}
