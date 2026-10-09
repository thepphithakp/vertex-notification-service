package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/vertex/notification-service/internal/adapter/repository/model"
	"github.com/vertex/notification-service/internal/domain"
)

type GORMNotificationLogRepository struct {
	db *gorm.DB
}

func NewGORMNotificationLogRepository(db *gorm.DB) *GORMNotificationLogRepository {
	return &GORMNotificationLogRepository{db: db}
}

func (r *GORMNotificationLogRepository) Create(ctx context.Context, log *domain.NotificationLog) error {
	row := model.NotificationLogRowFromDomain(*log)
	return r.db.WithContext(ctx).Create(&row).Error
}

func (r *GORMNotificationLogRepository) ListByUser(ctx context.Context, userID string, limit int) ([]domain.NotificationLog, error) {
	var rows []model.NotificationLogRow
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at desc").
		Limit(limit).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	logs := make([]domain.NotificationLog, 0, len(rows))
	for _, row := range rows {
		logs = append(logs, row.ToDomain())
	}
	return logs, nil
}

func (r *GORMNotificationLogRepository) UnreadCount(ctx context.Context, userID string) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&model.NotificationLogRow{}).
		Where("user_id = ? AND read_at IS NULL", userID).
		Count(&n).Error
	return n, err
}

func (r *GORMNotificationLogRepository) MarkAllRead(ctx context.Context, userID string) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&model.NotificationLogRow{}).
		Where("user_id = ? AND read_at IS NULL", userID).
		Update("read_at", now).Error
}
