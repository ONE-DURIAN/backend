package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"community-backend/config"
	"community-backend/internal/domain"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type Service interface {
	Register(ctx context.Context, req domain.RegisterRequest) (*domain.AuthResponse, error)
	Login(ctx context.Context, req domain.LoginRequest) (*domain.AuthResponse, error)
	RefreshToken(ctx context.Context, refreshToken string) (*domain.TokenPair, error)
	Logout(ctx context.Context, refreshToken string) error
	GetProfile(ctx context.Context, userID string) (*domain.User, error)
	ValidateAccessToken(tokenStr string) (*domain.JWTClaims, error)
}

type authService struct {
	repo Repository
	cfg  config.Config
}

func NewService(repo Repository, cfg config.Config) Service {
	return &authService{
		repo: repo,
		cfg:  cfg,
	}
}

func (s *authService) Register(ctx context.Context, req domain.RegisterRequest) (*domain.AuthResponse, error) {
	req.PhoneNumber = strings.TrimSpace(req.PhoneNumber)
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	req.FullName = strings.TrimSpace(req.FullName)

	if req.PhoneNumber == "" && req.Email == "" {
		return nil, errors.New("either phone number or email must be provided")
	}
	if len(req.Password) < 6 {
		return nil, errors.New("password must be at least 6 characters long")
	}
	if req.FullName == "" {
		return nil, errors.New("full name is required")
	}

	// Default role to farmer
	if req.Role == "" {
		req.Role = domain.RoleFarmer
	}

	// Check if already exists
	if req.PhoneNumber != "" {
		if existing, _ := s.repo.GetUserByIdentifier(ctx, req.PhoneNumber); existing != nil {
			return nil, errors.New("phone number already registered")
		}
	}
	if req.Email != "" {
		if existing, _ := s.repo.GetUserByIdentifier(ctx, req.Email); existing != nil {
			return nil, errors.New("email already registered")
		}
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user := domain.User{
		PasswordHash: string(hashedPassword),
		FullName:     req.FullName,
		Role:         req.Role,
	}
	if req.PhoneNumber != "" {
		user.PhoneNumber = &req.PhoneNumber
	}
	if req.Email != "" {
		user.Email = &req.Email
	}

	if err := s.repo.CreateUser(ctx, &user); err != nil {
		return nil, err
	}

	tokens, err := s.generateTokenPair(ctx, user.ID, user.Role)
	if err != nil {
		return nil, err
	}

	return &domain.AuthResponse{
		User:   user,
		Tokens: *tokens,
	}, nil
}

func (s *authService) Login(ctx context.Context, req domain.LoginRequest) (*domain.AuthResponse, error) {
	req.Identifier = strings.TrimSpace(req.Identifier)
	if req.Identifier == "" || req.Password == "" {
		return nil, errors.New("identifier and password are required")
	}

	user, err := s.repo.GetUserByIdentifier(ctx, req.Identifier)
	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	if !user.IsActive {
		return nil, errors.New("account is disabled")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, errors.New("invalid credentials")
	}

	tokens, err := s.generateTokenPair(ctx, user.ID, user.Role)
	if err != nil {
		return nil, err
	}

	return &domain.AuthResponse{
		User:   *user,
		Tokens: *tokens,
	}, nil
}

func (s *authService) RefreshToken(ctx context.Context, refreshToken string) (*domain.TokenPair, error) {
	if refreshToken == "" {
		return nil, errors.New("refresh token is required")
	}

	userID, err := s.repo.GetUserIDByRefreshToken(ctx, refreshToken)
	if err != nil {
		return nil, errors.New("invalid or expired refresh token")
	}

	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil || !user.IsActive {
		return nil, errors.New("user not found or inactive")
	}

	// Delete used refresh token (Token rotation)
	_ = s.repo.DeleteRefreshToken(ctx, refreshToken)

	return s.generateTokenPair(ctx, user.ID, user.Role)
}

func (s *authService) Logout(ctx context.Context, refreshToken string) error {
	if refreshToken == "" {
		return errors.New("refresh token is required")
	}
	return s.repo.DeleteRefreshToken(ctx, refreshToken)
}

func (s *authService) GetProfile(ctx context.Context, userID string) (*domain.User, error) {
	return s.repo.GetUserByID(ctx, userID)
}

func (s *authService) ValidateAccessToken(tokenStr string) (*domain.JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &domain.JWTClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(s.cfg.JWTSecret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*domain.JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

func (s *authService) generateTokenPair(ctx context.Context, userID string, role domain.Role) (*domain.TokenPair, error) {
	// Access Token (JWT)
	accessExp := time.Now().Add(time.Duration(s.cfg.JWTAccessExpirationHours) * time.Hour)
	claims := domain.JWTClaims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(accessExp),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   userID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	accessToken, err := token.SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		return nil, fmt.Errorf("failed to sign access token: %w", err)
	}

	// Refresh Token (Secure random string)
	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}
	refreshToken := hex.EncodeToString(randomBytes)

	// Save Refresh Token to Redis
	refreshTTL := time.Duration(s.cfg.JWTRefreshExpirationDays) * 24 * time.Hour
	if err := s.repo.SaveRefreshToken(ctx, userID, refreshToken, refreshTTL); err != nil {
		return nil, fmt.Errorf("failed to save refresh token: %w", err)
	}

	return &domain.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int64(s.cfg.JWTAccessExpirationHours * 3600),
	}, nil
}
