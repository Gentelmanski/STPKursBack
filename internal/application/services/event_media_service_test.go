package services

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"auth-system/internal/application/interfaces"
	"auth-system/internal/domain/entities"
)

func mediaPNG(t *testing.T) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := png.Encode(&b, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func TestValidateMediaImage(t *testing.T) {
	var jpg, gifData bytes.Buffer
	if err := jpeg.Encode(&jpg, image.NewRGBA(image.Rect(0, 0, 2, 2)), nil); err != nil {
		t.Fatal(err)
	}
	if err := gif.Encode(&gifData, image.NewRGBA(image.Rect(0, 0, 2, 2)), nil); err != nil {
		t.Fatal(err)
	}
	webp, err := base64.StdEncoding.DecodeString("UklGRh4AAABXRUJQVlA4TBEAAAAvAAAAAAfQ//73v/+BiOh/AAA=")
	if err != nil {
		t.Fatal(err)
	}
	pngData := mediaPNG(t)
	huge := append([]byte(nil), pngData...)
	binary.BigEndian.PutUint32(huge[16:20], 40000)
	binary.BigEndian.PutUint32(huge[20:24], 40000)
	binary.BigEndian.PutUint32(huge[29:33], crc32.ChecksumIEEE(huge[12:29]))
	exact := make([]byte, MaxMediaFileSize)
	copy(exact, pngData)
	for _, tc := range []struct {
		name      string
		data      []byte
		ext, mime string
		want      error
	}{
		{"png", pngData, "png", "image/png", nil},
		{"jpeg", jpg.Bytes(), "jpg", "image/jpeg", nil},
		{"webp", webp, "webp", "image/webp", nil},
		{"exact size boundary", exact, "png", "image/png", nil},
		{"empty", nil, "", "", ErrMediaInvalidImage},
		{"fake jpeg", []byte("<svg><script>alert(1)</script></svg>"), "", "", ErrMediaInvalidImage},
		{"gif", gifData.Bytes(), "", "", ErrMediaInvalidImage},
		{"truncated png", pngData[:len(pngData)/2], "", "", ErrMediaInvalidImage},
		{"truncated jpeg", jpg.Bytes()[:jpg.Len()/2], "", "", ErrMediaInvalidImage},
		{"too many pixels", huge, "", "", ErrMediaInvalidImage},
		{"over 5 MiB", make([]byte, MaxMediaFileSize+1), "", "", ErrMediaTooLarge},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ext, mime, err := validateMediaImage(tc.data)
			if !errors.Is(err, tc.want) || ext != tc.ext || mime != tc.mime {
				t.Fatalf("got (%s, %s, %v), want (%s, %s, %v)", ext, mime, err, tc.ext, tc.mime, tc.want)
			}
		})
	}
}

type memoryMediaRepository struct {
	mu                                         sync.Mutex
	events                                     map[uint]entities.Event
	media                                      map[uint]entities.EventMedia
	next                                       uint
	insertErr, deleteErr, commitErr, lookupErr error
	commitPersists                             bool
	insertHook                                 func()
}

func newMemoryMediaRepository() *memoryMediaRepository {
	return &memoryMediaRepository{
		events: map[uint]entities.Event{1: {ID: 1, CreatorID: 10, IsActive: true}, 2: {ID: 2, CreatorID: 10, IsActive: true}},
		media:  make(map[uint]entities.EventMedia), next: 1,
	}
}

func (r *memoryMediaRepository) WithEventLock(ctx context.Context, id uint, fn func(interfaces.EventMediaTransaction) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	event, ok := r.events[id]
	if !ok {
		return interfaces.ErrMediaEventNotFound
	}
	before := make(map[uint]entities.EventMedia, len(r.media))
	for id, m := range r.media {
		before[id] = m
	}
	err := fn(&memoryMediaTransaction{repo: r, event: &event})
	if err != nil {
		r.media = before
		return err
	}
	if r.commitErr != nil {
		if !r.commitPersists {
			r.media = before
		}
		return r.commitErr
	}
	return nil
}
func (r *memoryMediaRepository) FindByKey(ctx context.Context, eventID uint, key string) (*entities.EventMedia, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.lookupErr != nil {
		return nil, r.lookupErr
	}
	for _, m := range r.media {
		if m.EventID == eventID && m.FileURL == key {
			return &m, nil
		}
	}
	return nil, interfaces.ErrMediaNotFound
}

