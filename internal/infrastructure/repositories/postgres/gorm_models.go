package postgres

import (
	"time"
)

// GORM модели для миграции
type UserModel struct {
	ID           uint      `gorm:"primaryKey;column:id"`
	Username     string    `gorm:"unique;not null;column:username"`
	Email        string    `gorm:"unique;not null;column:email"`
	PasswordHash string    `gorm:"not null;column:password_hash"`
	Role         string    `gorm:"not null;default:'user';column:role"`
	AvatarURL    string    `gorm:"column:avatar_url"`
	IsBlocked    bool      `gorm:"default:false;column:is_blocked"`
	LastOnline   time.Time `gorm:"column:last_online"`
	CreatedAt    time.Time `gorm:"column:created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at"`
}

func (UserModel) TableName() string {
	return "users"
}

type EventModel struct {
	ID              uint      `gorm:"primaryKey;column:id"`
	Title           string    `gorm:"not null;column:title"`
	Description     string    `gorm:"type:text;not null;column:description"`
	EventDate       time.Time `gorm:"column:event_date"`
	Location        string    `gorm:"type:geography(Point,4326);column:location"`
	Type            string    `gorm:"not null;column:type"`
	MaxParticipants *int      `gorm:"column:max_participants"`
	Price           float64   `gorm:"type:numeric(10,2);column:price"`
	Address         string    `gorm:"type:text;column:address"`
	IsVerified      bool      `gorm:"default:false;column:is_verified"`
	IsActive        bool      `gorm:"default:true;column:is_active"`
	CreatorID       uint      `gorm:"column:creator_id"`
	CreatedAt       time.Time `gorm:"column:created_at"`
	UpdatedAt       time.Time `gorm:"column:updated_at"`
}

func (EventModel) TableName() string {
	return "events"
}

type TagModel struct {
	ID        uint      `gorm:"primaryKey;column:id"`
	Name      string    `gorm:"unique;not null;column:name"`
	Slug      string    `gorm:"unique;not null;column:slug"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

func (TagModel) TableName() string {
	return "tags"
}

type EventTagModel struct {
	EventID uint `gorm:"primaryKey;column:event_id"`
	TagID   uint `gorm:"primaryKey;column:tag_id"`
}

func (EventTagModel) TableName() string {
	return "event_tags"
}

type EventMediaModel struct {
	ID         uint      `gorm:"primaryKey;column:id"`
	EventID    uint      `gorm:"column:event_id"`
	FileURL    string    `gorm:"not null;column:file_url"`
	FileType   string    `gorm:"not null;column:file_type"`
	OrderIndex int       `gorm:"default:0;column:order_index"`
	UploadedAt time.Time `gorm:"not null;default:now();column:uploaded_at"`
	CreatedAt  time.Time `gorm:"column:created_at"`
}

func (EventMediaModel) TableName() string {
	return "event_media"
}

type EventParticipantModel struct {
	EventID  uint      `gorm:"primaryKey;column:event_id"`
	UserID   uint      `gorm:"primaryKey;column:user_id"`
	Status   string    `gorm:"not null;default:'going';column:status"`
	JoinedAt time.Time `gorm:"column:joined_at"`
}

func (EventParticipantModel) TableName() string {
	return "event_participants"
}

type CommentModel struct {
	ID        uint      `gorm:"primaryKey;column:id"`
	Content   string    `gorm:"type:text;not null;column:content"`
	EventID   uint      `gorm:"column:event_id"`
	UserID    uint      `gorm:"column:user_id"`
	ParentID  *uint     `gorm:"column:parent_id"`
	Score     int       `gorm:"default:0;column:score"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
	IsDeleted bool      `gorm:"default:false;column:is_deleted"`
}

func (CommentModel) TableName() string {
	return "comments"
}

type CommentVoteModel struct {
	UserID    uint      `gorm:"primaryKey;column:user_id"`
	CommentID uint      `gorm:"primaryKey;column:comment_id"`
	VoteType  string    `gorm:"not null;column:vote_type"`
	VotedAt   time.Time `gorm:"column:voted_at"`
}

func (CommentVoteModel) TableName() string {
	return "comment_votes"
}

type NotificationModel struct {
	ID        uint      `gorm:"primaryKey;column:id"`
	UserID    uint      `gorm:"column:user_id"`
	Message   string    `gorm:"type:text;column:message"`
	Type      string    `gorm:"column:type"`
	Read      bool      `gorm:"default:false;column:read"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

func (NotificationModel) TableName() string {
	return "notifications"
}

type AdminActionModel struct {
	ID          uint      `gorm:"primaryKey;column:id"`
	AdminID     uint      `gorm:"column:admin_id"`
	ActionType  string    `gorm:"not null;column:action_type"`
	TargetID    uint      `gorm:"column:target_id"`
	TargetType  string    `gorm:"not null;column:target_type"`
	Reason      string    `gorm:"type:text;column:reason"`
	PerformedAt time.Time `gorm:"column:performed_at"`
}

func (AdminActionModel) TableName() string {
	return "admin_actions"
}
