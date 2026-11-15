package repositories

import (
	"context"
	"errors"
	"fmt"

	"auth-system/internal/application/interfaces"
	"auth-system/internal/domain/entities"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type EventMediaRepository struct{ db *gorm.DB }

func NewEventMediaRepository(db *gorm.DB) *EventMediaRepository {
	return &EventMediaRepository{db: db}
}

func (r *EventMediaRepository) WithEventLock(ctx context.Context, eventID uint, fn func(interfaces.EventMediaTransaction) error) error {
	readyToCommit := false
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var event entities.Event
		err := tx.Table("events").Select("id, creator_id, is_active").
			Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", eventID).Take(&event).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return interfaces.ErrMediaEventNotFound
		}
		if err != nil {
			return err
		}
		if err := fn(&eventMediaTransaction{db: tx, event: &event}); err != nil {
			return err
		}
		readyToCommit = true
		return nil
	})
	if err != nil && readyToCommit {
		return fmt.Errorf("%w: %v", interfaces.ErrMediaCommitUnknown, err)
	}
	return err
}

// Reconciliation must acquire the same event lock: an ordinary SELECT could
// observe an absent row while the original COMMIT is still in progress.
func (r *EventMediaRepository) FindByKey(ctx context.Context, eventID uint, key string) (*entities.EventMedia, error) {
	var found *entities.EventMedia
	err := r.WithEventLock(ctx, eventID, func(tx interfaces.EventMediaTransaction) error {
		media, err := tx.List()
		if err != nil {
			return err
		}
		for i := range media {
			if media[i].FileURL == key {
				found = &media[i]
				return nil
			}
		}
		return interfaces.ErrMediaNotFound
	})
	if errors.Is(err, interfaces.ErrMediaEventNotFound) {
		return nil, interfaces.ErrMediaNotFound
	}
	return found, err
}

type eventMediaTransaction struct {
	db    *gorm.DB
	event *entities.Event
}

func (t *eventMediaTransaction) Event() *entities.Event { return t.event }

func (t *eventMediaTransaction) List() ([]entities.EventMedia, error) {
	media := make([]entities.EventMedia, 0)
	err := t.db.Table("event_media").Where("event_id = ?", t.event.ID).Order("order_index ASC, id ASC").Find(&media).Error
	return media, err
}

func (t *eventMediaTransaction) Insert(media *entities.EventMedia) error {
	return t.db.Table("event_media").Create(media).Error
}

func (t *eventMediaTransaction) Find(id uint) (*entities.EventMedia, error) {
	var media entities.EventMedia
	err := t.db.Table("event_media").Where("event_id = ? AND id = ?", t.event.ID, id).Take(&media).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, interfaces.ErrMediaNotFound
	}
	return &media, err
}

func (t *eventMediaTransaction) Delete(id uint) error {
	result := t.db.Table("event_media").Where("event_id = ? AND id = ?", t.event.ID, id).Delete(&entities.EventMedia{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return interfaces.ErrMediaNotFound
	}
	return nil
}
