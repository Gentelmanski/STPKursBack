package entities

import "time"

type Comment struct {
	ID        uint      `json:"id"`
	Content   string    `json:"content"`
	EventID   uint      `json:"event_id"`
	UserID    uint      `json:"user_id"`
	ParentID  *uint     `json:"parent_id"`
	Score     int       `json:"score"`
	IsDeleted bool      `json:"is_deleted"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	User      User      `json:"user" gorm:"-"`
	Replies   []Comment `json:"replies" gorm:"-"`
}
