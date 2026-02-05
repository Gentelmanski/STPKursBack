package controllers

import (
	"net/http"
	"strconv"

	"auth-system/internal/application/dto"
	"auth-system/internal/application/interfaces"

	"github.com/gin-gonic/gin"
)

// test
type EventController struct {
	eventService interfaces.EventService
}

func NewEventController(eventService interfaces.EventService) *EventController {
	return &EventController{eventService: eventService}
}

func (c *EventController) GetEvents(ctx *gin.Context) {
	var filter dto.EventFilter
	if err := ctx.ShouldBindQuery(&filter); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	events, err := c.eventService.GetEvents(ctx.Request.Context(), filter)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, events)
}

func (c *EventController) GetEventByID(ctx *gin.Context) {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid event ID"})
		return
	}

	event, err := c.eventService.GetEventByID(ctx.Request.Context(), uint(id))
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, event)
}

func (c *EventController) CreateEvent(ctx *gin.Context) {
	var req dto.CreateEventRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := ctx.Get("user_id")
	event, err := c.eventService.CreateEvent(ctx.Request.Context(), req, userID.(uint))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, event)
}

func (c *EventController) UpdateEvent(ctx *gin.Context) {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid event ID"})
		return
	}

	var req dto.UpdateEventRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := ctx.Get("user_id")
	event, err := c.eventService.UpdateEvent(ctx.Request.Context(), uint(id), req, userID.(uint))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, event)
}

func (c *EventController) DeleteEvent(ctx *gin.Context) {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid event ID"})
		return
	}

	userID, _ := ctx.Get("user_id")
	err = c.eventService.DeleteEvent(ctx.Request.Context(), uint(id), userID.(uint))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Event deleted successfully"})
}

func (c *EventController) Participate(ctx *gin.Context) {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid event ID"})
		return
	}

	userID, _ := ctx.Get("user_id")
	err = c.eventService.Participate(ctx.Request.Context(), uint(id), userID.(uint))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Successfully joined event"})
}

func (c *EventController) CancelParticipation(ctx *gin.Context) {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid event ID"})
		return
	}

	userID, _ := ctx.Get("user_id")
	err = c.eventService.CancelParticipation(ctx.Request.Context(), uint(id), userID.(uint))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Successfully cancelled participation"})
}

func (c *EventController) GetUserEvents(ctx *gin.Context) {
	userID, _ := ctx.Get("user_id")
	events, err := c.eventService.GetUserEvents(ctx.Request.Context(), userID.(uint))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, events)
}

func (c *EventController) GetParticipatedEvents(ctx *gin.Context) {
	userID, _ := ctx.Get("user_id")
	events, err := c.eventService.GetParticipatedEvents(ctx.Request.Context(), userID.(uint))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, events)
}

func (c *EventController) FilterEvents(ctx *gin.Context) {
	var filter dto.EventFilter
	if err := ctx.ShouldBindJSON(&filter); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	events, err := c.eventService.FilterEvents(ctx.Request.Context(), filter)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, events)
}
