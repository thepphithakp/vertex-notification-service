package model

import (
	"time"

	"github.com/google/uuid"

	"github.com/vertex/notification-service/internal/domain"
)

type NotificationLogRow struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey"`

	UserID string `gorm:"not null"`
	Title  string `gorm:"not null"`
	Body   string `gorm:"not null"`
	URL    string

	CreatedAt time.Time `gorm:"not null;default:now()"`

	ReadAt *time.Time
}

func (NotificationLogRow) TableName() string { return "notification_logs" }

func (r NotificationLogRow) ToDomain() domain.NotificationLog {
	return domain.NotificationLog{
		ID:        r.ID,
		UserID:    r.UserID,
		Title:     r.Title,
		Body:      r.Body,
		URL:       r.URL,
		CreatedAt: r.CreatedAt,
		ReadAt:    r.ReadAt,
	}
}

func NotificationLogRowFromDomain(n domain.NotificationLog) NotificationLogRow {
	return NotificationLogRow{
		ID:        n.ID,
		UserID:    n.UserID,
		Title:     n.Title,
		Body:      n.Body,
		URL:       n.URL,
		CreatedAt: n.CreatedAt,
		ReadAt:    n.ReadAt,
	}
}
