package auth

import (
	"encoding/json"
	"net/http"

	"community-backend/internal/domain"
	"community-backend/pkg/response"
)

type Handler struct {
	svc Service
}

func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

// Register godoc
// @Summary สมัครสมาชิกชาวสวน (Farmer Register)
// @Description สมัครสมาชิกใหม่ กำหนดสิทธิ์เริ่มต้นเป็น farmer (จำกัด 5 ครั้ง/นาที ต่อ 1 IP)
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body domain.RegisterRequest true "ข้อมูลการสมัครสมาชิก"
// @Success 201 {object} response.APIResponse{data=domain.AuthResponse}
// @Failure 400 {object} response.APIResponse
// @Failure 429 {object} response.APIResponse
// @Router /api/v1/auth/register [post]
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Protect against OOM attacks by limiting request body to 1MB
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var req domain.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	res, err := h.svc.Register(r.Context(), req)
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	response.Success(w, http.StatusCreated, "User registered successfully", res)
}

// Login godoc
// @Summary เข้าสู่ระบบ (Login)
// @Description เข้าสู่ระบบด้วยเบอร์โทรศัพท์หรืออีเมล (จำกัด 10 ครั้ง/นาที ต่อ 1 IP)
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body domain.LoginRequest true "ข้อมูลเข้าระบบ"
// @Success 200 {object} response.APIResponse{data=domain.AuthResponse}
// @Failure 401 {object} response.APIResponse
// @Failure 429 {object} response.APIResponse
// @Router /api/v1/auth/login [post]
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Protect against OOM attacks by limiting request body to 1MB
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var req domain.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	res, err := h.svc.Login(r.Context(), req)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, err.Error())
		return
	}

	response.Success(w, http.StatusOK, "Login successful", res)
}

// RefreshToken godoc
// @Summary ขอ Access Token ใหม่ (Refresh Token)
// @Description ส่ง Refresh Token เพื่อขอรับ Access Token ชุดใหม่
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body domain.RefreshTokenRequest true "Refresh Token"
// @Success 200 {object} response.APIResponse{data=domain.TokenPair}
// @Failure 401 {object} response.APIResponse
// @Router /api/v1/auth/refresh [post]
func (h *Handler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Protect against OOM attacks by limiting request body to 1MB
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var req domain.RefreshTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	tokens, err := h.svc.RefreshToken(r.Context(), req.RefreshToken)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, err.Error())
		return
	}

	response.Success(w, http.StatusOK, "Token refreshed successfully", tokens)
}

// Logout godoc
// @Summary ออกจากระบบ (Logout)
// @Description ลบ Refresh Token ออกจาก Redis
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body domain.LogoutRequest true "Refresh Token"
// @Success 200 {object} response.APIResponse
// @Failure 400 {object} response.APIResponse
// @Router /api/v1/auth/logout [post]
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Protect against OOM attacks by limiting request body to 1MB
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var req domain.LogoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := h.svc.Logout(r.Context(), req.RefreshToken); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	response.Success(w, http.StatusOK, "Logged out successfully", nil)
}

// GetMe godoc
// @Summary ดูข้อมูลโปรไฟล์ตัวเอง (Get Profile)
// @Description ดึงข้อมูลโปรไฟล์ผู้ใช้ปัจจุบันจาก JWT Access Token
// @Tags Authentication
// @Security BearerAuth
// @Produce json
// @Success 200 {object} response.APIResponse{data=domain.User}
// @Failure 401 {object} response.APIResponse
// @Router /api/v1/auth/me [get]
func (h *Handler) GetMe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	userID := GetUserID(r.Context())
	if userID == "" {
		response.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	user, err := h.svc.GetProfile(r.Context(), userID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "User not found")
		return
	}

	response.Success(w, http.StatusOK, "Profile retrieved", user)
}
