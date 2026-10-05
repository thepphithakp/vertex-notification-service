package bootstrap

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vertex/notification-service/internal/config"
	"github.com/vertex/notification-service/pkg/middleware"
)

// TestRoutes_DoNotCrossAuthenticate ป้องกันไม่ให้ bug นี้กลับมาอีก:
//
// Fiber ผูก middleware ของ group ด้วย string-prefix match ไม่ใช่ตาม segment
// ของ path เคยลองตั้ง path ภายในเป็น "/api/v1/push-internal" ซึ่งขึ้นต้นด้วย
// ตัวอักษรเดียวกับ "/api/v1/push" (ของ user route ที่ต้องการ JWT) ทุกตัว ทำให้
// request ไป /send โดน middleware ของ JWT ครอบไปด้วย ตอบ 401 "ต้องมี
// Authorization: Bearer" แทนที่จะเช็ค X-Service-Token เหมือนที่ตั้งใจ — เจอจริง
// ตอนเทส local ด้วย curl
func TestRoutes_DoNotCrossAuthenticate(t *testing.T) {
	app, _ := NewApp(nil, config.Config{Service: config.ServiceConfig{Token: "test-service-token-at-least-32-chars"}},
		middleware.AuthConfig{PublicKeys: nil})

	t.Run("user route ปฏิเสธ request ที่ไม่มี JWT", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/push/subscribe", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("app.Test ล้มเหลว: %v", err)
		}
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("ต้องการ 401 ได้ %d", resp.StatusCode)
		}
	})

	t.Run("internal route ปฏิเสธ request ที่ไม่มี service token แต่ไม่ขอ JWT", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/internal/push/send", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("app.Test ล้มเหลว: %v", err)
		}
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("ต้องการ 401 ได้ %d", resp.StatusCode)
		}
	})

	t.Run("internal route ยอมรับ service token โดยไม่ต้องมี JWT เลย", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/internal/push/send", nil)
		req.Header.Set("X-Service-Token", "test-service-token-at-least-32-chars")
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("app.Test ล้มเหลว: %v", err)
		}
		// ⚠️ นี่คือจุดสำคัญของเทสนี้ — ต้องไม่ใช่ 401 จาก NewAuth
		// (ถ้า bug กลับมา จะติด "ต้องมี Authorization: Bearer" ตรงนี้แทน)
		if resp.StatusCode == http.StatusUnauthorized {
			t.Fatalf("internal route ต้องไม่โดน JWT middleware ครอบ ได้ 401 ทั้งที่ส่ง service token แล้ว")
		}
	})
}
