package controllers

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"auth-system/internal/application/dto"
	"auth-system/internal/application/services"
	"auth-system/internal/infrastructure/http/middlewares"
	"auth-system/internal/pkg/utils"

	"github.com/gin-gonic/gin"
)

type mediaServiceStub struct {
	calls int
	err   error
	data  []byte
}

func (s *mediaServiceStub) Upload(ctx context.Context, eventID, userID uint, data []byte) (*dto.EventMediaResponse, error) {
	s.calls++
	s.data = data
	return &dto.EventMediaResponse{ID: 7, EventID: eventID}, s.err
}
func (s *mediaServiceStub) List(ctx context.Context, eventID uint) ([]dto.EventMediaResponse, error) {
	s.calls++
	return []dto.EventMediaResponse{}, s.err
}
func (s *mediaServiceStub) Delete(ctx context.Context, eventID, mediaID, userID uint) error {
	s.calls++
	return s.err
}

func mediaTestRouter(t *testing.T, service EventMediaService) (*gin.Engine, string) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	jwt := utils.NewJWTUtil("media-test-secret")
	token, err := jwt.GenerateToken(10, "media-test@example.invalid", "user")
	if err != nil {
		t.Fatal(err)
	}
	router := gin.New()
	router.Use(middlewares.AuthMiddleware(jwt))
	ctrl := NewEventMediaController(service)
	router.POST("/api/events/:id/media", ctrl.Upload)
	router.GET("/api/events/:id/media", ctrl.List)
	router.DELETE("/api/events/:id/media/:mediaId", ctrl.Delete)
	return router, token
}

func mediaMultipart(t *testing.T, files []string, data []byte, extraField bool) ([]byte, string) {
	t.Helper()
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	for _, field := range files {
		f, err := w.CreateFormFile(field, "untrusted.exe")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.Write(data); err != nil {
			t.Fatal(err)
		}
	}
	if extraField {
		if err := w.WriteField("other", "unexpected"); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return b.Bytes(), w.FormDataContentType()
}

func TestMediaControllerMultipartAndLimits(t *testing.T) {
	for _, tc := range []struct {
		name           string
		fields         []string
		size           int
		extra, chunked bool
		want           int
	}{
		{"one file", []string{"file"}, 8, false, false, 201},
		{"missing file", nil, 0, false, false, 400},
		{"wrong field", []string{"photo"}, 8, false, false, 400},
		{"two files", []string{"file", "file"}, 8, false, false, 400},
		{"unexpected field", []string{"file"}, 8, true, false, 400},
		{"file too large", []string{"file"}, services.MaxMediaFileSize + 1, false, false, 413},
		{"request too large", []string{"file"}, int(MaxMediaRequestSize) + 1, false, false, 413},
		{"chunked request too large", []string{"file"}, int(MaxMediaRequestSize) + 1, false, true, 413},
	} {
		t.Run(tc.name, func(t *testing.T) {
			service := &mediaServiceStub{}
			router, token := mediaTestRouter(t, service)
			body, contentType := mediaMultipart(t, tc.fields, make([]byte, tc.size), tc.extra)
			req := httptest.NewRequest("POST", "/api/events/1/media", bytes.NewReader(body))
			if tc.chunked {
				req.ContentLength = -1
				req.TransferEncoding = []string{"chunked"}
			}
			req.Header.Set("Content-Type", contentType)
			req.Header.Set("Authorization", "Bearer "+token)
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)
			if rr.Code != tc.want {
				t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
			}
			if (service.calls > 0) != (tc.want == 201) {
				t.Fatalf("service calls=%d", service.calls)
			}
		})
	}
}

func TestMediaControllerJWTAndIDs(t *testing.T) {
	for _, tc := range []struct {
		method, path, auth string
		want               int
	}{
		{"GET", "/api/events/1/media", "", 401},
		{"POST", "/api/events/1/media", "Bearer invalid", 401},
		{"DELETE", "/api/events/1/media/1", "", 401},
		{"GET", "/api/events/abc/media", "valid", 400},
		{"GET", "/api/events/0/media", "valid", 400},
		{"DELETE", "/api/events/1/media/-1", "valid", 400},
		{"GET", "/api/events/1/media", "valid", 200},
		{"DELETE", "/api/events/1/media/1", "valid", 204},
	} {
		t.Run(tc.method+tc.path+fmt.Sprint(tc.want), func(t *testing.T) {
			service := &mediaServiceStub{}
			router, token := mediaTestRouter(t, service)
			req := httptest.NewRequest(tc.method, tc.path, nil)
			auth := tc.auth
			if auth == "valid" {
				auth = "Bearer " + token
			}
			if auth != "" {
				req.Header.Set("Authorization", auth)
			}
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)
			if rr.Code != tc.want {
				t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
			}
			if tc.want == 200 && rr.Body.String() != `{"media":[]}` {
				t.Fatal(rr.Body.String())
			}
			if tc.want == 204 && rr.Body.Len() != 0 {
				t.Fatal("204 must have no body")
			}
			if tc.want >= 400 && service.calls != 0 {
				t.Fatal("invalid request reached service")
			}
		})
	}
}

func TestMediaControllerDoesNotExposeInternalErrors(t *testing.T) {
	service := &mediaServiceStub{err: errors.New("database password=DO-NOT-EXPOSE and signed URL")}
	router, token := mediaTestRouter(t, service)
	req := httptest.NewRequest("GET", "/api/events/1/media", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusInternalServerError || strings.Contains(rr.Body.String(), "DO-NOT-EXPOSE") {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}
