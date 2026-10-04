# 📋 สรุปผลการดำเนินงานและแผนงานโครงการ (Project Progress & Roadmap)
> **โครงการ**: Farm Community & GAP Digital Certificate Platform (ระบบชุมชนชาวสวนและตรวจสอบย้อนกลับมาตรฐาน GAP มกษ. 9001)  
> **วันที่บันทึก**: 4 ตุลาคม 2026  
> **สถานะภาพรวม**: ✅ Phase 1 (Core Infrastructure, Auth & CI/CD) เสร็จสมบูรณ์ 100%

---

## 🌟 1. สรุปผลงานที่ทำสำเร็จแล้วในวันนี้ (Completed Today)

### 🏗️ 1.1 โครงสร้างระบบและสถาปัตยกรรม (Infrastructure & Architecture)
- **สถาปัตยกรรมแบบ Self-hosted Zero Cloud Cost**: รันระบบทั้งหมดบนคลัสเตอร์ **Proxmox VE** ภายในบ้าน ประหยัดค่าบริการคลาวด์ 100%
  - **LXC 100 (Core Data Node: `192.168.1.202`)**:
    - **PostgreSQL 16** (`:5432`) - ฐานข้อมูลหลัก รองรับส่วนขยาย `pgcrypto`
    - **Redis 7** (`:6379`) - ระบบ Cache, Rate Limiting และ Token Revocation
    - **Garage S3** (`:3900`) - ระบบ Object Storage แบบกระจายศูนย์สำหรับเก็บรูปถ่ายแปลงและใบรับรอง
  - **LXC 101 (App Gateway: `192.168.1.203`)**:
    - **Go API Gateway** (`:8080`)
    - **Cloudflare Zero Trust Tunnel** - นำโดเมน `https://api.au-nongtota.com` ออกสู่อินเทอร์เน็ตโดยไม่ต้องเปิดพอร์ตเราเตอร์
    - **Watchtower** - ตัวช่วยตรวจสอบและอัปเดต Docker Image ให้อัตโนมัติ
- **วัดผลความเร็วระบบ (Ultra-Low Latency)**: 
  - PostgreSQL: **~0.44 ms**
  - Redis: **~0.27 ms**
  - Garage S3: **~0.94 ms**
  *(ทุกบริการตอบสนองต่ำกว่า 1 มิลลิวินาที)*

---

### 💻 1.2 การพัฒนา Backend Core Service (Go 1.24 & Clean Architecture)
- **จัดโครงสร้าง Clean Modular Architecture**:
  - `cmd/api` - Entrypoint หลัก, รวม Dependency Injection และ Graceful Shutdown
  - `config` - โหลดการตั้งค่าจาก `.env` อย่างปลอดภัย ไม่มีรหัสผ่าน Hardcoded ในโค้ด
  - `internal/domain` - Domain Entities, Request/Response DTOs, Role Management
  - `internal/auth` - Service, Repository, Handler, Middleware แยก Layer ชัดเจน
  - `pkg/` - โมดูลที่ใช้ซ้ำได้ เช่น `database`, `redisclient`, `s3client`, `uid`, `response`, `middleware`, `apidocs`
- **ระบบความปลอดภัยและการจัดการตัวระบุ (UUIDv7 & Security)**:
  - ใช้ **UUIDv7** ฝัง Timestamp ใน Primary Key ช่วยป้องกัน B-Tree Index Fragmentation บน PostgreSQL
  - ใช้ **Bcrypt** แฮชรหัสผ่านความปลอดภัยสูง
  - ซ่อน `PasswordHash` และ `DeletedAt` ไม่ให้รั่วไหลออกไปทาง JSON Response
  - ป้องกันการโจมตี Memory Exhaustion ด้วย `http.MaxBytesReader(1MB)`
- **ระบบสมาชิกและการยืนยันตัวตน (Authentication & Authorization)**:
  - `POST /api/v1/auth/register`: บังคับสิทธิ์เริ่มต้นเป็น `farmer` (ชาวสวน) เท่านั้น ป้องกัน Privilege Escalation
  - `POST /api/v1/auth/login`: คืนค่า Access Token (JWT) และ Refresh Token (สุ่มปลอดภัย 32-byte เก็บใน Redis)
  - `POST /api/v1/auth/refresh`: รองรับ Token Rotation
  - `POST /api/v1/auth/logout`: เพิกถอน Token ทันทีใน Redis
  - `GET /api/v1/auth/me`: ดูโปรไฟล์ผู้ใช้ปัจจุบัน (ต้องมี Bearer JWT)

---

