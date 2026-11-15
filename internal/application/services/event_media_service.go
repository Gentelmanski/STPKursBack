package services

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"time"

	"auth-system/internal/application/dto"
	"auth-system/internal/application/interfaces"
	"auth-system/internal/domain/entities"

	_ "golang.org/x/image/webp"
)

const (
	MaxMediaFileSize = 5 * 1024 * 1024
	MaxEventPhotos   = 5
	MediaURLTTL      = 15 * time.Minute
	// Bound decoded memory as well as compressed upload size.
	MaxMediaPixels = 25_000_000
)

var (
	ErrMediaForbidden    = errors.New("only the event creator can change photos")
	ErrMediaLimit        = errors.New("an event can have at most 5 photos")
	ErrMediaTooLarge     = errors.New("file exceeds 5 MiB")
	ErrMediaInvalidImage = errors.New("file must be a valid JPEG, PNG or WebP image of at most 25 megapixels")
	ErrMediaStorage      = errors.New("photo storage is temporarily unavailable")
)

type EventMediaService struct {
	repo    interfaces.EventMediaRepository
	storage interfaces.ObjectStorage
}

func NewEventMediaService(repo interfaces.EventMediaRepository, storage interfaces.ObjectStorage) *EventMediaService {
	return &EventMediaService{repo: repo, storage: storage}
}

func validateMediaImage(data []byte) (string, string, error) {
	if len(data) > MaxMediaFileSize {
		return "", "", ErrMediaTooLarge
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || cfg.Width <= 0 || cfg.Height <= 0 || int64(cfg.Width)*int64(cfg.Height) > MaxMediaPixels {
		return "", "", ErrMediaInvalidImage
	}
	var ext, mime string
	switch format {
	case "jpeg":
		ext, mime = "jpg", "image/jpeg"
	case "png":
		ext, mime = "png", "image/png"
	case "webp":
		ext, mime = "webp", "image/webp"
	default:
		return "", "", ErrMediaInvalidImage
	}
	if _, decodedFormat, err := image.Decode(bytes.NewReader(data)); err != nil || decodedFormat != format {
		return "", "", ErrMediaInvalidImage
	}
	return ext, mime, nil
}

func (s *EventMediaService) Upload(ctx context.Context, eventID, userID uint, data []byte) (*dto.EventMediaResponse, error) {
	ext, mime, err := validateMediaImage(data)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	var media entities.EventMedia
	var response *dto.EventMediaResponse
	var attempted, inserted bool
	err = s.repo.WithEventLock(ctx, eventID, func(tx interfaces.EventMediaTransaction) error {
		event := tx.Event()
		if !event.IsActive {
			return interfaces.ErrMediaEventNotFound
		}
		if event.CreatorID != userID {
			return ErrMediaForbidden
		}
		existing, err := tx.List()
		if err != nil {
			return err
		}
		if len(existing) >= MaxEventPhotos {
			return ErrMediaLimit
		}
		unique := make([]byte, 16)
		if _, err := rand.Read(unique); err != nil {
			return errors.New("cannot generate photo key")
		}
		now := time.Now().UTC()
		media = entities.EventMedia{
			EventID:    eventID,
			FileURL:    fmt.Sprintf("events/%d/%s.%s", eventID, hex.EncodeToString(unique), ext),
			FileType:   mime,
			UploadedAt: now, CreatedAt: now,
		}
		for _, item := range existing {
			if item.OrderIndex >= media.OrderIndex {
				media.OrderIndex = item.OrderIndex + 1
			}
		}
		// Sign before uploading: a signing error cannot turn a committed upload into an API failure.
		response, err = s.view(ctx, &media)
		if err != nil {
			return err
		}
		attempted = true
		if err := s.storage.Put(ctx, media.FileURL, bytes.NewReader(data), int64(len(data)), mime); err != nil {
			return ErrMediaStorage
		}
		if err := tx.Insert(&media); err != nil {
			return err
		}
		inserted = true
		response.ID = media.ID
		return nil
	})
	if err == nil {
		return response, nil
	}
	if inserted && errors.Is(err, interfaces.ErrMediaCommitUnknown) {
		// A failed COMMIT acknowledgement does not prove rollback. Never remove an
		// object until a fresh database read confirms that its metadata is absent.
		checkCtx, checkCancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		persisted, checkErr := s.repo.FindByKey(checkCtx, eventID, media.FileURL)
		checkCancel()
		if checkErr == nil {
			response.ID = persisted.ID
			return response, nil
		}
		if !errors.Is(checkErr, interfaces.ErrMediaNotFound) {
			log.Printf("media upload outcome unknown; reconcile event_id=%d object_key=%s", eventID, media.FileURL)
			return nil, err
		}
	}
	if attempted {
		// Cleanup must survive a disconnected HTTP client or an aborted DB transaction.
		if cleanupErr := s.removeWithRetry(context.WithoutCancel(ctx), media.FileURL); cleanupErr != nil {
			log.Printf("media upload cleanup failed; retry object removal event_id=%d object_key=%s", eventID, media.FileURL)
		}
	}
	return nil, err
}

func (s *EventMediaService) List(ctx context.Context, eventID uint) ([]dto.EventMediaResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	result := make([]dto.EventMediaResponse, 0)
	err := s.repo.WithEventLock(ctx, eventID, func(tx interfaces.EventMediaTransaction) error {
		// Mirrors EventService.GetEventByID: every authenticated user can view an active event.
		if !tx.Event().IsActive {
			return interfaces.ErrMediaEventNotFound
		}
		media, err := tx.List()
		if err != nil {
			return err
		}
		for i := range media {
			item, err := s.view(ctx, &media[i])
			if err != nil {
				return err
			}
			result = append(result, *item)
		}
		return nil
	})
	return result, err
}

func (s *EventMediaService) Delete(ctx context.Context, eventID, mediaID, userID uint) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	return s.repo.WithEventLock(ctx, eventID, func(tx interfaces.EventMediaTransaction) error {
		if tx.Event().CreatorID != userID {
			return ErrMediaForbidden
		}
		media, err := tx.Find(mediaID)
		if err != nil {
			return err
		}
		// Keep metadata on S3 failure. If DB deletion/COMMIT fails afterwards, a
		// repeated DELETE safely removes the already absent object and retries DB deletion.
		if err := s.removeWithRetry(ctx, media.FileURL); err != nil {
			return ErrMediaStorage
		}
		return tx.Delete(mediaID)
	})
}

func (s *EventMediaService) view(ctx context.Context, media *entities.EventMedia) (*dto.EventMediaResponse, error) {
	expiresAt := time.Now().UTC().Truncate(time.Second).Add(MediaURLTTL)
	signed, err := s.storage.PresignGet(ctx, media.FileURL, MediaURLTTL)
	if err != nil {
		return nil, ErrMediaStorage
	}
	return &dto.EventMediaResponse{
		ID: media.ID, EventID: media.EventID, FileURL: signed,
		FileType: media.FileType, OrderIndex: media.OrderIndex,
		UploadedAt: media.UploadedAt, CreatedAt: media.CreatedAt,
		URLExpiresAt: expiresAt,
	}, nil
}

func (s *EventMediaService) removeWithRetry(ctx context.Context, key string) error {
	ctx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	var err error
	for attempt := 0; attempt < 3; attempt++ {
		if err = s.storage.Remove(ctx, key); err == nil {
			return nil
		}
		if attempt == 2 {
			break
		}
		timer := time.NewTimer(time.Duration(attempt+1) * 100 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
	return err
}
