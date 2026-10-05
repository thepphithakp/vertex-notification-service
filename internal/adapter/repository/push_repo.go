package repository

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/vertex/notification-service/internal/domain"
)

type GORMSubscriptionRepository struct {
	db *gorm.DB
}

func NewGORMSubscriptionRepository(db *gorm.DB) *GORMSubscriptionRepository {
	return &GORMSubscriptionRepository{db: db}
}

// Upsert เขียนทับคีย์เข้ารหัสของอุปกรณ์เดิม แทนที่จะสร้างแถวใหม่
//
// browser อาจ subscribe ซ้ำด้วย endpoint เดิมแต่คีย์เปลี่ยน (เช่นหลัง
// reinstall PWA) — ux_push_subscriptions_user_endpoint กันแถวซ้ำไว้ที่ database
func (r *GORMSubscriptionRepository) Upsert(ctx context.Context, sub *domain.PushSubscription) error {
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "user_id"}, {Name: "endpoint"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"p256dh", "auth", "user_agent", "last_seen_at",
			}),
		}).
		Create(sub).Error
}

func (r *GORMSubscriptionRepository) Delete(ctx context.Context, userID, endpoint string) error {
	return r.db.WithContext(ctx).
		Where("user_id = ? AND endpoint = ?", userID, endpoint).
		Delete(&domain.PushSubscription{}).Error
}

func (r *GORMSubscriptionRepository) DeleteByID(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&domain.PushSubscription{}, "id = ?", id).Error
}

func (r *GORMSubscriptionRepository) ListByUser(ctx context.Context, userID string) ([]domain.PushSubscription, error) {
	var subs []domain.PushSubscription
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&subs).Error
	return subs, err
}
