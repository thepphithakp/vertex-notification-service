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
type PushSubscription struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`

	UserID   string `gorm:"not null" json:"userId"`
	Endpoint string `gorm:"not null" json:"endpoint"`

	// P256dh / Auth คือ public key และ auth secret ที่ browser สร้างไว้
	// ใช้เข้ารหัส payload ตาม RFC 8291 ก่อนส่งผ่าน push service
	P256dh string `gorm:"not null" json:"p256dh"`
	Auth   string `gorm:"not null" json:"auth"`

	// UserAgent ช่วย debug ว่า subscription นี้มาจากอุปกรณ์/เบราว์เซอร์ไหน
	UserAgent string `json:"userAgent"`

	CreatedAt  time.Time `gorm:"not null;default:now()" json:"createdAt"`
	LastSeenAt time.Time `gorm:"not null;default:now()" json:"lastSeenAt"`
}

func (PushSubscription) TableName() string { return "push_subscriptions" }
