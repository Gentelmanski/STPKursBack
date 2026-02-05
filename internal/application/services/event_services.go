package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"auth-system/internal/application/dto"
	appInterfaces "auth-system/internal/application/interfaces"
	"auth-system/internal/domain/entities"
)

type EventService struct {
	eventRepo        appInterfaces.EventRepository
	userRepo         appInterfaces.UserRepository
	notificationRepo appInterfaces.NotificationRepository
}

func NewEventService(eventRepo appInterfaces.EventRepository, userRepo appInterfaces.UserRepository, notificationRepo appInterfaces.NotificationRepository) *EventService {
	return &EventService{
		eventRepo:        eventRepo,
		userRepo:         userRepo,
		notificationRepo: notificationRepo,
	}
}

func (s *EventService) CreateEvent(ctx context.Context, req dto.CreateEventRequest, userID uint) (*dto.EventResponse, error) {
	event := &entities.Event{
		Title:           req.Title,
		Description:     req.Description,
		EventDate:       req.EventDate,
		Latitude:        req.Latitude,
		Longitude:       req.Longitude,
		Type:            req.Type,
		MaxParticipants: req.MaxParticipants,
		Price:           req.Price,
		Address:         req.Address,
		CreatorID:       userID,
		IsVerified:      false,
		IsActive:        true,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	if err := s.eventRepo.Create(ctx, event); err != nil {
		return nil, err
	}

	// Добавляем теги
	if len(req.Tags) > 0 {
		if err := s.eventRepo.AddTags(ctx, event.ID, req.Tags); err != nil {
			// Логируем ошибку, но не прерываем выполнение
			fmt.Printf("Failed to add tags: %v\n", err)
		}
	}

	// Notify admins
	admins, err := s.userRepo.GetAdmins(ctx)
	if err == nil && len(admins) > 0 {
		for _, admin := range admins {
			notification := &entities.Notification{
				UserID:    admin.ID,
				Message:   fmt.Sprintf("Новое мероприятие создано: %s", event.Title),
				Type:      "event_created",
				Read:      false,
				CreatedAt: time.Now(),
			}
			s.notificationRepo.Create(ctx, notification)
		}
	}

	// Получаем созданное событие со всеми данными
	createdEvent, err := s.eventRepo.FindByID(ctx, event.ID)
	if err != nil {
		return nil, err
	}

	return s.eventToDTO(createdEvent), nil
}

func (s *EventService) GetEvents(ctx context.Context, filter dto.EventFilter) ([]dto.EventResponse, error) {
	filterMap := make(map[string]interface{})
	filterMap["is_active"] = true

	if len(filter.Type) > 0 {
		filterMap["type"] = filter.Type
	}
	if !filter.Date.IsZero() {
		filterMap["date"] = filter.Date
	}

	events, err := s.eventRepo.FindAll(ctx, filterMap)
	if err != nil {
		return nil, err
	}

	response := make([]dto.EventResponse, len(events))
	for i, event := range events {
		response[i] = *s.eventToDTO(&event)
	}

	return response, nil
}

func (s *EventService) GetEventByID(ctx context.Context, id uint) (*dto.EventResponse, error) {
	event, err := s.eventRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if !event.IsActive {
		return nil, errors.New("event not found")
	}

	return s.eventToDTO(event), nil
}

func (s *EventService) UpdateEvent(ctx context.Context, id uint, req dto.UpdateEventRequest, userID uint) (*dto.EventResponse, error) {
	event, err := s.eventRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if event.CreatorID != userID {
		return nil, errors.New("not authorized to update this event")
	}

	// Update fields
	if req.Title != "" {
		event.Title = req.Title
	}
	if req.Description != "" {
		event.Description = req.Description
	}
	if !req.EventDate.IsZero() {
		event.EventDate = req.EventDate
	}
	if req.Type != "" {
		event.Type = req.Type
	}
	if req.MaxParticipants != nil {
		event.MaxParticipants = req.MaxParticipants
	}
	if req.Price >= 0 {
		event.Price = req.Price
	}
	event.UpdatedAt = time.Now()

	if err := s.eventRepo.Update(ctx, event); err != nil {
		return nil, err
	}

	return s.eventToDTO(event), nil
}

func (s *EventService) DeleteEvent(ctx context.Context, id uint, userID uint) error {
	event, err := s.eventRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	if event.CreatorID != userID {
		return errors.New("not authorized to delete this event")
	}

	event.IsActive = false
	event.UpdatedAt = time.Now()

	return s.eventRepo.Update(ctx, event)
}

func (s *EventService) Participate(ctx context.Context, eventID, userID uint) error {
	event, err := s.eventRepo.FindByID(ctx, eventID)
	if err != nil {
		return err
	}

	if !event.IsActive {
		return errors.New("event is not active")
	}

	// Check if event is full
	if event.MaxParticipants != nil {
		count, err := s.eventRepo.GetParticipantCount(ctx, eventID)
		if err != nil {
			return err
		}
		if int(count) >= *event.MaxParticipants {
			return errors.New("event is full")
		}
	}

	// Check if already participating
	isParticipant, err := s.eventRepo.IsParticipant(ctx, eventID, userID)
	if err != nil {
		return err
	}
	if isParticipant {
		return errors.New("already participating")
	}

	// Add participant
	if err := s.eventRepo.AddParticipant(ctx, eventID, userID); err != nil {
		return err
	}

	// Notify event creator
	if event.CreatorID != userID {
		notification := &entities.Notification{
			UserID:    event.CreatorID,
			Message:   fmt.Sprintf("Новый участник присоединился к вашему мероприятию: %s", event.Title),
			Type:      "participation",
			Read:      false,
			CreatedAt: time.Now(),
		}
		s.notificationRepo.Create(ctx, notification)
	}

	return nil
}

func (s *EventService) CancelParticipation(ctx context.Context, eventID, userID uint) error {
	return s.eventRepo.RemoveParticipant(ctx, eventID, userID)
}

func (s *EventService) GetUserEvents(ctx context.Context, userID uint) ([]dto.EventResponse, error) {
	events, err := s.eventRepo.GetByCreator(ctx, userID)
	if err != nil {
		return nil, err
	}

	response := make([]dto.EventResponse, len(events))
	for i, event := range events {
		response[i] = *s.eventToDTO(&event)
	}

	return response, nil
}

func (s *EventService) GetParticipatedEvents(ctx context.Context, userID uint) ([]dto.EventResponse, error) {
	events, err := s.eventRepo.GetParticipatedEvents(ctx, userID)
	if err != nil {
		return nil, err
	}

	response := make([]dto.EventResponse, len(events))
	for i, event := range events {
		response[i] = *s.eventToDTO(&event)
	}

	return response, nil
}

func (s *EventService) FilterEvents(ctx context.Context, filter dto.EventFilter) ([]dto.EventResponse, error) {
	return s.GetEvents(ctx, filter)
}

func (s *EventService) eventToDTO(event *entities.Event) *dto.EventResponse {
	// Convert tags
	tags := make([]dto.Tag, len(event.Tags))
	for i, tag := range event.Tags {
		tags[i] = dto.Tag{
			ID:   tag.ID,
			Name: tag.Name,
			Slug: tag.Slug,
		}
	}

	// Convert media
	media := make([]dto.Media, len(event.Media))
	for i, m := range event.Media {
		media[i] = dto.Media{
			ID:         m.ID,
			FileURL:    m.FileURL,
			FileType:   m.FileType,
			OrderIndex: m.OrderIndex,
		}
	}

	// Создаем UserShort (обрабатываем nil Creator)
	userShort := dto.UserShort{
		ID:       event.Creator.ID,
		Username: event.Creator.Username,
		Email:    event.Creator.Email,
		Role:     event.Creator.Role,
	}

	return &dto.EventResponse{
		ID:                event.ID,
		Title:             event.Title,
		Description:       event.Description,
		EventDate:         event.EventDate,
		Latitude:          event.Latitude,
		Longitude:         event.Longitude,
		Type:              event.Type,
		MaxParticipants:   event.MaxParticipants,
		Price:             event.Price,
		Address:           event.Address,
		IsVerified:        event.IsVerified,
		IsActive:          event.IsActive,
		CreatorID:         event.CreatorID,
		Creator:           userShort,
		ParticipantsCount: event.ParticipantsCount,
		CreatedAt:         event.CreatedAt.Format(time.RFC3339),
		UpdatedAt:         event.UpdatedAt.Format(time.RFC3339),
		Tags:              tags,
		Media:             media,
	}
}
