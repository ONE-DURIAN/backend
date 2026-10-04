package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	AppPort                  string
	DBHost                   string
	DBPort                   string
	DBUser                   string
	DBPassword               string
	DBName                   string
	RedisHost                string
	RedisPort                string
	RedisPassword            string
	S3Endpoint               string
	S3Region                 string
	S3AccessKey              string
	S3SecretKey              string
	S3Bucket                 string
	JWTSecret                string
	JWTAccessExpirationHours int
	JWTRefreshExpirationDays int
}

func LoadConfig() Config {
	// โหลดไฟล์ .env อัตโนมัติถ้ามี (สำหรับรัน local เครื่อง dev)
	// ถ้าไม่มีไฟล์ .env (เช่น รันใน Docker) ก็จะอ่านจาก Environment Variables ของระบบตามปกติ
	if err := godotenv.Load(); err != nil {
		// พยายามหาในไดเรกทอรีด้านบนเผื่อรันจาก cmd/api
		_ = godotenv.Load("../../.env")
	}

	port := getEnv("APP_PORT", getEnv("PORT", "8080"))

	accessHours, _ := strconv.Atoi(getEnv("JWT_ACCESS_EXPIRATION_HOURS", "2"))
	if accessHours <= 0 {
		accessHours = 2
	}

	refreshDays, _ := strconv.Atoi(getEnv("JWT_REFRESH_EXPIRATION_DAYS", "30"))
	if refreshDays <= 0 {
		refreshDays = 30
	}

	cfg := Config{
		AppPort:                  port,
		DBHost:                   os.Getenv("DB_HOST"),
		DBPort:                   getEnv("DB_PORT", "5432"),
		DBUser:                   os.Getenv("DB_USER"),
		DBPassword:               os.Getenv("DB_PASSWORD"),
		DBName:                   os.Getenv("DB_NAME"),
		RedisHost:                os.Getenv("REDIS_HOST"),
		RedisPort:                getEnv("REDIS_PORT", "6379"),
		RedisPassword:            os.Getenv("REDIS_PASSWORD"),
		S3Endpoint:               os.Getenv("S3_ENDPOINT"),
		S3Region:                 getEnv("S3_REGION", "garage"),
		S3AccessKey:              os.Getenv("S3_ACCESS_KEY"),
		S3SecretKey:              os.Getenv("S3_SECRET_KEY"),
		S3Bucket:                 os.Getenv("S3_BUCKET"),
		JWTSecret:                os.Getenv("JWT_SECRET"),
		JWTAccessExpirationHours: accessHours,
		JWTRefreshExpirationDays: refreshDays,
	}

	// แจ้งเตือนถ้าขาด Environment Variables ที่จำเป็น
	if cfg.DBHost == "" || cfg.DBName == "" {
		log.Println("⚠️ Warning: DB_HOST or DB_NAME is not set in environment")
	}
	if cfg.JWTSecret == "" {
		log.Println("⚠️ Warning: JWT_SECRET is not set in environment")
	}

	return cfg
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
