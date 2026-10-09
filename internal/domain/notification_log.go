package domain

import (
	"time"

	"github.com/google/uuid"
)

// NotificationLog คือประวัติการแจ้งเตือนที่เคยส่งให้ user คนหนึ่ง — ใช้แสดงเป็น
// in-app notification feed (แบบกระดิ่งของ Facebook) แยกต่างหากจาก
// PushSubscription โดยสิ้นเชิง
//
// บันทึกทุกครั้งที่มีการเรียก SendToUser ไม่ว่า device จะได้รับ push จริง
// หรือไม่ (ไม่มี subscription เลยก็ยังบันทึก) เพราะ in-app feed ควรเห็นสิ่งที่
// "ระบบพยายามแจ้ง" ไม่ใช่แค่สิ่งที่ "push ไปถึงเครื่องจริง"
//
// 🔴 type นี้เป็น domain ล้วน ไม่มี gorm/json tag — GORM อยู่ที่
// adapter/repository/model.NotificationLogRow และ wire format ไปอยู่
// adapter/handler (notificationLogResponse) แทน
type NotificationLog struct {
	ID uuid.UUID

	UserID string
	Title  string
	Body   string
	URL    string

	CreatedAt time.Time

	// ReadAt เป็น NULL แปลว่ายังไม่ได้อ่าน — ใช้คำนวณ unread badge
	ReadAt *time.Time
}
