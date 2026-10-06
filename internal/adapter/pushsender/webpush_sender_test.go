package pushsender

import "testing"

// webpush-go เติม "mailto:" ให้เองถ้าค่าที่ส่งเข้าไปไม่ได้ขึ้นต้นด้วย "https:"
// (ดู getVAPIDAuthorizationHeader ใน vendor ของ SherClockHolmes/webpush-go)
// VAPID_SUBSCRIBER ของเรามี "mailto:" ติดมาอยู่แล้วเสมอ (บังคับด้วย config.Load)
// ถ้าส่งตรงๆ ไม่ผ่านฟังก์ชันนี้ก่อน จะได้ claim "sub" เป็น "mailto:mailto:..."
// ซึ่ง Apple Web Push (web.push.apple.com) ปฏิเสธด้วย 403 — เจอจริงตอนทดสอบ
func TestVapidSubscriberArg(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"mailto:owner@example.com", "owner@example.com"},
		{"https://example.com/contact", "https://example.com/contact"},
	}
	for _, c := range cases {
		if got := vapidSubscriberArg(c.in); got != c.want {
			t.Errorf("vapidSubscriberArg(%q) = %q, ต้องการ %q", c.in, got, c.want)
		}
	}
}
