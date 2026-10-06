package port

import (
	"context"

	"github.com/google/uuid"

	"github.com/vertex/notification-service/internal/domain"
)

type SubscriptionRepository interface {
	// Upsert บันทึก subscription แบบ idempotent — อุปกรณ์เดิมเรียกซ้ำ
	// (เช่น service worker ถูกปลุกใหม่) จะอัปเดตคีย์ล่าสุดแทนที่จะสร้างแถวซ้ำ
	Upsert(ctx context.Context, sub *domain.PushSubscription) error

	// Delete ลบ subscription ของ user รายนี้ที่ endpoint นี้ — ใช้ตอนผู้ใช้กดปิดเอง
	Delete(ctx context.Context, userID, endpoint string) error

	// DeleteByID ใช้ตอน push service ตอบกลับว่า endpoint นี้ตายแล้ว (404/410)
	DeleteByID(ctx context.Context, id uuid.UUID) error

	ListByUser(ctx context.Context, userID string) ([]domain.PushSubscription, error)
}

// NotificationLogRepository เก็บประวัติการแจ้งเตือนสำหรับ in-app feed
// (แยกจาก SubscriptionRepository เพราะเป็นคนละ concern — ตัวนี้คือ "ประวัติ
// สิ่งที่เคยแจ้ง" ไม่ใช่ "อุปกรณ์ที่รับ push ได้")
type NotificationLogRepository interface {
	Create(ctx context.Context, log *domain.NotificationLog) error
	ListByUser(ctx context.Context, userID string, limit int) ([]domain.NotificationLog, error)
	UnreadCount(ctx context.Context, userID string) (int64, error)
	MarkAllRead(ctx context.Context, userID string) error
}

// Sender ส่ง payload ไปยัง endpoint จริง — แยก interface ออกจาก webpush-go
// เพื่อให้ mock ตอน unit test ได้โดยไม่ต้องยิง network จริง
type Sender interface {
	// Send คืน HTTP status code ที่ push service ตอบกลับมา
	// (เช่น 201 = ส่งสำเร็จ, 404/410 = endpoint หมดอายุแล้ว ควรลบทิ้ง)
	Send(ctx context.Context, sub domain.PushSubscription, payload []byte) (statusCode int, err error)
}

// Message คือเนื้อหาที่จะโผล่เป็น notification บนเครื่องผู้ใช้
type Message struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	Tag   string `json:"tag,omitempty"`
	URL   string `json:"url,omitempty"`
}

// SendResult สรุปผลการส่งให้ผู้เรียก (service อื่น) รู้ว่าเกิดอะไรขึ้นบ้าง
type SendResult struct {
	Sent   int `json:"sent"`
	Failed int `json:"failed"`
	Pruned int `json:"pruned"`
}

type SubscribeInput struct {
	Endpoint  string
	P256dh    string
	Auth      string
	UserAgent string
}

type PushUseCase interface {
	Subscribe(ctx context.Context, userID string, in SubscribeInput) error
	Unsubscribe(ctx context.Context, userID, endpoint string) error
	SendToUser(ctx context.Context, userID string, msg Message) (SendResult, error)

	ListNotifications(ctx context.Context, userID string) (NotificationFeed, error)
	MarkAllRead(ctx context.Context, userID string) error
}

const NotificationFeedLimit = 50

// NotificationFeed คือสิ่งที่หน้า in-app notification feed ต้องใช้แสดงผล
type NotificationFeed struct {
	Items       []domain.NotificationLog `json:"items"`
	UnreadCount int64                    `json:"unreadCount"`
}
