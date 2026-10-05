package pushsender

import (
	"context"

	webpush "github.com/SherClockHolmes/webpush-go"

	"github.com/vertex/notification-service/internal/domain"
)

// ttlSeconds คือเวลาที่ push service ควรเก็บ notification ไว้รอถ้าอุปกรณ์ออฟไลน์
// ก่อนจะทิ้งไปเลย — ไม่ตั้งนานไป เพราะการแจ้งเตือนของ vertex (เตือนจอดรถซ้อน,
// นัดหมายใกล้ถึง) หมดความหมายถ้าไปโผล่ช้าเกินไป
const ttlSeconds = 60 * 60 // 1 ชั่วโมง

// WebPushSender ส่งจริงผ่าน Web Push protocol (RFC 8030/8291) ด้วย VAPID
type WebPushSender struct {
	publicKey  string
	privateKey string
	subscriber string // "mailto:..." ตามข้อกำหนดของ VAPID
}

func NewWebPushSender(publicKey, privateKey, subscriber string) *WebPushSender {
	return &WebPushSender{publicKey: publicKey, privateKey: privateKey, subscriber: subscriber}
}

func (w *WebPushSender) Send(ctx context.Context, sub domain.PushSubscription, payload []byte) (int, error) {
	resp, err := webpush.SendNotificationWithContext(ctx, payload, &webpush.Subscription{
		Endpoint: sub.Endpoint,
		Keys: webpush.Keys{
			P256dh: sub.P256dh,
			Auth:   sub.Auth,
		},
	}, &webpush.Options{
		Subscriber:      w.subscriber,
		VAPIDPublicKey:  w.publicKey,
		VAPIDPrivateKey: w.privateKey,
		TTL:             ttlSeconds,
	})
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	return resp.StatusCode, nil
}
