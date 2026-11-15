package dto

import "time"

// FileURL is a temporary viewing URL; it is never persisted in event_media.
type EventMediaResponse struct {
	ID           uint      `json:"id"`
	EventID      uint      `json:"event_id"`
	FileURL      string    `json:"file_url"`
	FileType     string    `json:"file_type"`
	OrderIndex   int       `json:"order_index"`
	UploadedAt   time.Time `json:"uploaded_at"`
	CreatedAt    time.Time `json:"created_at"`
	URLExpiresAt time.Time `json:"url_expires_at"`
}
