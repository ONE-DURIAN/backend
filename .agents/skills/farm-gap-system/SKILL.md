---
name: farm-gap-system
description: Domain knowledge, standards, and business logic for the Farm Community & GAP Digital Certificate Platform (มกษ. 9001). Use when designing schemas, implementing farm/plot management, chemical spray logbooks, and GAP audit traceability.
---

# 🌿 Skill: Farm Community & GAP Digital Certificate System (มกษ. 9001)

## 📌 มาตรฐาน มกษ. 9001 (Good Agricultural Practices for Food Crop)
มาตรฐานการปฏิบัติทางการเกษตรที่ดีสำหรับพืชอาหาร เพื่อให้ได้ผลผลิตที่มีคุณภาพ ปลอดภัยต่อผู้บริโภคและผู้ปฏิบัติงาน และไม่ทำลายสิ่งแวดล้อม

### 1. ข้อมูลแปลงสวน (Farm & Plot Data)
- **หน่วยวัดพื้นที่การเกษตรไทย**:
  - `1 ไร่ = 4 งาน = 400 ตารางวา = 1,600 ตารางเมตร`
  - `1 งาน = 100 ตารางวา = 400 ตารางเมตร`
  - `1 ตารางวา = 4 ตารางเมตร`
  - **สูตรคำนวณเป็นไร่ทศนิยม (Decimal Rai)**:
    $$\text{Decimal Rai} = \text{Rai} + \frac{\text{Ngan}}{4} + \frac{\text{Tarangwa}}{400}$$
- **สายพันธุ์ทุเรียนหลัก**:
  - หมอนทอง (Monthong)
  - ชะนี (Chanee)
  - ก้านยาว (Kan Yao)
  - พวงมณี (Puang Manee)
  - มูซานคิง (Musan King)
  - โอวฉี / หนามดำ (Black Thorn)

### 2. สมุดบันทึกการใช้สารเคมีและการพ่นยา (Spraying Logbook)
หัวใจสำคัญของการตรวจประเมิน GAP คือ **"บันทึกที่ตรวจสอบย้อนกลับได้ (Traceability)"**:
1. **สารเคมีต้องขึ้นทะเบียนถูกต้อง**: เลขทะเบียนวัตถุอันตรายทางการเกษตร (กปภ.)
2. **กลุ่มกลไกการออกฤทธิ์ (Mode of Action)**:
   - สารกำจัดแมลง: IRAC Group (เช่น กลุ่ม 1B, กลุ่ม 4A, กลุ่ม 28)
   - สารกำจัดเชื้อรา: FRAC Group (เช่น กลุ่ม 1, กลุ่ม 3, กลุ่ม 11)
   - ต้องมีการแนะนำการสลับกลุ่มกลไกเพื่อป้องกันแมลง/เชื้อราดื้อยา
3. **ระยะปลอดภัยก่อนเก็บเกี่ยว (Pre-Harvest Interval - PHI)**:
   - สารเคมีแต่ละชนิดมีค่า PHI เป็นจำนวนวัน (เช่น 7 วัน, 14 วัน, 21 วัน)
   - ระบบต้องคำนวณวันเก็บเกี่ยวปลอดภัย และเตือนหากมีการเก็บเกี่ยวก่อนพ้นระยะ PHI
4. **หลักฐานภาพถ่าย (Audit Evidence)**:
   - บันทึกการพ่นยาต้องสามารถแนบรูปใบเสร็จการซื้อสารเคมี และรูปหน้างานได้ (เก็บใน Garage S3)