type memoryMediaTransaction struct {
	repo  *memoryMediaRepository
	event *entities.Event
}

func (t *memoryMediaTransaction) Event() *entities.Event { return t.event }
func (t *memoryMediaTransaction) List() ([]entities.EventMedia, error) {
	media := make([]entities.EventMedia, 0)
	for _, m := range t.repo.media {
		if m.EventID == t.event.ID {
			media = append(media, m)
		}
	}
	sort.Slice(media, func(i, j int) bool { return media[i].OrderIndex < media[j].OrderIndex })
	return media, nil
}
func (t *memoryMediaTransaction) Insert(m *entities.EventMedia) error {
	if t.repo.insertHook != nil {
		t.repo.insertHook()
	}
	if t.repo.insertErr != nil {
		return t.repo.insertErr
	}
	m.ID = t.repo.next
	t.repo.next++
	t.repo.media[m.ID] = *m
	return nil
}
func (t *memoryMediaTransaction) Find(id uint) (*entities.EventMedia, error) {
	m, ok := t.repo.media[id]
	if !ok || m.EventID != t.event.ID {
		return nil, interfaces.ErrMediaNotFound
	}
	return &m, nil
}
func (t *memoryMediaTransaction) Delete(id uint) error {
	if t.repo.deleteErr != nil {
		return t.repo.deleteErr
	}
	delete(t.repo.media, id)
	return nil
}

type memoryObjectStorage struct {
	objects         map[string][]byte
	mime            map[string]string
	putErr, signErr error
	removeFailures  int
	puts, removes   int
	cancelledRemove bool
}

func newMemoryStorage() *memoryObjectStorage {
	return &memoryObjectStorage{objects: make(map[string][]byte), mime: make(map[string]string)}
}
func (s *memoryObjectStorage) Put(ctx context.Context, key string, r io.Reader, size int64, mime string) error {
	s.puts++
	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	if int64(len(data)) != size {
		return errors.New("incorrect object size")
	}
	s.objects[key] = data
	s.mime[key] = mime
	return s.putErr
}
func (s *memoryObjectStorage) Remove(ctx context.Context, key string) error {
	s.removes++
	if ctx.Err() != nil {
		s.cancelledRemove = true
		return ctx.Err()
	}
	if s.removeFailures != 0 {
		if s.removeFailures > 0 {
			s.removeFailures--
		}
		return errors.New("storage down")
	}
	delete(s.objects, key)
	return nil
}
func (s *memoryObjectStorage) PresignGet(ctx context.Context, key string, ttl time.Duration) (string, error) {
	if s.signErr != nil {
		return "", s.signErr
	}
	return "https://storage.invalid/" + key + "?signed=temporary", nil
}

func TestMediaUploadOwnershipAndVisibility(t *testing.T) {
	for _, tc := range []struct {
		name            string
		eventID, userID uint
		inactive        bool
		want            error
	}{
		{"non-owner", 1, 20, false, ErrMediaForbidden},
		{"missing event", 99, 10, false, interfaces.ErrMediaEventNotFound},
		{"inactive", 1, 10, true, interfaces.ErrMediaEventNotFound},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo, store := newMemoryMediaRepository(), newMemoryStorage()
			if tc.inactive {
				e := repo.events[1]
				e.IsActive = false
				repo.events[1] = e
			}
			_, err := NewEventMediaService(repo, store).Upload(context.Background(), tc.eventID, tc.userID, mediaPNG(t))
			if !errors.Is(err, tc.want) || store.puts != 0 || len(repo.media) != 0 {
				t.Fatalf("err=%v puts=%d rows=%d", err, store.puts, len(repo.media))
			}
		})
	}
}

