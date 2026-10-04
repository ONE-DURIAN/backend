# 🏛️ Architecture & Coding Standards (ONE-DURIAN Backend)

## 1. Clean Modular Architecture Pattern
ทุกฟีเจอร์ใหม่ที่พัฒนาใน `backend/internal/` ต้องแบ่ง Layer ชัดเจน:
- **`domain/`**: กำหนด Entity structs, Enums, DTOs (`CreateRequest`, `Response`), JSON tags
- **`repository.go`**: มี Interface และ Struct ทำหน้าที่คุยกับ Database / Cache เท่านั้น (ใช้ Prepared Statements / Parameterized Query เสมอ)
- **`service.go`**: มี Interface และ Struct บรรจุ Business Logic ทั้งหมด (ไม่ผูกกับ HTTP Request/Response โดยตรง)
- **`handler.go`**: รับ HTTP Request, ตรวจสอบ Payload (`json.NewDecoder`), จำกัด Payload ด้วย `http.MaxBytesReader(1MB)`, เรียก Service, และตอบกลับผ่าน `pkg/response.JSON` หรือ `pkg/response.Error`
- **`middleware.go`**: จัดการเรื่อง Guard, RBAC, Rate Limiting

## 2. API Documentation Standards
ทุก Handler ใน Go ต้องเขียน Go Comment ในรูปแบบ Swag Annotation เสมอ เช่น:
```go
// CreateFarm godoc
// @Summary สร้างข้อมูลแปลงสวนใหม่ (Create Farm)
// @Description บันทึกข้อมูลแปลงสวนใหม่ของชาวสวน
// @Tags Farm Management
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body domain.CreateFarmRequest true "ข้อมูลแปลงสวน"
// @Success 201 {object} response.APIResponse{data=domain.Farm}
// @Failure 400 {object} response.APIResponse
// @Failure 401 {object} response.APIResponse
// @Router /api/v1/farms [post]
```
หลังจากเขียนเสร็จ ต้องรัน:
```bash
swag init -g cmd/api/main.go -o docs
```

## 3. Database Conventions
- ใช้ PostgreSQL 16
- ทุกตารางต้องมี:
  - `id UUID PRIMARY KEY DEFAULT gen_random_uuid()` (ใน Go ให้สร้างด้วย `uid.NewV7()`)
  - `created_at TIMESTAMPTZ DEFAULT NOW() NOT NULL`
  - `updated_at TIMESTAMPTZ DEFAULT NOW() NOT NULL`
  - `deleted_at TIMESTAMPTZ` (สำหรับ Soft Delete)
- ดัชนี Unique ต้องใช้ Partial Index เสมอ:
  ```sql
  CREATE UNIQUE INDEX IF NOT EXISTS idx_<table_column>_unique ON <table>(<column>) WHERE deleted_at IS NULL;
  ```

## 4. Network Environment Reference
- PostgreSQL: `192.168.1.202:5432` (LXC 100)
- Redis: `192.168.1.202:6379` (LXC 100)
- Garage S3: `http://192.168.1.202:3900` (LXC 100, Bucket: `community-media`)
- API Gateway: `192.168.1.203:8080` (LXC 101, Public: `https://api.au-nongtota.com`)
