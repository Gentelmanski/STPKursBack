package entities

import "time"

type Event struct {
	ID                uint               `json:"id"`
	Title             string             `json:"title"`
	Description       string             `json:"description"`
	EventDate         time.Time          `json:"event_date"`
	Latitude          float64            `json:"latitude"`
	Longitude         float64            `json:"longitude"`
	Type              string             `json:"type"`
	MaxParticipants   *int               `json:"max_participants"`
	Price             float64            `json:"price"`
	Address           string             `json:"address"`
	IsVerified        bool               `json:"is_verified"`
	IsActive          bool               `json:"is_active"`
	CreatorID         uint               `json:"creator_id"`
	Creator           User               `json:"creator"`
	CreatedAt         time.Time          `json:"created_at"`
	UpdatedAt         time.Time          `json:"updated_at"`
	ParticipantsCount int                `json:"participants_count"`
	Tags              []Tag              `json:"tags"`
	Media             []EventMedia       `json:"media"`
	Participants      []EventParticipant `json:"participants"`
	Comments          []Comment          `json:"comments"`
}

func (e *Event) IsFull() bool {
	if e.MaxParticipants == nil {
		return false
	}
	return e.ParticipantsCount >= *e.MaxParticipants
}

func (e *Event) CanEdit(userID uint) bool {
	return e.CreatorID == userID
}