func TestMediaUploadConcurrentLimitAndPersistentKeys(t *testing.T) {
	repo, store := newMemoryMediaRepository(), newMemoryStorage()
	service := NewEventMediaService(repo, store)
	data := mediaPNG(t)
	var wg sync.WaitGroup
	results := make(chan error, 12)
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); _, err := service.Upload(context.Background(), 1, 10, data); results <- err }()
	}
	wg.Wait()
	close(results)
	success, limited := 0, 0
	for err := range results {
		if err == nil {
			success++
		} else if errors.Is(err, ErrMediaLimit) {
			limited++
		} else {
			t.Fatal(err)
		}
	}
	if success != 5 || limited != 7 || len(store.objects) != 5 || len(repo.media) != 5 {
		t.Fatalf("success=%d limited=%d objects=%d rows=%d", success, limited, len(store.objects), len(repo.media))
	}
	indexes := make(map[int]bool)
	for _, m := range repo.media {
		if !strings.HasPrefix(m.FileURL, "events/1/") || !strings.HasSuffix(m.FileURL, ".png") || strings.Contains(m.FileURL, "?") {
			t.Fatal("invalid persistent key")
		}
		if !bytes.Equal(store.objects[m.FileURL], data) || m.FileType != "image/png" || m.UploadedAt.IsZero() {
			t.Fatal("incorrect stored metadata or bytes")
		}
		indexes[m.OrderIndex] = true
	}
	if len(indexes) != 5 {
		t.Fatal("order indexes must be distinct")
	}
	list, err := service.List(context.Background(), 1)
	if err != nil || len(list) != 5 {
		t.Fatalf("list len=%d err=%v", len(list), err)
	}
	for _, m := range list {
		if !strings.Contains(m.FileURL, "?signed=") || time.Until(m.URLExpiresAt) > MediaURLTTL {
			t.Fatal("missing temporary URL or invalid expiry")
		}
	}
}

func TestMediaUploadFailureCleanup(t *testing.T) {
	for _, where := range []string{"insert", "put", "sign"} {
		t.Run(where, func(t *testing.T) {
			repo, store := newMemoryMediaRepository(), newMemoryStorage()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			switch where {
			case "insert":
				repo.insertErr = errors.New("database unavailable")
				repo.insertHook = cancel
				store.removeFailures = 1
			case "put":
				store.putErr = errors.New("upload acknowledgement lost")
			case "sign":
				store.signErr = errors.New("sign failed")
			}
			_, err := NewEventMediaService(repo, store).Upload(ctx, 1, 10, mediaPNG(t))
			if err == nil || len(repo.media) != 0 || len(store.objects) != 0 || store.cancelledRemove {
				t.Fatalf("err=%v rows=%d objects=%d cancelledCleanup=%v", err, len(repo.media), len(store.objects), store.cancelledRemove)
			}
			if where == "insert" && store.removes != 2 {
				t.Fatalf("cleanup attempts=%d", store.removes)
			}
			if where == "sign" && store.puts != 0 {
				t.Fatal("upload before signing succeeded")
			}
		})
	}
}

func TestMediaUploadCommitUncertainty(t *testing.T) {
	for _, tc := range []struct {
		name                    string
		persist, unknown        bool
		wantSuccess, wantObject bool
	}{
		{"committed acknowledgement lost", true, false, true, true},
		{"rolled back", false, false, false, false},
		{"database unreachable", false, true, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo, store := newMemoryMediaRepository(), newMemoryStorage()
			repo.commitErr = interfaces.ErrMediaCommitUnknown
			repo.commitPersists = tc.persist
			if tc.unknown {
				repo.lookupErr = errors.New("database down")
			}
			_, err := NewEventMediaService(repo, store).Upload(context.Background(), 1, 10, mediaPNG(t))
			if (err == nil) != tc.wantSuccess || (len(store.objects) > 0) != tc.wantObject {
				t.Fatalf("err=%v objects=%d", err, len(store.objects))
			}
		})
	}
}

