package interfaces

import (
	"context"
	"errors"
	"io"
	"time"

	"auth-system/internal/domain/entities"
)

var (
	ErrMediaEventNotFound = errors.New("event not found")
	ErrMediaNotFound      = errors.New("media not found")
	ErrMediaCommitUnknown = errors.New("media transaction commit outcome unknown")
)

// WithEventLock serializes media mutations across all application instances.
// Every method on EventMediaTransaction uses the same database transaction.
type EventMediaRepository interface {
	WithEventLock(context.Context, uint, func(EventMediaTransaction) error) error
	FindByKey(context.Context, uint, string) (*entities.EventMedia, error)
}

type EventMediaTransaction interface {
	Event() *entities.Event
	List() ([]entities.EventMedia, error)
	Insert(*entities.EventMedia) error
	Find(uint) (*entities.EventMedia, error)
	Delete(uint) error
}

type ObjectStorage interface {
	Put(context.Context, string, io.Reader, int64, string) error
	Remove(context.Context, string) error
	PresignGet(context.Context, string, time.Duration) (string, error)
}
