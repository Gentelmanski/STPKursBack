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
	err := r.db.WithContext(ctx).
		Preload("Creator").
		Preload("Tags").
		Preload("Media").
		Preload("Comments", "parent_id IS NULL AND is_deleted = false").
		Preload("Comments.User").
		Preload("Comments.Replies", "is_deleted = false").
		Preload("Comments.Replies.User").
		First(&event, id).Error
	if err != nil {
		return nil, err
	}
	return &event, nil
}

func (r *EventRepository) FindAll(ctx context.Context, filter map[string]interface{}) ([]entities.Event, error) {
	var events []entities.Event

	query := r.db.WithContext(ctx).
		Preload("Creator").
		Preload("Tags").
		Where("is_active = ?", true)

	// Применяем фильтры
	if types, ok := filter["type"]; ok {
		query = query.Where("type IN ?", types)
	}
	if date, ok := filter["date"]; ok {
		if !date.(time.Time).IsZero() {
			startOfDay := date.(time.Time).Truncate(24 * time.Hour)
			endOfDay := startOfDay.Add(24 * time.Hour)
			query = query.Where("event_date BETWEEN ? AND ?", startOfDay, endOfDay)
		}
	}

	err := query.Order("created_at DESC").Find(&events).Error
	return events, err
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
