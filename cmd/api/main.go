package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"community-backend/config"
	"community-backend/internal/auth"
	"community-backend/pkg/apidocs"
	"community-backend/pkg/database"
	"community-backend/pkg/redisclient"
	"community-backend/pkg/response"
	"community-backend/pkg/s3client"
)

// @title Farm Community & GAP Digital Certificate API
// @version 0.2.0
// @description API Gateway สำหรับแพลตฟอร์มชุมชนชาวสวน และระบบตรวจสอบย้อนกลับมาตรฐาน GAP (มกษ. 9001)
// @description รองรับระบบ Authentication, แปลงสวน, และสมุดบันทึกการพ่นยา
// @host api.au-nongtota.com
// @BasePath /
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description กรอก "Bearer <JWT_TOKEN>" (มีเว้นวรรค 1 ช่อง)
func main() {
	cfg := config.LoadConfig()

	log.Printf("🚀 Starting Farm Community API Gateway...")
	log.Printf("📡 Configuration: Port=%s, DB=%s:%s, Redis=%s:%s, S3=%s (Bucket: %s)",
		cfg.AppPort, cfg.DBHost, cfg.DBPort, cfg.RedisHost, cfg.RedisPort, cfg.S3Endpoint, cfg.S3Bucket)

	// 1. Initialize PostgreSQL Connection
	db, err := database.ConnectPostgres(cfg)
	if err != nil {
		log.Printf("⚠️ PostgreSQL Connection Warning: %v (Continuing, check /healthz)", err)
	} else {
		defer db.Close()
		log.Printf("✅ PostgreSQL Connected successfully")

		// Run Auto Migration for Users table
		if err := db.AutoMigrate(); err != nil {
			log.Printf("⚠️ Auto-Migration Warning: %v", err)
		}
	}

	// 2. Initialize Redis Connection
	rdb, err := redisclient.ConnectRedis(cfg)
	if err != nil {
		log.Printf("⚠️ Redis Connection Warning: %v (Continuing, check /healthz)", err)
	} else {
		defer rdb.Close()
		log.Printf("✅ Redis Connected successfully")
	}

	// 3. Initialize Garage S3 Client
	s3, err := s3client.ConnectS3(cfg)
	if err != nil {
		log.Printf("⚠️ Garage S3 Connection Warning: %v (Continuing, check /healthz)", err)
	} else {
		log.Printf("✅ Garage S3 Connected (Bucket '%s' ready)", s3.Bucket)
	}

	// 4. Initialize Handlers & Services
	var authHandler *auth.Handler
	var authSvc auth.Service
	if db != nil && rdb != nil {
		authRepo := auth.NewRepository(db, rdb)
		authSvc = auth.NewService(authRepo, cfg)
		authHandler = auth.NewHandler(authSvc)
	}

	mux := http.NewServeMux()

	// Root Route
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		response.JSON(w, http.StatusOK, map[string]any{
			"app":         "Farm Community & GAP Digital Cert Platform",
			"status":      "running",
			"version":     "v0.2.0-modular",
			"time":        time.Now().Format(time.RFC3339),
			"healthcheck": "/healthz",
		})
	})

	// Liveness Probe (Instant response for Docker/Cloudflare ping without DB load)
	mux.HandleFunc("/livez", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"alive"}`))
	})

	// Readiness / Deep Healthcheck Route
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		handleHealthCheck(w, r, db, rdb, s3, cfg)
	})

	// API Documentation Routes (/docs, /docs/swagger, /docs/openapi.json)
	apidocs.RegisterRoutes(mux)

	// Auth Routes (Protected by Redis Rate Limiting)
	if authHandler != nil {
		registerLimiter := auth.RateLimitMiddleware(rdb, 5, 1*time.Minute, "register")
		loginLimiter := auth.RateLimitMiddleware(rdb, 10, 1*time.Minute, "login")

		mux.Handle("/api/v1/auth/register", registerLimiter(http.HandlerFunc(authHandler.Register)))
		mux.Handle("/api/v1/auth/login", loginLimiter(http.HandlerFunc(authHandler.Login)))
		mux.HandleFunc("/api/v1/auth/refresh", authHandler.RefreshToken)
		mux.HandleFunc("/api/v1/auth/logout", authHandler.Logout)

		// Protected Route: /api/v1/auth/me
		protectedMe := auth.AuthMiddleware(authSvc)(http.HandlerFunc(authHandler.GetMe))
		mux.Handle("/api/v1/auth/me", protectedMe)
	}

	server := &http.Server{
		Addr:         ":" + cfg.AppPort,
		Handler:      corsMiddleware(mux),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful Shutdown Channel
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("🌿 API Server listening on port %s", cfg.AppPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	<-stopChan
	log.Println("🛑 Shutting down server gracefully...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}
	log.Println("👋 Server stopped")
}

func handleHealthCheck(w http.ResponseWriter, r *http.Request, db *database.DB, rdb *redisclient.Client, s3 *s3client.S3Client, cfg config.Config) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	type ServiceStatus struct {
		Status    string  `json:"status"`
		LatencyMS float64 `json:"latency_ms,omitempty"`
		Message   string  `json:"message,omitempty"`
	}

	services := make(map[string]ServiceStatus)
	allHealthy := true

	// Postgres
	if db != nil {
		if lat, err := db.HealthCheck(ctx); err == nil {
			services["postgresql"] = ServiceStatus{
				Status:    "connected",
				LatencyMS: lat,
				Message:   fmt.Sprintf("Connected to database '%s'", cfg.DBName),
			}
		} else {
			allHealthy = false
			services["postgresql"] = ServiceStatus{Status: "error", Message: err.Error()}
		}
	} else {
		allHealthy = false
		services["postgresql"] = ServiceStatus{Status: "disconnected", Message: "database not initialized"}
	}

	// Redis
	if rdb != nil {
		if lat, err := rdb.HealthCheck(ctx); err == nil {
			services["redis"] = ServiceStatus{
				Status:    "connected",
				LatencyMS: lat,
				Message:   "Received 'PONG' from Redis",
			}
		} else {
			allHealthy = false
			services["redis"] = ServiceStatus{Status: "error", Message: err.Error()}
		}
	} else {
		allHealthy = false
		services["redis"] = ServiceStatus{Status: "disconnected", Message: "redis not initialized"}
	}

	// Garage S3
	if s3 != nil {
		if lat, exists, err := s3.HealthCheck(ctx); err == nil {
			services["garage_s3"] = ServiceStatus{
				Status:    "connected",
				LatencyMS: lat,
				Message:   fmt.Sprintf("Bucket '%s' exists: %v", s3.Bucket, exists),
			}
		} else {
			allHealthy = false
			services["garage_s3"] = ServiceStatus{Status: "error", Message: err.Error()}
		}
	} else {
		allHealthy = false
		services["garage_s3"] = ServiceStatus{Status: "disconnected", Message: "s3 not initialized"}
	}

	overall := "healthy"
	statusHTTP := http.StatusOK
	if !allHealthy {
		overall = "degraded"
		statusHTTP = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusHTTP)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"overall_status": overall,
		"timestamp":      time.Now().Format(time.RFC3339),
		"environment":    "production-local",
		"services":       services,
	})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
