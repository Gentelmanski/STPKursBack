package postgres

import (
	"time"
)

// GORM модели для миграции (без бизнес-логики)
type UserModel struct {
	ID           uint   `gorm:"primaryKey"`
	Username     string `gorm:"unique;not null"`
	Email        string `gorm:"unique;not null"`
	PasswordHash string `gorm:"not null"`
	Role         string `gorm:"not null;default:'user'"`
	AvatarURL    string
	IsBlocked    bool `gorm:"default:false"`
	LastOnline   time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type EventModel struct {
	ID              uint   `gorm:"primaryKey"`
	Title           string `gorm:"not null"`
	Description     string `gorm:"type:text;not null"`
	EventDate       time.Time
	Location        string `gorm:"type:geography(Point,4326)"`
	Type            string `gorm:"not null"`
	MaxParticipants *int
	Price           float64
	Address         string `gorm:"type:text"`
	IsVerified      bool   `gorm:"default:false"`
	IsActive        bool   `gorm:"default:true"`
	CreatorID       uint
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type TagModel struct {
	ID        uint   `gorm:"primaryKey"`
	Name      string `gorm:"unique;not null"`
	Slug      string `gorm:"unique;not null"`
	CreatedAt time.Time
}

type EventTagModel struct {
	EventID uint `gorm:"primaryKey"`
	TagID   uint `gorm:"primaryKey"`
}

type EventMediaModel struct {
	ID         uint `gorm:"primaryKey"`
	EventID    uint
	FileURL    string `gorm:"not null"`
	FileType   string `gorm:"not null"`
	OrderIndex int    `gorm:"default:0"`
	CreatedAt  time.Time
}

type EventParticipantModel struct {
	EventID  uint   `gorm:"primaryKey"`
	UserID   uint   `gorm:"primaryKey"`
	Status   string `gorm:"not null;default:'going'"`
	JoinedAt time.Time
}

type CommentModel struct {
	ID        uint   `gorm:"primaryKey"`
	Content   string `gorm:"type:text;not null"`
	EventID   uint
	UserID    uint
	ParentID  *uint
	Score     int `gorm:"default:0"`
	CreatedAt time.Time
	UpdatedAt time.Time
	IsDeleted bool `gorm:"default:false"`
}

type CommentVoteModel struct {
	UserID    uint   `gorm:"primaryKey"`
	CommentID uint   `gorm:"primaryKey"`
	VoteType  string `gorm:"not null"`
	VotedAt   time.Time
}

type NotificationModel struct {
	ID        uint `gorm:"primaryKey"`
	UserID    uint
	Message   string `gorm:"type:text"`
	Type      string
	Read      bool `gorm:"default:false"`
	CreatedAt time.Time
}

type AdminActionModel struct {
	ID          uint `gorm:"primaryKey"`
	AdminID     uint
	ActionType  string `gorm:"not null"`
	TargetID    uint
	TargetType  string `gorm:"not null"`
	Reason      string `gorm:"type:text"`
	PerformedAt time.Time
}