### 🛡️ 1.3 การยกระดับความเสถียรและความปลอดภัย (Reliability & Security Hardening)
1. **Panic Recovery Middleware**: ดักจับ Runtime Panic ทุกจุด บันทึก Stack Trace และส่ง JSON 500 กลับไป ป้องกันเซิร์ฟเวอร์หลุดการเชื่อมต่อ
2. **Access Logger**: บันทึก Log การยิง Request แบบ Real-time แสดง Method, URL, Status Code, และ Latency
3. **Database Cold-Start Retry**: มี Retry Loop 5 ครั้ง (ครั้งละ 2 วินาที) ป้องกัน API แครชตอนบูตเครื่องพร้อมกับ PostgreSQL
4. **Soft Delete Partial Unique Index**: เบอร์โทรและอีเมลใช้ `CREATE UNIQUE INDEX ... WHERE deleted_at IS NULL` ทำให้ชาวสวนที่ขอลบบัญชีไปแล้ว สามารถกลับมาสมัครใหม่ด้วยเบอร์เดิมได้
5. **Redis Rate Limiting Deadlock Protection**: เพิ่มระบบตรวจเช็คและต่ออายุ TTL อัตโนมัติหาก `TTL < 0` ป้องกันไม่ให้ผู้ใช้ถูก Rate Limit บล็อกถาวร พร้อมตัดพอร์ตออกจาก Client IP อย่างถูกต้อง
6. **JWT Secret Enforcement**: บังคับให้ `JWT_SECRET` ต้องยาวอย่างน้อย 32 ตัวอักษร และหากไม่มีจะสร้าง Ephemeral Key อัตโนมัติ ป้องกันการ Sign ด้วย String ว่าง

---

### 📚 1.4 ระบบเอกสาร API อัตโนมัติ (Modern API Docs)
- ติดตั้ง **`swaggo/swag` v1.16.4** อ่าน Go Comments สร้าง OpenAPI 3.0 อัตโนมัติ
- รองรับ 2 รูปแบบเอกสาร:
  - **Modern Scalar Portal**: `https://api.au-nongtota.com/docs` (หน้าตาสวยงาม ทันสมัย ใช้งานง่าย)
  - **Official Swagger UI**: `https://api.au-nongtota.com/swagger` (สำหรับทดสอบยิง Request)
- จัดการเรื่อง Trailing Slash (`/docs/` -> `/docs`) และ Redirect เรียบร้อย

---

### 🚀 1.5 ระบบ CI/CD & Automated Deployment
- **GitHub Actions (`.github/workflows/deploy.yml`)**:
  - ใช้ Native Docker CLI แทน Third-party Actions เพื่อให้สอดคล้องกับข้อกำหนดความปลอดภัยของ GitHub Organization `ONE-DURIAN`
  - ทำการบิลด์โค้ด Go, สร้างเอกสาร Swag, บิลด์ Docker Image และส่งขึ้น **GitHub Container Registry (GHCR)**: `ghcr.io/one-durian/backend:latest`
- **Watchtower บน LXC 101**:
  - ตรวจจับ Image ใหม่บน GHCR ทุกๆ 60 วินาที และสั่ง Graceful Restart อัตโนมัติ
  - แก้ไขปัญหา Docker API Version Mismatch ด้วย `DOCKER_API_VERSION=1.44`

---

## 🎯 2. แผนงานที่ต้องทำต่อไป (Next Steps & Roadmap)

```text
[Phase 1: Core System & Auth] (✅ สำเร็จแล้ว)
       ↓
[Phase 2: Farm & Plot Management] (📌 ถัดไป)
       ↓
[Phase 3: GAP Spraying & Chemical Logbook]
       ↓
[Phase 4: S3 Media & File Upload]
       ↓
[Phase 5: Mobile App Frontend (React Native + NativeWind)]
```

### 📌 ก้าวถัดไป: Phase 2 — ระบบแปลงสวนและต้นทุเรียน (Farm & Plot Management)
1. **ออกแบบ Schema และ Migration ตารางแปลงสวน**:
   - ตาราง `farms`: ชื่อสวน, เจ้าของสวน (`user_id`), ที่ตั้ง (จังหวัด, อำเภอ, ตำบล), ขนาดพื้นที่รวม (ไร่-งาน-ตารางวา แปลงเป็น Decimal Rai)
   - ตาราง `plots`: แปลงย่อยภายในสวน, ขอบเขตแปลง (GPS Coordinates / GeoJSON Polygon), ชนิดดิน, แหล่งน้ำ
   - ตาราง `trees`: บันทึกจำนวนต้นทุเรียนแต่ละสายพันธุ์ (หมอนทอง, ชะนี, ก้านยาว, มูซานคิง, หนามดำ), อายุต้น, วันที่เริ่มปลูก
