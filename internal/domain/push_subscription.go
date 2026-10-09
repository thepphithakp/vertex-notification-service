package domain

import (
	"time"

	"github.com/google/uuid"
)

// PushSubscription คือ endpoint ของ browser หนึ่งเครื่องที่ผู้ใช้คนหนึ่งเปิดรับ push ไว้
//
// ผู้ใช้คนเดียวเปิดได้หลายเครื่อง (มือถือ + browser อื่น) จึงเก็บเป็นหลายแถวต่อ UserID
// unique ที่ (user_id, endpoint) — endpoint มาจาก browser's push service เอง
// รับประกันว่าไม่ซ้ำกันข้ามเครื่อง อยู่แล้ว
//
// 🔴 type นี้เป็น domain ล้วน ไม่มี gorm tag — GORM อยู่ที่
// adapter/repository/model.PushSubscriptionRow แทน (ไม่มี json tag เพราะ
// ไม่มีใครส่งค่านี้กลับไปให้ client เลย handler ไม่เคย serialize
// PushSubscription ตรงๆ)
type PushSubscription struct {
	ID uuid.UUID

	UserID   string
	Endpoint string

	// P256dh / Auth คือ public key และ auth secret ที่ browser สร้างไว้
	// ใช้เข้ารหัส payload ตาม RFC 8291 ก่อนส่งผ่าน push service
	P256dh string
	Auth   string

	// UserAgent ช่วย debug ว่า subscription นี้มาจากอุปกรณ์/เบราว์เซอร์ไหน
	UserAgent string

	CreatedAt  time.Time
	LastSeenAt time.Time
}
