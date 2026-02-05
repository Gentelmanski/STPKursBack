package dto

import "time"

type CreateEventRequest struct {
	Title           string    `json:"title" binding:"required"`
	Description     string    `json:"description" binding:"required"`
	EventDate       time.Time `json:"event_date" binding:"required"`
	Latitude        float64   `json:"latitude" binding:"required"`
	Longitude       float64   `json:"longitude" binding:"required"`
	Type            string    `json:"type" binding:"required,oneof=concert exhibition meetup workshop sport festival other"`
	MaxParticipants *int      `json:"max_participants"`
	Price           float64   `json:"price"`
	Tags            []string  `json:"tags"`
	Address         string    `json:"address"`
}

type UpdateEventRequest struct {
	Title           string    `json:"title"`
	Description     string    `json:"description"`
	EventDate       time.Time `json:"event_date"`
	Type            string    `json:"type" oneof:"concert,exhibition,meetup,workshop,sport,festival,other"`
	MaxParticipants *int      `json:"max_participants"`
	Price           float64   `json:"price"`
}

type EventFilter struct {
	Type []string  `json:"type"`
	Date time.Time `json:"date"`
	Tags string    `json:"tags"`
}

type EventResponse struct {
	ID                uint      `json:"id"`
	Title             string    `json:"title"`
	Description       string    `json:"description"`
	EventDate         time.Time `json:"event_date"`
	Latitude          float64   `json:"latitude"`
	Longitude         float64   `json:"longitude"`
	Type              string    `json:"type"`
	MaxParticipants   *int      `json:"max_participants"`
	Price             float64   `json:"price"`
	Address           string    `json:"address"`
	IsVerified        bool      `json:"is_verified"`
	IsActive          bool      `json:"is_active"`
	CreatorID         uint      `json:"creator_id"`
	Creator           UserShort `json:"creator"`
	ParticipantsCount int       `json:"participants_count"`
	CreatedAt         string    `json:"created_at"`
	UpdatedAt         string    `json:"updated_at"`
	Tags              []Tag     `json:"tags"`
	Media             []Media   `json:"media"`
}

type Tag struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type Media struct {
	ID         uint   `json:"id"`
	FileURL    string `json:"file_url"`
	FileType   string `json:"file_type"`
	OrderIndex int    `json:"order_index"`
}

type EventParticipantResponse struct {
	EventID  uint       `json:"event_id"`
	UserID   uint       `json:"user_id"`
	Status   string     `json:"status"`
	JoinedAt string     `json:"joined_at"`
	User     UserShort  `json:"user"`
	Event    EventShort `json:"event"`
}

type EventShort struct {
	ID       uint   `json:"id"`
	Title    string `json:"title"`
	Type     string `json:"type"`
	IsActive bool   `json:"is_active"`
}
