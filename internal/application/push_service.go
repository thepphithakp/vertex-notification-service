package application

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/vertex/notification-service/internal/domain"
	"github.com/vertex/notification-service/internal/port"
)

// ขอบเขตค่าที่ยอมรับ — กันไม่ให้ผู้เรียกเขียนขยะลงตาราง subscription
const (
	maxFieldLength = 4096 // endpoint/คีย์เข้ารหัสยาวกว่า field ทั่วไปมาก
	maxUALength    = 512
)

type PushService struct {
	repo   port.SubscriptionRepository
	sender port.Sender
	now    func() time.Time
}

func NewPushService(repo port.SubscriptionRepository, sender port.Sender) *PushService {
	return &PushService{repo: repo, sender: sender, now: time.Now}
}

func (s *PushService) Subscribe(ctx context.Context, userID string, in port.SubscribeInput) error {
	endpoint := strings.TrimSpace(in.Endpoint)
	p256dh := strings.TrimSpace(in.P256dh)
	auth := strings.TrimSpace(in.Auth)

	if userID == "" {
		return &ValidationError{Field: "userId", Reason: "ต้องไม่ว่าง"}
	}
	if !strings.HasPrefix(endpoint, "https://") {
		return &ValidationError{Field: "endpoint", Reason: "ต้องเป็น URL แบบ https"}
	}
	if p256dh == "" || auth == "" {
		return &ValidationError{Field: "keys", Reason: "ต้องมีทั้ง p256dh และ auth"}
	}
	for name, v := range map[string]string{"endpoint": endpoint, "p256dh": p256dh, "auth": auth} {
		if len(v) > maxFieldLength {
			return &ValidationError{Field: name, Reason: fmt.Sprintf("ยาวเกิน %d ตัวอักษร", maxFieldLength)}
		}
	}

	ua := strings.TrimSpace(in.UserAgent)
	if len(ua) > maxUALength {
		ua = ua[:maxUALength]
	}

	return s.repo.Upsert(ctx, &domain.PushSubscription{
		ID:         uuid.New(),
		UserID:     userID,
		Endpoint:   endpoint,
		P256dh:     p256dh,
		Auth:       auth,
		UserAgent:  ua,
		LastSeenAt: s.now(),
	})
}

func (s *PushService) Unsubscribe(ctx context.Context, userID, endpoint string) error {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return &ValidationError{Field: "endpoint", Reason: "ต้องไม่ว่าง"}
	}
	return s.repo.Delete(ctx, userID, endpoint)
}

// gonePushStatuses คือ status ที่ push service ใช้บอกว่า endpoint นี้ตายแล้ว
// ตาม RFC 8030 — ไม่มีทางส่งสำเร็จอีกแล้ว ต้องลบทิ้งไม่ใช่ retry
var gonePushStatuses = map[int]bool{
	http.StatusNotFound: true,
	http.StatusGone:     true,
}

// SendToUser ส่ง push ไปทุกอุปกรณ์ที่ user คนนี้เคยเปิดรับไว้
//
// ส่งทีละอุปกรณ์แยกจากกัน — อุปกรณ์หนึ่งตายไม่ควรทำให้อุปกรณ์อื่นไม่ได้รับ
// endpoint ที่ตายแล้ว (404/410) ถูกลบทิ้งทันทีแทนที่จะปล่อยค้างไว้ส่งซ้ำไม่จบ
func (s *PushService) SendToUser(ctx context.Context, userID string, msg port.Message) (port.SendResult, error) {
	var result port.SendResult

	if strings.TrimSpace(msg.Title) == "" || strings.TrimSpace(msg.Body) == "" {
		return result, &ValidationError{Field: "title/body", Reason: "ต้องไม่ว่าง"}
	}

	subs, err := s.repo.ListByUser(ctx, userID)
	if err != nil {
		return result, err
	}

	payload, err := json.Marshal(msg)
	if err != nil {
		return result, fmt.Errorf("แปลง payload เป็น JSON ไม่สำเร็จ: %w", err)
	}

	for _, sub := range subs {
		status, err := s.sender.Send(ctx, sub, payload)
		switch {
		case err != nil:
			slog.ErrorContext(ctx, "ส่ง push ไม่สำเร็จ",
				"subscription_id", sub.ID, "user_id", userID, "error", err)
			result.Failed++
		case gonePushStatuses[status]:
			if delErr := s.repo.DeleteByID(ctx, sub.ID); delErr != nil {
				slog.ErrorContext(ctx, "ลบ subscription ที่ตายแล้วไม่สำเร็จ",
					"subscription_id", sub.ID, "error", delErr)
			}
			result.Pruned++
		case status >= 200 && status < 300:
			result.Sent++
		default:
			slog.WarnContext(ctx, "push service ตอบ status ที่ไม่คาดคิด",
				"subscription_id", sub.ID, "status", status)
			result.Failed++
		}
	}

	return result, nil
}

// ValidationError บอกว่าผู้เรียกส่งอะไรมาผิด เพื่อให้ handler ตอบ 400 ได้
type ValidationError struct {
	Field  string
	Reason string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s %s", e.Field, e.Reason)
}
