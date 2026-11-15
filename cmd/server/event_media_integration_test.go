package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"net/url"
	"os"
	"regexp"
	"sync"
	"testing"
	"time"

	"auth-system/internal/application/dto"
	"auth-system/internal/application/interfaces"
	"auth-system/internal/application/services"
	"auth-system/internal/config"
	"auth-system/internal/domain/entities"
	"auth-system/internal/infrastructure/repositories"
	"auth-system/internal/infrastructure/storage"
	"auth-system/internal/pkg/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Explicit opt-in: this test uses Config, creates isolated fixtures in the real
// database/bucket, exercises the real JWT routes, and cleans up only its fixtures.
func TestEventMediaIntegration(t *testing.T) {
	if os.Getenv("RUN_MEDIA_INTEGRATION") != "1" {
		t.Skip("set RUN_MEDIA_INTEGRATION=1 to exercise local PostgreSQL and S3")
	}
	cfg := config.Load()
	db, err := gorm.Open(postgres.Open(cfg.DatabaseURL), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal("cannot connect to integration database")
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal("cannot access integration database")
	}
	t.Cleanup(func() { sqlDB.Close() })
	store, err := storage.NewS3(cfg)
	if err != nil {
		t.Fatal("cannot initialize integration S3 client")
	}
	runID := fmt.Sprintf("media-it-%d", time.Now().UnixNano())
	userIDs := make([]uint, 0, 2)
	eventIDs := make([]uint, 0, 2)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		if len(eventIDs) > 0 {
			var media []entities.EventMedia
			if err := db.WithContext(ctx).Table("event_media").Where("event_id IN ?", eventIDs).Find(&media).Error; err != nil {
				t.Error("fixture media lookup failed")
			}
			for _, m := range media {
				if err := store.Remove(ctx, m.FileURL); err != nil {
					t.Errorf("fixture S3 cleanup failed for media ID %d", m.ID)
				}
			}
			if err := db.WithContext(ctx).Exec("DELETE FROM events WHERE id IN ?", eventIDs).Error; err != nil {
				t.Error("fixture event cleanup failed")
			}
		}
		if len(userIDs) > 0 {
			if err := db.WithContext(ctx).Exec("DELETE FROM users WHERE id IN ?", userIDs).Error; err != nil {
				t.Error("fixture user cleanup failed")
			}
		}
	})
	for i := 0; i < 2; i++ {
		user := entities.User{Username: fmt.Sprintf("%s-%d", runID, i), Email: fmt.Sprintf("%s-%d@example.invalid", runID, i), PasswordHash: "integration-fixture-no-login", Role: "user", LastOnline: time.Now()}
		if err := db.Create(&user).Error; err != nil {
			t.Fatal("could not create isolated fixture user")
		}
		userIDs = append(userIDs, user.ID)
	}
	eventRepo := repositories.NewEventRepository(db)
	for i := 0; i < 2; i++ {
		event := &entities.Event{Title: fmt.Sprintf("%s-%d", runID, i), Description: "temporary media integration fixture", EventDate: time.Now().Add(time.Hour), Latitude: 55.75, Longitude: 37.62, Type: "other", IsActive: true, CreatorID: userIDs[0]}
		if err := eventRepo.Create(context.Background(), event); err != nil {
			t.Fatal("existing event creation failed")
		}
		eventIDs = append(eventIDs, event.ID)
	}
	jwt := utils.NewJWTUtil(cfg.JWTSecret)
	ownerToken, err := jwt.GenerateToken(userIDs[0], "owner@example.invalid", "user")
	if err != nil {
		t.Fatal("token creation failed")
	}
	otherToken, err := jwt.GenerateToken(userIDs[1], "other@example.invalid", "user")
	if err != nil {
		t.Fatal("token creation failed")
	}
	svc := setupServices(repositories.NewRepositories(db), jwt, utils.NewPasswordUtil())
	mediaRepo := repositories.NewEventMediaRepository(db)
	svc.EventMedia = services.NewEventMediaService(mediaRepo, store)
	gin.SetMode(gin.TestMode)
	server := httptest.NewServer(setupRouter(gin.New(), setupControllers(svc), jwt))
	defer server.Close()
	client := &http.Client{Timeout: 45 * time.Second}
	endpoint := func(id uint) string { return fmt.Sprintf("/api/events/%d/media", id) }
	request := func(method, path, token string, data []byte, contentType string) (int, []byte) {
		req, err := http.NewRequest(method, server.URL+path, bytes.NewReader(data))
		if err != nil {
			t.Fatal("could not create API request")
		}
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		if contentType != "" {
			req.Header.Set("Content-Type", contentType)
		}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatal("API request failed")
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal("API response read failed")
		}
		return resp.StatusCode, body
	}
	var pngBuffer, jpgBuffer bytes.Buffer
	if err := png.Encode(&pngBuffer, image.NewRGBA(image.Rect(0, 0, 3, 3))); err != nil {
		t.Fatal(err)
	}
	if err := jpeg.Encode(&jpgBuffer, image.NewRGBA(image.Rect(0, 0, 3, 3)), nil); err != nil {
		t.Fatal(err)
	}
	pngData := pngBuffer.Bytes()
	upload := func(id uint, token string, data []byte) (int, []byte) {
		body, contentType := integrationMultipart(data)
		return request("POST", endpoint(id), token, body, contentType)
	}
	t.Run("JWT and ownership", func(t *testing.T) {
		if status, _ := upload(eventIDs[0], "", pngData); status != 401 {
			t.Fatalf("missing JWT: %d", status)
		}
		if status, _ := upload(eventIDs[0], otherToken, pngData); status != 403 {
			t.Fatalf("non-owner upload: %d", status)
		}
		status, body := request("GET", endpoint(eventIDs[0]), otherToken, nil, "")
		if status != 200 || string(body) != `{"media":[]}` {
			t.Fatalf("authenticated empty list status=%d", status)
		}
		if status, _ := upload(eventIDs[0], ownerToken, []byte("not an image")); status != 415 {
			t.Fatalf("fake image: %d", status)
		}
	})
	var first dto.EventMediaResponse
	t.Run("upload, persistent key, signed GET and private bucket", func(t *testing.T) {
		status, body := upload(eventIDs[0], ownerToken, pngData)
		if status != 201 {
			t.Fatalf("real S3 upload status=%d; check local S3 configuration", status)
		}
		if err := json.Unmarshal(body, &first); err != nil {
			t.Fatal("invalid upload response")
		}
		if first.ID == 0 || first.EventID != eventIDs[0] || first.FileType != "image/png" || first.OrderIndex != 0 || first.UploadedAt.IsZero() {
			t.Fatal("invalid upload metadata")
		}
		var stored entities.EventMedia
		if err := db.Table("event_media").Where("id = ?", first.ID).Take(&stored).Error; err != nil {
			t.Fatal("metadata row missing")
		}

		reconciled, lookupErr := mediaRepo.FindByKey(context.Background(), eventIDs[0], stored.FileURL)
		if lookupErr != nil || reconciled.ID != first.ID {
			t.Fatal("committed media reconciliation failed")
		}
		if _, lookupErr := mediaRepo.FindByKey(context.Background(), eventIDs[0], "missing-integration-object"); !errors.Is(lookupErr, interfaces.ErrMediaNotFound) {
			t.Fatal("missing media reconciliation failed")
		}
		pattern := fmt.Sprintf(`^events/%d/[0-9a-f]{32}\.png$`, eventIDs[0])
		if !regexp.MustCompile(pattern).MatchString(stored.FileURL) {
			t.Fatal("database must store only an object key")
		}
		signed, err := url.Parse(first.FileURL)
		if err != nil || signed.Query().Get("X-Amz-Expires") != "900" {
			t.Fatal("expected 15-minute signed viewing URL")
		}
		resp, err := client.Get(first.FileURL)
		if err != nil {
			t.Fatal("signed S3 GET failed")
		}
		got, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil || resp.StatusCode != 200 || !bytes.Equal(got, pngData) || resp.Header.Get("Content-Type") != "image/png" {
			t.Fatalf("signed GET status=%d", resp.StatusCode)
		}
		signed.RawQuery = ""
		anon, err := client.Get(signed.String())
		if err != nil {
			t.Fatal("unsigned S3 GET failed")
		}
		anon.Body.Close()
		if anon.StatusCode != 403 {
			t.Fatalf("private bucket unsigned GET status=%d", anon.StatusCode)
		}
		if status, _ := request("DELETE", fmt.Sprintf("%s/%d", endpoint(eventIDs[0]), first.ID), otherToken, nil, ""); status != 403 {
			t.Fatalf("non-owner delete status=%d", status)
		}
		if status, _ := request("DELETE", fmt.Sprintf("%s/%d", endpoint(eventIDs[1]), first.ID), ownerToken, nil, ""); status != 404 {
			t.Fatalf("cross-event delete status=%d", status)
		}
		status, body = request("GET", endpoint(eventIDs[0]), otherToken, nil, "")
		var list struct {
			Media []dto.EventMediaResponse `json:"media"`
		}
		if status != 200 || json.Unmarshal(body, &list) != nil || len(list.Media) != 1 || list.Media[0].ID != first.ID {
			t.Fatal("non-owner should be able to view active event photos")
		}
	})
	if first.ID == 0 {
		t.Fatal("cannot continue without real S3 upload")
	}
	t.Run("JPEG and WebP content detection", func(t *testing.T) {
		webp, err := base64.StdEncoding.DecodeString("UklGRh4AAABXRUJQVlA4TBEAAAAvAAAAAAfQ//73v/+BiOh/AAA=")
		if err != nil {
			t.Fatal(err)
		}
		for _, tc := range []struct {
			data []byte
			mime string
		}{{jpgBuffer.Bytes(), "image/jpeg"}, {webp, "image/webp"}} {
			status, body := upload(eventIDs[1], ownerToken, tc.data)
			var photo dto.EventMediaResponse
			if status != 201 || json.Unmarshal(body, &photo) != nil || photo.FileType != tc.mime {
				t.Fatalf("%s upload status=%d", tc.mime, status)
			}
		}
	})
	t.Run("metadata failure removes real S3 object", func(t *testing.T) {
		watched := &observedMediaStorage{ObjectStorage: store}
		failing := &insertFailMediaRepository{EventMediaRepository: mediaRepo}
		_, err := services.NewEventMediaService(failing, watched).Upload(context.Background(), eventIDs[1], userIDs[0], pngData)
		if err == nil || watched.key == "" {
			t.Fatal("expected database failure after real upload")
		}
		signed, err := store.PresignGet(context.Background(), watched.key, time.Minute)
		if err != nil {
			t.Fatal("cleanup verification signing failed")
		}
		resp, err := client.Get(signed)
		if err != nil {
			t.Fatal("cleanup verification GET failed")
		}
		resp.Body.Close()
		if resp.StatusCode != 404 {
			t.Fatalf("compensated object GET status=%d", resp.StatusCode)
		}
		var count int64
		if err := db.Table("event_media").Where("file_url = ?", watched.key).Count(&count).Error; err != nil || count != 0 {
			t.Fatal("failed metadata insertion left a row")
		}
	})
	t.Run("concurrent upload limit with PostgreSQL row lock", func(t *testing.T) {
		body, contentType := integrationMultipart(pngData)
		statuses := make(chan int, 8)
		var wg sync.WaitGroup
		for i := 0; i < 8; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				req, err := http.NewRequest("POST", server.URL+endpoint(eventIDs[0]), bytes.NewReader(body))
				if err != nil {
					statuses <- 0
					return
				}
				req.Header.Set("Authorization", "Bearer "+ownerToken)
				req.Header.Set("Content-Type", contentType)
				resp, err := client.Do(req)
				if err != nil {
					statuses <- 0
					return
				}
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
				statuses <- resp.StatusCode
			}()
		}
		wg.Wait()
		close(statuses)
		created, conflicts := 0, 0
		for status := range statuses {
			switch status {
			case 201:
				created++
			case 409:
				conflicts++
			default:
				t.Errorf("parallel upload status=%d", status)
			}
		}
		var count int64
		db.Table("event_media").Where("event_id = ?", eventIDs[0]).Count(&count)
		if created != 4 || conflicts != 4 || count != 5 {
			t.Fatalf("created=%d conflicts=%d rows=%d", created, conflicts, count)
		}
	})
	t.Run("file and request size limits", func(t *testing.T) {
		exact := make([]byte, services.MaxMediaFileSize)
		copy(exact, pngData)
		if status, _ := upload(eventIDs[1], ownerToken, exact); status != 201 {
			t.Fatalf("exact 5 MiB boundary status=%d", status)
		}

		for _, size := range []int{services.MaxMediaFileSize + 1, 6 * 1024 * 1024} {
			if status, _ := upload(eventIDs[1], ownerToken, make([]byte, size)); status != 413 {
				t.Fatalf("oversize file/request status=%d", status)
			}
		}
	})
	t.Run("existing verification and participation", func(t *testing.T) {
		if err := eventRepo.VerifyEvent(context.Background(), eventIDs[0]); err != nil {
			t.Fatal("existing verification failed")
		}
		path := fmt.Sprintf("/api/events/%d/participate", eventIDs[0])
		if status, _ := request("POST", path, otherToken, nil, ""); status != 200 {
			t.Fatalf("existing participation status=%d", status)
		}
		event, err := eventRepo.FindByID(context.Background(), eventIDs[0])
		if err != nil || !event.IsVerified || event.ParticipantsCount != 1 || event.Latitude != 55.75 || event.Longitude != 37.62 {
			t.Fatal("existing event read/verification/coordinates/participation changed")
		}
		if status, _ := request("DELETE", path, otherToken, nil, ""); status != 200 {
			t.Fatalf("existing participation cancellation status=%d", status)
		}
	})
	t.Run("repeat removal of absent S3 object", func(t *testing.T) {
		var stored entities.EventMedia
		if err := db.Table("event_media").Where("id = ?", first.ID).Take(&stored).Error; err != nil {
			t.Fatal("metadata missing")
		}
		if err := store.Remove(context.Background(), stored.FileURL); err != nil {
			t.Fatal("first real S3 deletion failed")
		}
		path := fmt.Sprintf("%s/%d", endpoint(eventIDs[0]), first.ID)
		if status, _ := request("DELETE", path, ownerToken, nil, ""); status != 204 {
			t.Fatalf("retry delete status=%d", status)
		}
		if status, _ := request("DELETE", path, ownerToken, nil, ""); status != 404 {
			t.Fatalf("already removed metadata status=%d", status)
		}
	})
	t.Run("inactive visibility and owner cleanup", func(t *testing.T) {
		if status, _ := request("DELETE", fmt.Sprintf("/api/events/%d", eventIDs[0]), ownerToken, nil, ""); status != 200 {
			t.Fatalf("existing event soft deletion status=%d", status)
		}
		if status, _ := request("GET", endpoint(eventIDs[0]), ownerToken, nil, ""); status != 404 {
			t.Fatalf("inactive media list status=%d", status)
		}
		if status, _ := upload(eventIDs[0], ownerToken, pngData); status != 404 {
			t.Fatalf("inactive upload status=%d", status)
		}
		var remaining entities.EventMedia
		if err := db.Table("event_media").Where("event_id = ?", eventIDs[0]).Take(&remaining).Error; err != nil {
			t.Fatal("remaining fixture photo missing")
		}
		if status, _ := request("DELETE", fmt.Sprintf("%s/%d", endpoint(eventIDs[0]), remaining.ID), ownerToken, nil, ""); status != 204 {
			t.Fatalf("inactive owner cleanup status=%d", status)
		}
		signed, signErr := store.PresignGet(context.Background(), remaining.FileURL, time.Minute)
		if signErr != nil {
			t.Fatal("cannot verify normal object deletion")
		}
		removed, getErr := client.Get(signed)
		if getErr != nil {
			t.Fatal("deleted object GET failed")
		}
		removed.Body.Close()
		if removed.StatusCode != 404 {
			t.Fatalf("deleted object GET status=%d", removed.StatusCode)
		}

	})
	t.Log("real PostgreSQL + Silo: JWT routes, PNG/JPEG/WebP, signed GET, private bucket, object keys, ownership, concurrent limit, compensation, deletion, existing event operations verified")
}

