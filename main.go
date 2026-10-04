package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	_ "github.com/lib/pq"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/redis/go-redis/v9"
)

type Config struct {
	AppPort       string
	DBHost        string
	DBPort        string
	DBUser        string
	DBPassword    string
	DBName        string
	RedisHost     string
	RedisPort     string
	RedisPassword string
	S3Endpoint    string
	S3Region      string
	S3AccessKey   string
	S3SecretKey   string
	S3Bucket      string
}

func loadConfig() Config {
	port := os.Getenv("APP_PORT")
	if port == "" {
		port = os.Getenv("PORT")
	}
	if port == "" {
		port = "8080"
	}

	return Config{
		AppPort:       port,
		DBHost:        getEnv("DB_HOST", "192.168.1.202"),
		DBPort:        getEnv("DB_PORT", "5432"),
		DBUser:        getEnv("DB_USER", "admin"),
		DBPassword:    getEnv("DB_PASSWORD", "St@rtuPPr0j3ct!01"),
		DBName:        getEnv("DB_NAME", "farm_community"),
		RedisHost:     getEnv("REDIS_HOST", "192.168.1.202"),
		RedisPort:     getEnv("REDIS_PORT", "6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", "St@rtuPPr0j3ct!01"),
		S3Endpoint:    getEnv("S3_ENDPOINT", "http://192.168.1.202:3900"),
		S3Region:      getEnv("S3_REGION", "garage"),
		S3AccessKey:   getEnv("S3_ACCESS_KEY", "GK2b35f24d335606e07c78f500"),
		S3SecretKey:   getEnv("S3_SECRET_KEY", "7bceac0f079e839f0c3ef22ef4f9cff442b581fdc7c4996d14f5ebf9cf1c20ea"),
		S3Bucket:      getEnv("S3_BUCKET", "community-media"),
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

type ServiceStatus struct {
	Status    string  `json:"status"`
	LatencyMS float64 `json:"latency_ms,omitempty"`
	Message   string  `json:"message,omitempty"`
}

type HealthResponse struct {
	OverallStatus string                   `json:"overall_status"`
	Timestamp     string                   `json:"timestamp"`
	Environment   string                   `json:"environment"`
	Services      map[string]ServiceStatus `json:"services"`
}

var cfg Config

func main() {
	cfg = loadConfig()

	mux := http.NewServeMux()

	// Root Route
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"app":         "Farm Community & GAP Digital Cert API",
			"status":      "running",
			"version":     "v0.1.0-alpha",
			"time":        time.Now().Format(time.RFC3339),
			"healthcheck": "/healthz",
		})
	})

	// Healthcheck & Connectivity Ping Route
	mux.HandleFunc("/healthz", handleHealthCheck)

	serverAddr := ":" + cfg.AppPort
	log.Printf("🚀 Farm Community API Gateway starting on port %s ...", cfg.AppPort)
	log.Printf("📡 Configured Data Node: DB=%s:%s, Redis=%s:%s, S3=%s (Bucket: %s)",
		cfg.DBHost, cfg.DBPort, cfg.RedisHost, cfg.RedisPort, cfg.S3Endpoint, cfg.S3Bucket)

	if err := http.ListenAndServe(serverAddr, mux); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

func handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	services := make(map[string]ServiceStatus)
	allHealthy := true

	// 1. Check PostgreSQL
	dbStatus, ok := checkPostgres()
	services["postgresql"] = dbStatus
	if !ok {
		allHealthy = false
	}

	// 2. Check Redis
	redisStatus, ok := checkRedis()
	services["redis"] = redisStatus
	if !ok {
		allHealthy = false
	}

	// 3. Check Garage S3
	s3Status, ok := checkGarageS3()
	services["garage_s3"] = s3Status
	if !ok {
		allHealthy = false
	}

	overall := "healthy"
	httpStatus := http.StatusOK
	if !allHealthy {
		overall = "degraded"
		httpStatus = http.StatusServiceUnavailable
	}

	resp := HealthResponse{
		OverallStatus: overall,
		Timestamp:     time.Now().Format(time.RFC3339),
		Environment:   "production-local",
		Services:      services,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)
	json.NewEncoder(w).Encode(resp)
}

func checkPostgres() (ServiceStatus, bool) {
	start := time.Now()
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable connect_timeout=3",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return ServiceStatus{Status: "error", Message: err.Error()}, false
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return ServiceStatus{Status: "error", Message: fmt.Sprintf("Ping failed: %v", err)}, false
	}

	latency := float64(time.Since(start).Microseconds()) / 1000.0
	return ServiceStatus{
		Status:    "connected",
		LatencyMS: latency,
		Message:   fmt.Sprintf("Connected to database '%s'", cfg.DBName),
	}, true
}

func checkRedis() (ServiceStatus, bool) {
	start := time.Now()
	rdb := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%s", cfg.RedisHost, cfg.RedisPort),
		Password:     cfg.RedisPassword,
		DB:           0,
		DialTimeout:  3 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})
	defer rdb.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	pong, err := rdb.Ping(ctx).Result()
	if err != nil {
		return ServiceStatus{Status: "error", Message: fmt.Sprintf("Ping failed: %v", err)}, false
	}

	latency := float64(time.Since(start).Microseconds()) / 1000.0
	return ServiceStatus{
		Status:    "connected",
		LatencyMS: latency,
		Message:   fmt.Sprintf("Received '%s' from Redis", pong),
	}, true
}

func checkGarageS3() (ServiceStatus, bool) {
	start := time.Now()

	// Parse S3 endpoint
	endpoint := cfg.S3Endpoint
	useSSL := false
	if strings.HasPrefix(endpoint, "https://") {
		useSSL = true
		endpoint = strings.TrimPrefix(endpoint, "https://")
	} else if strings.HasPrefix(endpoint, "http://") {
		endpoint = strings.TrimPrefix(endpoint, "http://")
	}

	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.S3AccessKey, cfg.S3SecretKey, ""),
		Secure: useSSL,
		Region: cfg.S3Region,
	})
	if err != nil {
		return ServiceStatus{Status: "error", Message: fmt.Sprintf("Client init failed: %v", err)}, false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Check if bucket exists
	exists, err := minioClient.BucketExists(ctx, cfg.S3Bucket)
	if err != nil {
		return ServiceStatus{Status: "error", Message: fmt.Sprintf("S3 Check failed: %v", err)}, false
	}

	latency := float64(time.Since(start).Microseconds()) / 1000.0
	msg := fmt.Sprintf("Bucket '%s' exists: %v", cfg.S3Bucket, exists)
	return ServiceStatus{
		Status:    "connected",
		LatencyMS: latency,
		Message:   msg,
	}, true
}
