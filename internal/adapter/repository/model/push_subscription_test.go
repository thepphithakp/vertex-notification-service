package model

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/vertex/notification-service/internal/domain"
)

func TestPushSubscriptionRow_RoundTrip(t *testing.T) {
	want := domain.PushSubscription{
		ID:         uuid.New(),
		UserID:     "u1",
		Endpoint:   "https://fcm.googleapis.com/x",
		P256dh:     "key",
		Auth:       "secret",
		UserAgent:  "Mozilla/5.0",
		CreatedAt:  time.Now().Truncate(time.Second),
		LastSeenAt: time.Now().Truncate(time.Second),
	}

	row := PushSubscriptionRowFromDomain(want)
	got := row.ToDomain()

	if got.ID != want.ID || got.UserID != want.UserID || got.Endpoint != want.Endpoint {
		t.Fatalf("round trip ไม่ตรง: %+v", got)
	}
	if got.P256dh != want.P256dh || got.Auth != want.Auth {
		t.Fatalf("คีย์เข้ารหัสไม่ตรงหลัง round trip: %+v", got)
	}
}
