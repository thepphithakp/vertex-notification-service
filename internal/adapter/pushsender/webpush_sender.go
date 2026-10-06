package pushsender

import (
	"context"
	"strings"

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
		// ⚠️ webpush-go เติม "mailto:" ให้เองถ้าค่าที่ส่งเข้าไปไม่ได้ขึ้นต้นด้วย
		// "https:" (ดู getVAPIDAuthorizationHeader ใน vapid.go) — w.subscriber
		// ของเรามี "mailto:" ติดมาอยู่แล้วเสมอ (บังคับด้วย config.Load) ถ้าส่งตรงๆ
		// จะได้ claim "sub" เป็น "mailto:mailto:..." ซึ่งผิดรูปแบบ VAPID
		//
		// เจอจริงตอนทดสอบกับ Apple Web Push (web.push.apple.com) ซึ่งตรวจ VAPID
		// claim เข้มกว่า push service เจ้าอื่น — ปฏิเสธด้วย 403 ไม่ใช่ error
		// ที่อ่านออกง่ายๆ ส่วน FCM/Mozilla อาจจะปล่อยผ่าน claim ที่ผิดแบบนี้เงียบๆ
		Subscriber:      vapidSubscriberArg(w.subscriber),
		VAPIDPublicKey:  w.publicKey,
		VAPIDPrivateKey: w.privateKey,
		TTL:             ttlSeconds,
	})
	if err != nil {
		return 0, err
	}
	defer func() { _ = resp.Body.Close() }()
	return resp.StatusCode, nil
}

// vapidSubscriberArg กัน "mailto:" ซ้อนกันสองชั้น — แยกออกมาเพื่อให้เทสได้
// โดยไม่ต้องยิง network จริง (ดูคอมเมนต์ยาวที่จุดเรียกใช้ด้านบน)
func vapidSubscriberArg(subscriber string) string {
	return strings.TrimPrefix(subscriber, "mailto:")
}