func TestMediaDeleteAuthorizationAndRetries(t *testing.T) {
	repo, store := newMemoryMediaRepository(), newMemoryStorage()
	service := NewEventMediaService(repo, store)
	photo, err := service.Upload(context.Background(), 1, 10, mediaPNG(t))
	if err != nil {
		t.Fatal(err)
	}
	if err := service.Delete(context.Background(), 1, photo.ID, 20); !errors.Is(err, ErrMediaForbidden) {
		t.Fatal(err)
	}
	if err := service.Delete(context.Background(), 2, photo.ID, 10); !errors.Is(err, interfaces.ErrMediaNotFound) {
		t.Fatal(err)
	}
	if store.removes != 0 {
		t.Fatal("unauthorized delete reached storage")
	}
	store.removeFailures = -1
	if err := service.Delete(context.Background(), 1, photo.ID, 10); !errors.Is(err, ErrMediaStorage) {
		t.Fatal(err)
	}
	if len(repo.media) != 1 || len(store.objects) != 1 || store.removes != 3 {
		t.Fatal("failed storage deletion must keep metadata")
	}
	store.removeFailures = 0
	repo.deleteErr = errors.New("database delete failed")
	if err := service.Delete(context.Background(), 1, photo.ID, 10); err == nil {
		t.Fatal("expected database error")
	}
	if len(repo.media) != 1 || len(store.objects) != 0 {
		t.Fatal("database failure must preserve metadata for retry")
	}
	repo.deleteErr = nil
	if err := service.Delete(context.Background(), 1, photo.ID, 10); err != nil {
		t.Fatal(err)
	}
	if len(repo.media) != 0 {
		t.Fatal("retry did not remove metadata")
	}
}

func TestMediaListInactiveAndEmpty(t *testing.T) {
	repo := newMemoryMediaRepository()
	service := NewEventMediaService(repo, newMemoryStorage())
	media, err := service.List(context.Background(), 1)
	if err != nil || media == nil || len(media) != 0 {
		t.Fatalf("empty list=%v err=%v", media, err)
	}
	event := repo.events[1]
	event.IsActive = false
	repo.events[1] = event
	if _, err := service.List(context.Background(), 1); !errors.Is(err, interfaces.ErrMediaEventNotFound) {
		t.Fatal(err)
	}
	if _, err := service.List(context.Background(), 999); !errors.Is(err, interfaces.ErrMediaEventNotFound) {
		t.Fatal(err)
	}
}

func TestMediaUploadCleanupExhaustion(t *testing.T) {
	repo, store := newMemoryMediaRepository(), newMemoryStorage()
	failedInsert := errors.New("metadata insertion failed")
	repo.insertErr = failedInsert
	store.removeFailures = -1
	_, err := NewEventMediaService(repo, store).Upload(context.Background(), 1, 10, mediaPNG(t))
	if !errors.Is(err, failedInsert) || store.removes != 3 || len(repo.media) != 0 || len(store.objects) != 1 {
		t.Fatalf("err=%v attempts=%d rows=%d remainingObjects=%d", err, store.removes, len(repo.media), len(store.objects))
	}
}

func TestMediaDeleteCommitFailureCanBeRetried(t *testing.T) {
	repo, store := newMemoryMediaRepository(), newMemoryStorage()
	service := NewEventMediaService(repo, store)
	photo, err := service.Upload(context.Background(), 1, 10, mediaPNG(t))
	if err != nil {
		t.Fatal(err)
	}
	repo.commitErr = interfaces.ErrMediaCommitUnknown
	if err := service.Delete(context.Background(), 1, photo.ID, 10); !errors.Is(err, interfaces.ErrMediaCommitUnknown) {
		t.Fatal(err)
	}
	if len(repo.media) != 1 || len(store.objects) != 0 {
		t.Fatal("a failed commit must leave metadata available for retry")
	}
	repo.commitErr = nil
	if err := service.Delete(context.Background(), 1, photo.ID, 10); err != nil {
		t.Fatal(err)
	}
	if len(repo.media) != 0 {
		t.Fatal("retry must delete retained metadata")
	}
}
