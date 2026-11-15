package controllers

import (
	"context"
	"errors"
	"io"
	"mime"
	"net/http"
	"strconv"

	"auth-system/internal/application/dto"
	"auth-system/internal/application/interfaces"
	"auth-system/internal/application/services"

	"github.com/gin-gonic/gin"
)

// The allowance covers multipart headers; the file itself is still limited to 5 MiB.
const MaxMediaRequestSize int64 = services.MaxMediaFileSize + 64*1024

type EventMediaService interface {
	Upload(ctx context.Context, eventID, userID uint, data []byte) (*dto.EventMediaResponse, error)
	List(ctx context.Context, eventID uint) ([]dto.EventMediaResponse, error)
	Delete(ctx context.Context, eventID, mediaID, userID uint) error
}

type EventMediaController struct{ service EventMediaService }

func NewEventMediaController(service EventMediaService) *EventMediaController {
	return &EventMediaController{service: service}
}

func mediaID(ctx *gin.Context, param string) (uint, bool) {
	id, err := strconv.ParseUint(ctx.Param(param), 10, 32)
	if err != nil || id == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid " + param})
		return 0, false
	}
	return uint(id), true
}

func mediaUser(ctx *gin.Context) (uint, bool) {
	id, ok := ctx.Get("user_id")
	userID, valid := id.(uint)
	if !ok || !valid || userID == 0 {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return 0, false
	}
	return userID, true
}

func (c *EventMediaController) Upload(ctx *gin.Context) {
	eventID, ok := mediaID(ctx, "id")
	if !ok {
		return
	}
	userID, ok := mediaUser(ctx)
	if !ok {
		return
	}
	ctx.Header("Cache-Control", "no-store")
	ctx.Request.Body = http.MaxBytesReader(ctx.Writer, ctx.Request.Body, MaxMediaRequestSize)
	if ctx.Request.ContentLength > MaxMediaRequestSize {
		ctx.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "request exceeds upload limit"})
		return
	}
	contentType, _, err := mime.ParseMediaType(ctx.GetHeader("Content-Type"))
	if err != nil || contentType != "multipart/form-data" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "expected multipart/form-data with one file field"})
		return
	}
	if err := ctx.Request.ParseMultipartForm(MaxMediaRequestSize); err != nil {
		if ctx.Request.MultipartForm != nil {
			defer ctx.Request.MultipartForm.RemoveAll()
		}
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			ctx.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "request exceeds upload limit"})
		} else {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid multipart form"})
		}
		return
	}
	form := ctx.Request.MultipartForm
	defer form.RemoveAll()
	if len(form.File) != 1 || len(form.File["file"]) != 1 || len(form.Value) != 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "expected exactly one file field and no other fields"})
		return
	}
	header := form.File["file"][0]
	if header.Size > services.MaxMediaFileSize {
		mediaError(ctx, services.ErrMediaTooLarge)
		return
	}
	file, err := header.Open()
	if err != nil {
		mediaError(ctx, err)
		return
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, services.MaxMediaFileSize+1))
	if err != nil {
		mediaError(ctx, err)
		return
	}
	response, err := c.service.Upload(ctx.Request.Context(), eventID, userID, data)
	if err != nil {
		mediaError(ctx, err)
		return
	}
	ctx.JSON(http.StatusCreated, response)
}

func (c *EventMediaController) List(ctx *gin.Context) {
	eventID, ok := mediaID(ctx, "id")
	if !ok {
		return
	}
	if _, ok := mediaUser(ctx); !ok {
		return
	}
	ctx.Header("Cache-Control", "no-store")
	media, err := c.service.List(ctx.Request.Context(), eventID)
	if err != nil {
		mediaError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"media": media})
}

func (c *EventMediaController) Delete(ctx *gin.Context) {
	eventID, ok := mediaID(ctx, "id")
	if !ok {
		return
	}
	id, ok := mediaID(ctx, "mediaId")
	if !ok {
		return
	}
	userID, ok := mediaUser(ctx)
	if !ok {
		return
	}
	if err := c.service.Delete(ctx.Request.Context(), eventID, id, userID); err != nil {
		mediaError(ctx, err)
		return
	}
	ctx.Status(http.StatusNoContent)
}

func mediaError(ctx *gin.Context, err error) {
	status, message := http.StatusInternalServerError, "could not complete photo operation"
	switch {
	case errors.Is(err, interfaces.ErrMediaEventNotFound):
		status, message = http.StatusNotFound, interfaces.ErrMediaEventNotFound.Error()
	case errors.Is(err, interfaces.ErrMediaNotFound):
		status, message = http.StatusNotFound, interfaces.ErrMediaNotFound.Error()
	case errors.Is(err, services.ErrMediaForbidden):
		status, message = http.StatusForbidden, services.ErrMediaForbidden.Error()
	case errors.Is(err, services.ErrMediaLimit):
		status, message = http.StatusConflict, services.ErrMediaLimit.Error()
	case errors.Is(err, services.ErrMediaTooLarge):
		status, message = http.StatusRequestEntityTooLarge, services.ErrMediaTooLarge.Error()
	case errors.Is(err, services.ErrMediaInvalidImage):
		status, message = http.StatusUnsupportedMediaType, services.ErrMediaInvalidImage.Error()
	case errors.Is(err, services.ErrMediaStorage):
		status, message = http.StatusServiceUnavailable, services.ErrMediaStorage.Error()
	}
	ctx.JSON(status, gin.H{"error": message})
}
