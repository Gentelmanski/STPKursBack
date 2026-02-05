package repositories

import (
	"context"
	"time"

	"auth-system/internal/application/interfaces"
	"auth-system/internal/domain/entities"

	"github.com/gosimple/slug"
	"gorm.io/gorm"
)

type EventRepository struct {
	db *gorm.DB
}

func NewEventRepository(db *gorm.DB) interfaces.EventRepository {
	return &EventRepository{db: db}
}

func (r *EventRepository) Create(ctx context.Context, event *entities.Event) error {
	return r.db.WithContext(ctx).Create(event).Error
}

func (r *EventRepository) FindByID(ctx context.Context, id uint) (*entities.Event, error) {
	var event entities.Event

	// Загружаем базовую информацию о событии
	err := r.db.WithContext(ctx).
		Table("events").
		Select("events.*, users.username, users.email, users.role, users.avatar_url").
		Joins("LEFT JOIN users ON events.creator_id = users.id").
		Where("events.id = ?", id).
		First(&event).Error

	if err != nil {
		return nil, err
	}

	// Ручной расчет количества участников
	var participantsCount int64
	r.db.WithContext(ctx).Table("event_participants").
		Where("event_id = ? AND status = 'going'", id).
		Count(&participantsCount)
	event.ParticipantsCount = int(participantsCount)

	// Преобразуем координаты из PostGIS
	var coords struct {
		Latitude  float64
		Longitude float64
	}

	r.db.WithContext(ctx).Raw(`
		SELECT 
			ST_Y(location::geometry) as latitude,
			ST_X(location::geometry) as longitude 
		FROM events WHERE id = ?
	`, id).Scan(&coords)

	event.Latitude = coords.Latitude
	event.Longitude = coords.Longitude

	return &event, nil
}

func (r *EventRepository) FindAll(ctx context.Context, filter map[string]interface{}) ([]entities.Event, error) {
	var events []entities.Event

	// Базовый запрос
	query := r.db.WithContext(ctx).
		Table("events").
		Select("events.*, users.username, users.email, users.role, users.avatar_url").
		Joins("LEFT JOIN users ON events.creator_id = users.id").
		Where("events.is_active = ?", true)

	// Фильтры
	if types, ok := filter["type"]; ok {
		query = query.Where("events.type IN ?", types)
	}
	if date, ok := filter["date"]; ok {
		if !date.(time.Time).IsZero() {
			startOfDay := date.(time.Time).Truncate(24 * time.Hour)
			endOfDay := startOfDay.Add(24 * time.Hour)
			query = query.Where("events.event_date BETWEEN ? AND ?", startOfDay, endOfDay)
		}
	}

	// Получаем события
	err := query.Order("events.created_at DESC").Find(&events).Error
	if err != nil {
		return nil, err
	}

	// Для каждого события получаем дополнительные данные
	for i := range events {
		// Количество участников
		var count int64
		r.db.WithContext(ctx).Table("event_participants").
			Where("event_id = ? AND status = 'going'", events[i].ID).
			Count(&count)
		events[i].ParticipantsCount = int(count)

		// Координаты
		var coords struct {
			Latitude  float64
			Longitude float64
		}
		r.db.WithContext(ctx).Raw(`
			SELECT 
				ST_Y(location::geometry) as latitude,
				ST_X(location::geometry) as longitude 
			FROM events WHERE id = ?
		`, events[i].ID).Scan(&coords)

		events[i].Latitude = coords.Latitude
		events[i].Longitude = coords.Longitude
	}

	return events, nil
}

func (r *EventRepository) GetEventTags(ctx context.Context, eventID uint) ([]entities.Tag, error) {
	var tags []entities.Tag

	err := r.db.WithContext(ctx).
		Table("tags").
		Select("tags.*").
		Joins("JOIN event_tags ON tags.id = event_tags.tag_id").
		Where("event_tags.event_id = ?", eventID).
		Find(&tags).Error

	return tags, err
}

func (r *EventRepository) GetEventParticipants(ctx context.Context, eventID uint) ([]entities.EventParticipant, error) {
	var participants []entities.EventParticipant

	err := r.db.WithContext(ctx).
		Table("event_participants").
		Select("event_participants.*, users.username, users.email, users.role, users.avatar_url").
		Joins("LEFT JOIN users ON event_participants.user_id = users.id").
		Where("event_participants.event_id = ?", eventID).
		Find(&participants).Error

	return participants, err
}