func integrationMultipart(data []byte) ([]byte, string) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", `form-data; name="file"; filename="untrusted.exe"`)
	header.Set("Content-Type", "text/plain")
	part, err := writer.CreatePart(header)
	if err != nil {
		panic(err)
	}
	if _, err := part.Write(data); err != nil {
		panic(err)
	}
	if err := writer.Close(); err != nil {
		panic(err)
	}
	return body.Bytes(), writer.FormDataContentType()
}

type insertFailMediaRepository struct {
	interfaces.EventMediaRepository
}

func (r *insertFailMediaRepository) WithEventLock(ctx context.Context, id uint, fn func(interfaces.EventMediaTransaction) error) error {
	return r.EventMediaRepository.WithEventLock(ctx, id, func(tx interfaces.EventMediaTransaction) error { return fn(insertFailMediaTransaction{tx}) })
}

type insertFailMediaTransaction struct {
	interfaces.EventMediaTransaction
}

func (t insertFailMediaTransaction) Insert(*entities.EventMedia) error {
	return errors.New("injected metadata insertion failure")
}

type observedMediaStorage struct {
	interfaces.ObjectStorage
	key string
}

func (s *observedMediaStorage) Put(ctx context.Context, key string, r io.Reader, size int64, mime string) error {
	s.key = key
	return s.ObjectStorage.Put(ctx, key, r, size, mime)
}
