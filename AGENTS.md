# 🤖 Backend Microservice Agent Guide

> **Core Go API Gateway & Business Logic for ONE-DURIAN Platform**  
> รายละเอียดและข้อตกลงสำหรับ AI Agent ในการพัฒนา Service ฝั่ง Backend

---

## 📌 สรุป Architecture สำคัญ
- **Language**: Go 1.24 (Standard Library `net/http` + Clean Modular Pattern)
- **Primary Keys**: UUIDv7 สร้างผ่าน `pkg/uid.NewV7()`
- **Database**: PostgreSQL 16 (Pool 25 max conns, 10 idle, Cold-start retry 5x)
- **Cache**: Redis 7 (Rate limit with TTL recovery guard)
- **Object Storage**: Garage S3 via `pkg/s3client` (MinIO Go SDK)
- **Docs**: OpenAPI 3.0 via `swaggo/swag` (`/docs` - Scalar, `/swagger` - Swagger UI)

---

## 🚀 ลำดับงานถัดไป (Active Task: Phase 2 Farm & Plot Management)
เมื่อเริ่มงาน Phase 2 ให้สร้างแพ็กเกจใหม่ที่ `backend/internal/farm/`:
1. `domain/farm.go`: Model สำหรับ `Farm`, `Plot`, `Tree`, Request/Response DTOs
2. `repository.go`: Database queries สำหรับตาราง `farms`, `plots`, `trees`
3. `service.go`: Business logic จัดการแปลงสวน
4. `handler.go`: HTTP endpoints พร้อม Swag documentation
5. Wiring ใน `cmd/api/main.go`
6. รัน `swag init -g cmd/api/main.go -o docs` และ `go build ./...`
