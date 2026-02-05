package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"auth-system/internal/application/dto"
	appInterfaces "auth-system/internal/application/interfaces"
	"auth-system/internal/domain/entities"
)

type CommentService struct {
	commentRepo      appInterfaces.CommentRepository
	userRepo         appInterfaces.UserRepository
	eventRepo        appInterfaces.EventRepository
	notificationRepo appInterfaces.NotificationRepository
}

func NewCommentService(
	commentRepo appInterfaces.CommentRepository,
	userRepo appInterfaces.UserRepository,
	eventRepo appInterfaces.EventRepository,
	notificationRepo appInterfaces.NotificationRepository,
) *CommentService {
	return &CommentService{
		commentRepo:      commentRepo,
		userRepo:         userRepo,
		eventRepo:        eventRepo,
		notificationRepo: notificationRepo,
	}
}

func (s *CommentService) CreateComment(ctx context.Context, req dto.CreateCommentRequest, eventID, userID uint) (*dto.CommentResponse, error) {
	// Проверяем существование мероприятия
	event, err := s.eventRepo.FindByID(ctx, eventID)
	if err != nil {
		return nil, errors.New("event not found")
	}

	if !event.IsActive {
		return nil, errors.New("event is not active")
	}

	// Если есть parent_id, проверяем существование родительского комментария
	if req.ParentID != nil {
		parentComment, err := s.commentRepo.FindByID(ctx, *req.ParentID)
		if err != nil || parentComment.IsDeleted {
			return nil, errors.New("parent comment not found")
		}
	}

	// Создаем комментарий
	comment := &entities.Comment{
		Content:   req.Content,
		EventID:   eventID,
		UserID:    userID,
		ParentID:  req.ParentID,
		Score:     0,
		IsDeleted: false,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.commentRepo.Create(ctx, comment); err != nil {
		return nil, err
	}

	// Загружаем пользователя для комментария
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	comment.User = *user

	// Создаем уведомление для создателя мероприятия, если это не он сам
	if event.CreatorID != userID {
		notification := &entities.Notification{
			UserID:    event.CreatorID,
			Message:   fmt.Sprintf("Новый комментарий к вашему мероприятию: %s", event.Title),
			Type:      "comment_added",
			Read:      false,
			CreatedAt: time.Now(),
		}
		s.notificationRepo.Create(ctx, notification)
	}

	// Если это ответ на комментарий, уведомляем автора родительского комментария
	if req.ParentID != nil {
		parentComment, err := s.commentRepo.FindByID(ctx, *req.ParentID)
		if err == nil && parentComment.UserID != userID && parentComment.UserID != event.CreatorID {
			notification := &entities.Notification{
				UserID:    parentComment.UserID,
				Message:   fmt.Sprintf("Ответ на ваш комментарий в мероприятии: %s", event.Title),
				Type:      "comment_reply",
				Read:      false,
				CreatedAt: time.Now(),
			}
			s.notificationRepo.Create(ctx, notification)
		}
	}

	return s.commentToDTO(comment), nil
}

func (s *CommentService) GetComments(ctx context.Context, eventID uint) ([]dto.CommentResponse, error) {
	comments, err := s.commentRepo.FindByEventID(ctx, eventID)
	if err != nil {
		return nil, err
	}

	// Преобразуем в DTO
	response := make([]dto.CommentResponse, len(comments))
	for i, comment := range comments {
		response[i] = *s.commentToDTO(&comment)
	}

	return response, nil
}

func (s *CommentService) UpdateComment(ctx context.Context, commentID uint, req dto.UpdateCommentRequest, userID uint) (*dto.CommentResponse, error) {
	comment, err := s.commentRepo.FindByID(ctx, commentID)
	if err != nil {
		return nil, errors.New("comment not found")
	}

	if comment.UserID != userID {
		return nil, errors.New("not authorized to update this comment")
	}

	if comment.IsDeleted {
		return nil, errors.New("comment is deleted")
	}

	comment.Content = req.Content
	comment.UpdatedAt = time.Now()

	if err := s.commentRepo.Update(ctx, comment); err != nil {
		return nil, err
	}

	// Загружаем пользователя
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	comment.User = *user

	return s.commentToDTO(comment), nil
}

func (s *CommentService) DeleteComment(ctx context.Context, commentID, userID uint) error {
	comment, err := s.commentRepo.FindByID(ctx, commentID)
	if err != nil {
		return errors.New("comment not found")
	}

	if comment.UserID != userID {
		return errors.New("not authorized to delete this comment")
	}

	// Мягкое удаление
	comment.IsDeleted = true
	comment.UpdatedAt = time.Now()

	return s.commentRepo.Update(ctx, comment)
}

func (s *CommentService) VoteComment(ctx context.Context, commentID, userID uint, voteType string) (*dto.CommentResponse, error) {
	// Проверяем существование комментария
	comment, err := s.commentRepo.FindByID(ctx, commentID)
	if err != nil {
		return nil, errors.New("comment not found")
	}

	if comment.IsDeleted {
		return nil, errors.New("comment is deleted")
	}

	// Проверяем, не голосовал ли уже пользователь
	existingVote, err := s.commentRepo.GetVote(ctx, commentID, userID)
	if err == nil {
		// Если голос уже есть, обновляем его
		if existingVote.VoteType == voteType {
			// Если тот же тип голоса, удаляем голос (отмена голоса)
			s.commentRepo.DeleteVote(ctx, commentID, userID)
		} else {
			// Иначе обновляем тип голоса
			existingVote.VoteType = voteType
			existingVote.VotedAt = time.Now()
			s.commentRepo.UpdateVote(ctx, existingVote)
		}
	} else {
		// Создаем новый голос
		vote := &entities.CommentVote{
			UserID:    userID,
			CommentID: commentID,
			VoteType:  voteType,
			VotedAt:   time.Now(),
		}
		s.commentRepo.CreateVote(ctx, vote)
	}

	// Получаем обновленный счет
	score, err := s.commentRepo.GetScore(ctx, commentID)
	if err != nil {
		return nil, err
	}
	comment.Score = score

	// Загружаем пользователя
	user, err := s.userRepo.FindByID(ctx, comment.UserID)
	if err != nil {
		return nil, err
	}
	comment.User = *user

	return s.commentToDTO(comment), nil
}

func (s *CommentService) commentToDTO(comment *entities.Comment) *dto.CommentResponse {
	// Преобразуем ответы (реплики)
	replies := make([]dto.CommentResponse, len(comment.Replies))
	for i, reply := range comment.Replies {
		replies[i] = *s.commentToDTO(&reply)
	}

	return &dto.CommentResponse{
		ID:        comment.ID,
		Content:   comment.Content,
		EventID:   comment.EventID,
		UserID:    comment.UserID,
		ParentID:  comment.ParentID,
		Score:     comment.Score,
		IsDeleted: comment.IsDeleted,
		CreatedAt: comment.CreatedAt.Format(time.RFC3339),
		UpdatedAt: comment.UpdatedAt.Format(time.RFC3339),
		User: dto.UserShort{
			ID:       comment.User.ID,
			Username: comment.User.Username,
			Email:    comment.User.Email,
			Role:     comment.User.Role,
		},
		Replies: replies,
	}
}
