package model

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/vertex/notification-service/internal/domain"
)

func TestNotificationLogRow_RoundTrip(t *testing.T) {
	readAt := time.Now().Truncate(time.Second)
	want := domain.NotificationLog{
		ID:        uuid.New(),
		UserID:    "u1",
		Title:     "มีข้อความใหม่",
		Body:      "สวัสดี",
		URL:       "/chat/1",
		CreatedAt: time.Now().Truncate(time.Second),
		ReadAt:    &readAt,
	}

	row := NotificationLogRowFromDomain(want)
	got := row.ToDomain()

	if got.ID != want.ID || got.Title != want.Title || got.Body != want.Body {
		t.Fatalf("round trip ไม่ตรง: %+v", got)
	}
	if got.ReadAt == nil || !got.ReadAt.Equal(readAt) {
		t.Fatalf("ReadAt ไม่ตรงหลัง round trip: %v", got.ReadAt)
	}
}

func TestNotificationLogRow_NilReadAtStaysNil(t *testing.T) {
	row := NotificationLogRowFromDomain(domain.NotificationLog{ReadAt: nil})
	if row.ReadAt != nil {
		t.Fatal("nil ต้องคง nil")
	}
	got := row.ToDomain()
	if got.ReadAt != nil {
		t.Fatal("nil ต้องคง nil หลัง round trip (unread)")
	}
}
