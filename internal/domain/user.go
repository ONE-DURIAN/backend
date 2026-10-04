package domain

import (
	"github.com/golang-jwt/jwt/v5"
	"time"
)

type Role string

const (
	RoleFarmer  Role = "farmer"
	RoleAuditor Role = "auditor"
	RoleAdmin   Role = "admin"
)

type User struct {
	ID           string    `json:"id"`
	PhoneNumber  *string   `json:"phone_number,omitempty"`
	Email        *string   `json:"email,omitempty"`
	PasswordHash string    `json:"-"`
	FullName     string    `json:"full_name"`
	AvatarURL    *string   `json:"avatar_url,omitempty"`
	Role         Role      `json:"role"`
	NationalID   *string   `json:"national_id,omitempty"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	DeletedAt    *time.Time `json:"-"`
}

type RegisterRequest struct {
	PhoneNumber string `json:"phone_number"`
	Email       string `json:"email"`
	Password    string `json:"password"`
	FullName    string `json:"full_name"`
	Role        Role   `json:"role,omitempty"` // Default to farmer if empty
}

type LoginRequest struct {
	Identifier string `json:"identifier"` // Phone number or email
	Password   string `json:"password"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type LogoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"` // Seconds
}

type AuthResponse struct {
	User   User      `json:"user"`
	Tokens TokenPair `json:"tokens"`
}

type JWTClaims struct {
	UserID string `json:"user_id"`
	Role   Role   `json:"role"`
	jwt.RegisteredClaims
}
