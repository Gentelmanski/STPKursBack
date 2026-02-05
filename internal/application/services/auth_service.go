package services

import (
	"context"
	"errors"
	"time"

	"auth-system/internal/application/dto"
	appInterfaces "auth-system/internal/application/interfaces"
	"auth-system/internal/domain/entities"
	"auth-system/internal/pkg/utils"
)

type AuthService struct {
	userRepo     appInterfaces.UserRepository
	jwtUtil      utils.JWTUtil
	passwordUtil utils.PasswordUtil
}

func NewAuthService(userRepo appInterfaces.UserRepository, jwtUtil utils.JWTUtil, passwordUtil utils.PasswordUtil) *AuthService {
	return &AuthService{
		userRepo:     userRepo,
		jwtUtil:      jwtUtil,
		passwordUtil: passwordUtil,
	}
}

func (s *AuthService) Register(ctx context.Context, req dto.RegisterRequest) (*dto.AuthResponse, error) {
	// Check if user exists
	existingUser, _ := s.userRepo.FindByEmail(ctx, req.Email)
	if existingUser != nil {
		return nil, errors.New("user already exists")
	}

	// Hash password
	hashedPassword, err := s.passwordUtil.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	// Validate role
	role := req.Role
	if role != "user" && role != "admin" {
		role = "user"
	}

	// Create user
	user := &entities.User{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: hashedPassword,
		Role:         role,
		LastOnline:   time.Now(),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	// Generate token
	token, err := s.jwtUtil.GenerateToken(user.ID, user.Email, user.Role)
	if err != nil {
		return nil, err
	}

	return &dto.AuthResponse{
		Token: token,
		User: dto.UserResponse{
			ID:         user.ID,
			Username:   user.Username,
			Email:      user.Email,
			Role:       user.Role,
			AvatarURL:  user.AvatarURL,
			IsBlocked:  user.IsBlocked,
			LastOnline: user.LastOnline.Format(time.RFC3339),
			CreatedAt:  user.CreatedAt.Format(time.RFC3339),
		},
	}, nil
}

func (s *AuthService) Login(ctx context.Context, req dto.LoginRequest) (*dto.AuthResponse, error) {
	// Find user
	user, err := s.userRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	// Check if user is blocked
	if user.IsBlocked {
		return nil, errors.New("account is blocked")
	}

	// Check password
	if !s.passwordUtil.CheckPasswordHash(req.Password, user.PasswordHash) {
		return nil, errors.New("invalid credentials")
	}

	// Update last online
	user.LastOnline = time.Now()
	s.userRepo.Update(ctx, user)

	// Generate token
	token, err := s.jwtUtil.GenerateToken(user.ID, user.Email, user.Role)
	if err != nil {
		return nil, err
	}

	return &dto.AuthResponse{
		Token: token,
		User: dto.UserResponse{
			ID:         user.ID,
			Username:   user.Username,
			Email:      user.Email,
			Role:       user.Role,
			AvatarURL:  user.AvatarURL,
			IsBlocked:  user.IsBlocked,
			LastOnline: user.LastOnline.Format(time.RFC3339),
			CreatedAt:  user.CreatedAt.Format(time.RFC3339),
		},
	}, nil
}

func (s *AuthService) GetProfile(ctx context.Context, userID uint) (*dto.UserResponse, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &dto.UserResponse{
		ID:         user.ID,
		Username:   user.Username,
		Email:      user.Email,
		Role:       user.Role,
		AvatarURL:  user.AvatarURL,
		IsBlocked:  user.IsBlocked,
		LastOnline: user.LastOnline.Format(time.RFC3339),
		CreatedAt:  user.CreatedAt.Format(time.RFC3339),
	}, nil
}

func (s *AuthService) UpdateLastOnline(ctx context.Context, userID uint) error {
	return s.userRepo.UpdateLastOnline(ctx, userID)
}
