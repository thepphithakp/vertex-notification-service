// Package model เก็บ struct ที่ผูกกับ GORM/schema โดยตรง — ไม่มีใครนอก
// adapter/repository เห็น struct พวกนี้เลย แปลงเป็น domain type ที่ชายแดน
// ด้วย ToDomain()/FromDomain() ตามแบบของ vertex-auth-service
package model

import (
	"time"

	"github.com/google/uuid"

	"github.com/vertex/notification-service/internal/domain"
)

type PushSubscriptionRow struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey"`

	UserID   string `gorm:"not null"`
	Endpoint string `gorm:"not null"`

	P256dh string `gorm:"not null"`
	Auth   string `gorm:"not null"`

	UserAgent string

	CreatedAt  time.Time `gorm:"not null;default:now()"`
	LastSeenAt time.Time `gorm:"not null;default:now()"`
}

func (PushSubscriptionRow) TableName() string { return "push_subscriptions" }

func (r PushSubscriptionRow) ToDomain() domain.PushSubscription {
	return domain.PushSubscription{
		ID:         r.ID,
		UserID:     r.UserID,
		Endpoint:   r.Endpoint,
		P256dh:     r.P256dh,
		Auth:       r.Auth,
		UserAgent:  r.UserAgent,
		CreatedAt:  r.CreatedAt,
		LastSeenAt: r.LastSeenAt,
	}
}

func PushSubscriptionRowFromDomain(s domain.PushSubscription) PushSubscriptionRow {
	return PushSubscriptionRow{
		ID:         s.ID,
		UserID:     s.UserID,
		Endpoint:   s.Endpoint,
		P256dh:     s.P256dh,
		Auth:       s.Auth,
		UserAgent:  s.UserAgent,
		CreatedAt:  s.CreatedAt,
		LastSeenAt: s.LastSeenAt,
	}
}