func (r *EventRepository) Update(ctx context.Context, event *entities.Event) error {
	return r.db.WithContext(ctx).Save(event).Error
}

func (r *EventRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&entities.Event{}, id).Error
}

func (r *EventRepository) GetByCreator(ctx context.Context, creatorID uint) ([]entities.Event, error) {
	var events []entities.Event
	err := r.db.WithContext(ctx).
		Preload("Creator").
		Preload("Tags").
		Where("creator_id = ? AND is_active = ?", creatorID, true).
		Order("created_at DESC").
		Find(&events).Error
	return events, err
}

func (r *EventRepository) GetParticipatedEvents(ctx context.Context, userID uint) ([]entities.Event, error) {
	var events []entities.Event

	// Получаем мероприятия, в которых участвует пользователь
	err := r.db.WithContext(ctx).
		Joins("JOIN event_participants ep ON events.id = ep.event_id").
		Preload("Creator").
		Preload("Tags").
		Where("ep.user_id = ? AND ep.status = 'going' AND events.is_active = ?", userID, true).
		Order("ep.joined_at DESC").
		Find(&events).Error

	return events, err
}

func (r *EventRepository) VerifyEvent(ctx context.Context, eventID uint) error {
	return r.db.WithContext(ctx).
		Model(&entities.Event{}).
		Where("id = ?", eventID).
		Update("is_verified", true).Error
}

func (r *EventRepository) RejectEvent(ctx context.Context, eventID uint, reason string) error {
	return r.db.WithContext(ctx).
		Model(&entities.Event{}).
		Where("id = ?", eventID).
		Updates(map[string]interface{}{
			"is_active":   false,
			"is_verified": false,
		}).Error
}

func (r *EventRepository) GetPendingEvents(ctx context.Context) ([]entities.Event, error) {
	var events []entities.Event
	err := r.db.WithContext(ctx).
		Preload("Creator").
		Preload("Tags").
		Where("is_verified = ? AND is_active = ?", false, true).
		Order("created_at DESC").
		Find(&events).Error
	return events, err
}

func (r *EventRepository) GetStatistics(ctx context.Context) (map[string]interface{}, error) {
	// Этот метод уже не нужен, так как статистика разбита на методы в AdminRepository
	// Оставляем пустую реализацию для совместимости
	return make(map[string]interface{}), nil
}

func (r *EventRepository) AddParticipant(ctx context.Context, eventID, userID uint) error {
	participant := &entities.EventParticipant{
		EventID:  eventID,
		UserID:   userID,
		Status:   "going",
		JoinedAt: time.Now(),
	}
	return r.db.WithContext(ctx).Create(participant).Error
}

func (r *EventRepository) RemoveParticipant(ctx context.Context, eventID, userID uint) error {
	return r.db.WithContext(ctx).
		Where("event_id = ? AND user_id = ?", eventID, userID).
		Delete(&entities.EventParticipant{}).Error
}

func (r *EventRepository) GetParticipantCount(ctx context.Context, eventID uint) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&entities.EventParticipant{}).
		Where("event_id = ? AND status = 'going'", eventID).
		Count(&count).Error
	return count, err
}

func (r *EventRepository) IsParticipant(ctx context.Context, eventID, userID uint) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&entities.EventParticipant{}).
		Where("event_id = ? AND user_id = ? AND status = 'going'", eventID, userID).
		Count(&count).Error
	return count > 0, err
}

func (r *EventRepository) AddTags(ctx context.Context, eventID uint, tags []string) error {
	// Начинаем транзакцию
	tx := r.db.WithContext(ctx).Begin()

	for _, tagName := range tags {
		var tag entities.Tag
		// Ищем или создаем тег
		if err := tx.Where("name = ?", tagName).First(&tag).Error; err != nil {
			// Тег не найден, создаем новый
			tag = entities.Tag{
				Name:      tagName,
				Slug:      slug.Make(tagName),
				CreatedAt: time.Now(),
			}
			if err := tx.Create(&tag).Error; err != nil {
				tx.Rollback()
				return err
			}
		}

		// Создаем связь события с тегом
		eventTag := entities.EventTag{
			EventID: eventID,
			TagID:   tag.ID,
		}
		if err := tx.Create(&eventTag).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit().Error
}

func (r *EventRepository) GetTopEvents(ctx context.Context, limit int) ([]map[string]interface{}, error) {
	// Этот метод теперь находится в AdminRepository
	// Оставляем пустую реализацию для совместимости
	return nil, nil
}
