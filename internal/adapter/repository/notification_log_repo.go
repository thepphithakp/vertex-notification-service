package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/vertex/notification-service/internal/domain"
)

type GORMNotificationLogRepository struct {
	db *gorm.DB
}

func NewGORMNotificationLogRepository(db *gorm.DB) *GORMNotificationLogRepository {
	return &GORMNotificationLogRepository{db: db}
}

func (r *GORMNotificationLogRepository) Create(ctx context.Context, log *domain.NotificationLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

func (r *GORMNotificationLogRepository) ListByUser(ctx context.Context, userID string, limit int) ([]domain.NotificationLog, error) {
	var logs []domain.NotificationLog
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at desc").
		Limit(limit).
		Find(&logs).Error
	return logs, err
}

func (r *GORMNotificationLogRepository) UnreadCount(ctx context.Context, userID string) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&domain.NotificationLog{}).
		Where("user_id = ? AND read_at IS NULL", userID).
		Count(&n).Error
	return n, err
}

func (r *GORMNotificationLogRepository) MarkAllRead(ctx context.Context, userID string) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&domain.NotificationLog{}).
		Where("user_id = ? AND read_at IS NULL", userID).
		Update("read_at", now).Error
}