2. **สร้าง CRUD API Endpoints สำหรับจัดการสวน**:
   - `POST /api/v1/farms` — สร้างข้อมูลสวนใหม่
   - `GET /api/v1/farms` — ดูรายการสวนทั้งหมดของชาวสวนคนนั้น
   - `GET /api/v1/farms/:id` — ดูรายละเอียดสวนและแปลงย่อย
   - `PUT /api/v1/farms/:id` — แก้ไขข้อมูลสวน
   - `DELETE /api/v1/farms/:id` — ลบสวน (Soft Delete)
   - `POST /api/v1/farms/:id/plots` — เพิ่มแปลงย่อยในสวน

---

### 📋 Phase 3: สมุดบันทึกกิจกรรมและสารเคมีมาตรฐาน GAP (มกษ. 9001)
1. **ระบบฐานข้อมูลสารเคมีและปุ๋ย**:
   - รายชื่อสารป้องกันกำจัดศัตรูพืชที่ได้รับอนุญาตจากกรมวิชาการเกษตร
   - กลุ่มกลไกการออกฤทธิ์ (IRAC / FRAC) เพื่อแนะนำการสลับกลุ่มยา ป้องกันหนอน/เพลี้ยดื้อยา
2. **ระบบนับถอยหลังระยะปลอดภัยก่อนเก็บเกี่ยว (Pre-Harvest Interval - PHI)**:
   - เตือนอัตโนมัติหากยังไม่พ้นระยะปลอดภัยก่อนวันตัดทุเรียน เพื่อให้ผ่านเกณฑ์ GAP ตรวจสารตกค้าง 100%
3. **ระบบบันทึกการพ่นยา/ให้น้ำ/ใส่ปุ๋ย**:
   - `POST /api/v1/plots/:id/logs` — บันทึกการพ่นยาพร้อมแนบรูปถ่ายใบเสร็จ/ขวดสารเคมี

---

### 🖼️ Phase 4: ระบบอัปโหลดไฟล์รูปภาพขึ้น Garage S3 (Media Service)
1. **Presigned URL Flow**:
   - Mobile Client ขอ Presigned Upload URL จาก Go API
   - Mobile Client อัปโหลดรูปภาพใบเสร็จ/รูปแปลงตรงเข้า Garage S3 โดยตรง (ลดภาระ CPU และแบนด์วิดท์ของ API Gateway)
2. **การจัดหมวดหมู่ Bucket**:
   - `avatars/` - รูปโปรไฟล์ผู้ใช้
   - `farms/` - รูปแปลงสวนและโฉนด
   - `gap-logs/` - รูปหลักฐานการพ่นยาและฉลากสารเคมี
   - `certificates/` - เอกสารใบรับรองมาตรฐาน GAP ดิจิทัล

---

### 📱 Phase 5: พัฒนาแอปพลิเคชันมือถือ (React Native Mobile Frontend)
1. **โครงสร้างโปรเจกต์ Mobile App**:
   - **Framework**: React Native with Expo (TypeScript)
   - **Styling**: NativeWind v4 (Tailwind CSS for Mobile) เพื่อดีไซน์ UI ที่สวยงาม ทันสมัย และลื่นไหล
   - **State Management & Caching**: TanStack Query (React Query) สำหรับซิงค์ข้อมูลกับ Backend
   - **Secure Storage**: Expo SecureStore สำหรับเก็บ JWT Access Token อย่างปลอดภัย
2. **ฟีเจอร์แรกสำหรับชาวสวน**:
   - หน้าจอ Login / Register ที่ใช้งานง่าย มีปุ่มสลับเบอร์โทร/อีเมล
   - หน้าแดชบอร์ดสรุปแปลงสวนและพยากรณ์อากาศท้องถิ่น
   - แผนที่แสดงพิกัดแปลงทุเรียน

---

### 👥 ข้อแนะนำสำหรับการเตรียมตัวทำงานร่วมกับทีม (Team Collaboration)
- **ใช้กลยุทธ์ "Shared Remote Dev Node via Tailscale"**:
  - เมื่อมีสมาชิกใหม่เข้าทีม ให้เชื่อมต่อผ่าน **Tailscale Mesh VPN** ของคุณ
  - สมาชิกในทีมสามารถรันโค้ด Go ตัวเดียวเพียวๆ (`go run ./cmd/api`) โดยชี้ Database และ Redis ข้าม Tailscale ไปที่ LXC 100 ก้อน Dev ได้ทันที
  - **ข้อดี**: สมาชิกในทีมไม่ต้องเปิด Docker หลายตัวให้เครื่องช้า ประหยัดแรม และข้อมูลเทสอยู่ในที่เดียวกัน

---

*เอกสารนี้ถูกบันทึกไว้ในโปรเจกต์ที่ `PROJECT_PROGRESS.md` เพื่อให้ทีมงานสามารถเปิดอ่านและดำเนินงานต่อได้ทันทีครับ*
