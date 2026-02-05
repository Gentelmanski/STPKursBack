package entities

import "time"

type Tag struct {
	ID        uint      `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	CreatedAt time.Time `json:"created_at"`
}

type EventTag struct {
	EventID uint `json:"event_id"`
	TagID   uint `json:"tag_id"`
}

type EventMedia struct {
	ID         uint      `json:"id"`
	EventID    uint      `json:"event_id"`
	FileURL    string    `json:"file_url"`
	FileType   string    `json:"file_type"`
	OrderIndex int       `json:"order_index"`
	UploadedAt time.Time `json:"uploaded_at"`
	CreatedAt  time.Time `json:"created_at"`
}

type EventParticipant struct {
	EventID  uint      `json:"event_id"`
	UserID   uint      `json:"user_id"`
	Status   string    `json:"status"`
	JoinedAt time.Time `json:"joined_at"`
	User     User      `json:"user" gorm:"-"`
	Event    Event     `json:"event" gorm:"-"`
}

type CommentVote struct {
	UserID    uint      `json:"user_id"`
	CommentID uint      `json:"comment_id"`
	VoteType  string    `json:"vote_type"`
	VotedAt   time.Time `json:"voted_at"`
}

type Notification struct {
	ID        uint      `json:"id"`
	UserID    uint      `json:"user_id"`
	Message   string    `json:"message"`
	Type      string    `json:"type"`
	Read      bool      `json:"read"`
	CreatedAt time.Time `json:"created_at"`
}

type AdminAction struct {
	ID          uint      `json:"id"`
	AdminID     uint      `json:"admin_id"`
	ActionType  string    `json:"action_type"`
	TargetID    uint      `json:"target_id"`
	TargetType  string    `json:"target_type"`
	Reason      string    `json:"reason"`
	PerformedAt time.Time `json:"performed_at"`
}
