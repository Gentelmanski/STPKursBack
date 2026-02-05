package repositories

import (
	"context"
	"time"

	"auth-system/internal/application/interfaces"
	"auth-system/internal/domain/entities"

	"gorm.io/gorm"
)

type CommentRepository struct {
	db *gorm.DB
}

func NewCommentRepository(db *gorm.DB) interfaces.CommentRepository {
	return &CommentRepository{db: db}
}

func (r *CommentRepository) Create(ctx context.Context, comment *entities.Comment) error {
	return r.db.WithContext(ctx).Create(comment).Error
}

func (r *CommentRepository) FindByID(ctx context.Context, id uint) (*entities.Comment, error) {
	var comment entities.Comment
	err := r.db.WithContext(ctx).
		Preload("User").
		Preload("Replies", "is_deleted = ?", false).
		Preload("Replies.User").
		First(&comment, id).Error
	if err != nil {
		return nil, err
	}
	return &comment, nil
}

func (r *CommentRepository) FindByEventID(ctx context.Context, eventID uint) ([]entities.Comment, error) {
	var comments []entities.Comment
	err := r.db.WithContext(ctx).
		Preload("User").
		Preload("Replies", "is_deleted = ?", false).
		Preload("Replies.User").
		Where("event_id = ? AND parent_id IS NULL AND is_deleted = ?", eventID, false).
		Order("created_at DESC").
		Find(&comments).Error
	return comments, err
}

func (r *CommentRepository) Update(ctx context.Context, comment *entities.Comment) error {
	return r.db.WithContext(ctx).Save(comment).Error
}

func (r *CommentRepository) SoftDelete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Model(&entities.Comment{}).
		Where("id = ?", id).
		Update("is_deleted", true).Error
}

func (r *CommentRepository) Vote(ctx context.Context, commentID, userID uint, voteType string) error {
	// Проверяем существование голоса
	var existingVote entities.CommentVote
	err := r.db.WithContext(ctx).
		Where("comment_id = ? AND user_id = ?", commentID, userID).
		First(&existingVote).Error

	if err == nil {
		// Обновляем существующий голос
		existingVote.VoteType = voteType
		existingVote.VotedAt = time.Now()
		return r.db.WithContext(ctx).Save(&existingVote).Error
	}

	// Создаем новый голос
	vote := &entities.CommentVote{
		UserID:    userID,
		CommentID: commentID,
		VoteType:  voteType,
		VotedAt:   time.Now(),
	}
	return r.db.WithContext(ctx).Create(vote).Error
}

func (r *CommentRepository) GetVote(ctx context.Context, commentID, userID uint) (*entities.CommentVote, error) {
	var vote entities.CommentVote
	err := r.db.WithContext(ctx).
		Where("comment_id = ? AND user_id = ?", commentID, userID).
		First(&vote).Error
	if err != nil {
		return nil, err
	}
	return &vote, nil
}

func (r *CommentRepository) GetScore(ctx context.Context, commentID uint) (int, error) {
	var upvotes, downvotes int64

	// Считаем upvotes
	r.db.WithContext(ctx).Model(&entities.CommentVote{}).
		Where("comment_id = ? AND vote_type = ?", commentID, "upvote").
		Count(&upvotes)

	// Считаем downvotes
	r.db.WithContext(ctx).Model(&entities.CommentVote{}).
		Where("comment_id = ? AND vote_type = ?", commentID, "downvote").
		Count(&downvotes)

	return int(upvotes - downvotes), nil
}

// Новые методы, которые нужно добавить:

func (r *CommentRepository) DeleteVote(ctx context.Context, commentID, userID uint) error {
	return r.db.WithContext(ctx).
		Where("comment_id = ? AND user_id = ?", commentID, userID).
		Delete(&entities.CommentVote{}).Error
}

func (r *CommentRepository) UpdateVote(ctx context.Context, vote *entities.CommentVote) error {
	return r.db.WithContext(ctx).Save(vote).Error
}

func (r *CommentRepository) CreateVote(ctx context.Context, vote *entities.CommentVote) error {
	return r.db.WithContext(ctx).Create(vote).Error
}
