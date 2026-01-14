package models

import (
	"time"
)

type Event struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	Title           string    `gorm:"not null" json:"title"`
	Description     string    `gorm:"type:text;not null" json:"description"`
	EventDate       time.Time `json:"event_date"`
	Location        string    `gorm:"type:geography(Point,4326)" json:"-"`
	Latitude        float64   `gorm:"-" json:"latitude"`
	Longitude       float64   `gorm:"-" json:"longitude"`
	Type            string    `gorm:"not null;check:type IN ('concert','exhibition','meetup','workshop','sport','festival','other')" json:"type"`
	MaxParticipants *int      `json:"max_participants"`
	Price           float64   `json:"price"`
	Address         string    `gorm:"type:text" json:"address"`
	IsVerified      bool      `gorm:"default:false" json:"is_verified"`
	IsActive        bool      `gorm:"default:true" json:"is_active"`
	CreatorID       uint      `json:"creator_id"`
	Creator         User      `gorm:"foreignKey:CreatorID" json:"creator"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`

	Tags         []Tag              `gorm:"many2many:event_tags;" json:"tags"`
	Media        []EventMedia       `gorm:"foreignKey:EventID" json:"media"`
	Participants []EventParticipant `gorm:"foreignKey:EventID" json:"participants"`
	Comments     []Comment          `gorm:"foreignKey:EventID" json:"comments"`

	// Добавляем поле для хранения количества участников
	ParticipantsCount int `gorm:"-" json:"participants_count"`
}

type Tag struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"unique;not null" json:"name"`
	Slug      string    `gorm:"unique;not null" json:"slug"`
	CreatedAt time.Time `json:"created_at"`
}

type EventTag struct {
	EventID uint `gorm:"primaryKey" json:"event_id"`
	TagID   uint `gorm:"primaryKey" json:"tag_id"`
}

type EventMedia struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	EventID    uint      `json:"event_id"`
	FileURL    string    `gorm:"not null" json:"file_url"`
	FileType   string    `gorm:"not null" json:"file_type"`
	OrderIndex int       `gorm:"default:0" json:"order_index"`
	CreatedAt  time.Time `json:"created_at"`
}

type EventParticipant struct {
	EventID  uint      `gorm:"primaryKey" json:"event_id"`
	UserID   uint      `gorm:"primaryKey" json:"user_id"`
	Status   string    `gorm:"not null;default:'going';check:status IN ('going','maybe','declined')" json:"status"`
	JoinedAt time.Time `json:"joined_at"`

	User User `gorm:"foreignKey:UserID" json:"user"`
}

type Comment struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Content   string    `gorm:"type:text;not null" json:"content"`
	EventID   uint      `json:"event_id"`
	UserID    uint      `json:"user_id"`
	ParentID  *uint     `json:"parent_id"`
	Score     int       `gorm:"default:0" json:"score"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	IsDeleted bool      `gorm:"default:false" json:"is_deleted"`

	User    User      `gorm:"foreignKey:UserID" json:"user"`
	Replies []Comment `gorm:"foreignKey:ParentID" json:"replies"`
}

type CommentVote struct {
	UserID    uint      `gorm:"primaryKey" json:"user_id"`
	CommentID uint      `gorm:"primaryKey" json:"comment_id"`
	VoteType  string    `gorm:"not null;check:vote_type IN ('upvote','downvote')" json:"vote_type"`
	VotedAt   time.Time `json:"voted_at"`
}

type Notification struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `json:"user_id"`
	Message   string    `json:"message"`
	Type      string    `json:"type"` // event_created, event_updated, comment_added, participation, system
	Read      bool      `gorm:"default:false" json:"read"`
	CreatedAt time.Time `json:"created_at"`
}

type AdminAction struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	AdminID     uint      `json:"admin_id"`
	ActionType  string    `gorm:"not null;check:action_type IN ('verify_event','reject_event','delete_event','delete_comment','block_user','unblock_user')" json:"action_type"`
	TargetID    uint      `json:"target_id"`
	TargetType  string    `gorm:"not null;check:target_type IN ('event','user','comment')" json:"target_type"`
	Reason      string    `json:"reason"`
	PerformedAt time.Time `json:"performed_at"`
}
