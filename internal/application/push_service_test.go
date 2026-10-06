package application

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"

	"github.com/vertex/notification-service/internal/domain"
	"github.com/vertex/notification-service/internal/port"
)

type fakeRepo struct {
	subs      map[string][]domain.PushSubscription // keyed by userID
	upserted  []domain.PushSubscription
	deletedID []uuid.UUID
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{subs: map[string][]domain.PushSubscription{}}
}

func (r *fakeRepo) Upsert(ctx context.Context, sub *domain.PushSubscription) error {
	r.upserted = append(r.upserted, *sub)
	r.subs[sub.UserID] = append(r.subs[sub.UserID], *sub)
	return nil
}

func (r *fakeRepo) Delete(ctx context.Context, userID, endpoint string) error {
	var kept []domain.PushSubscription
	for _, s := range r.subs[userID] {
		if s.Endpoint != endpoint {
			kept = append(kept, s)
		}
	}
	r.subs[userID] = kept
	return nil
}

func (r *fakeRepo) DeleteByID(ctx context.Context, id uuid.UUID) error {
	r.deletedID = append(r.deletedID, id)
	for userID, subs := range r.subs {
		var kept []domain.PushSubscription
		for _, s := range subs {
			if s.ID != id {
				kept = append(kept, s)
			}
		}
		r.subs[userID] = kept
	}
	return nil
}

func (r *fakeRepo) ListByUser(ctx context.Context, userID string) ([]domain.PushSubscription, error) {
	return r.subs[userID], nil
}

type fakeLogRepo struct {
	created []domain.NotificationLog
}

func (r *fakeLogRepo) Create(ctx context.Context, log *domain.NotificationLog) error {
	r.created = append(r.created, *log)
	return nil
}

func (r *fakeLogRepo) ListByUser(ctx context.Context, userID string, limit int) ([]domain.NotificationLog, error) {
	return nil, nil
}

func (r *fakeLogRepo) UnreadCount(ctx context.Context, userID string) (int64, error) {
	return 0, nil
}

func (r *fakeLogRepo) MarkAllRead(ctx context.Context, userID string) error {
	return nil
}

type fakeSender struct {
	statusByEndpoint map[string]int
	sent             []domain.PushSubscription
}

func (s *fakeSender) Send(ctx context.Context, sub domain.PushSubscription, payload []byte) (int, error) {
	s.sent = append(s.sent, sub)
	if status, ok := s.statusByEndpoint[sub.Endpoint]; ok {
		return status, nil
	}
	return http.StatusCreated, nil
}

func TestSubscribe_RejectsNonHTTPSEndpoint(t *testing.T) {
	svc := NewPushService(newFakeRepo(), &fakeLogRepo{}, &fakeSender{})
	err := svc.Subscribe(context.Background(), "user-1", port.SubscribeInput{
		Endpoint: "http://insecure.example/push",
		P256dh:   "key", Auth: "auth",
	})
	if err == nil {
		t.Fatal("ต้อง reject endpoint ที่ไม่ใช่ https")
	}
}

func TestSubscribe_RejectsMissingKeys(t *testing.T) {
	svc := NewPushService(newFakeRepo(), &fakeLogRepo{}, &fakeSender{})
	err := svc.Subscribe(context.Background(), "user-1", port.SubscribeInput{
		Endpoint: "https://push.example/abc",
	})
	if err == nil {
		t.Fatal("ต้อง reject ที่ไม่มี p256dh/auth")
	}
}

func TestSubscribe_StoresValidSubscription(t *testing.T) {
	repo := newFakeRepo()
	svc := NewPushService(repo, &fakeLogRepo{}, &fakeSender{})
	err := svc.Subscribe(context.Background(), "user-1", port.SubscribeInput{
		Endpoint: "https://push.example/abc",
		P256dh:   "key", Auth: "auth", UserAgent: "test-agent",
	})
	if err != nil {
		t.Fatalf("ไม่ควร error: %v", err)
	}
	if len(repo.upserted) != 1 {
		t.Fatalf("ต้องมี subscription ถูก upsert 1 รายการ ได้ %d", len(repo.upserted))
	}
	if repo.upserted[0].UserID != "user-1" {
		t.Errorf("userID ผิด: %s", repo.upserted[0].UserID)
	}
}

func TestSendToUser_CountsSentAndPrunesDeadEndpoints(t *testing.T) {
	repo := newFakeRepo()
	svc := NewPushService(repo, &fakeLogRepo{}, &fakeSender{})

	aliveID := uuid.New()
	deadID := uuid.New()
	repo.subs["user-1"] = []domain.PushSubscription{
		{ID: aliveID, UserID: "user-1", Endpoint: "https://push.example/alive"},
		{ID: deadID, UserID: "user-1", Endpoint: "https://push.example/dead"},
	}

	sender := &fakeSender{statusByEndpoint: map[string]int{
		"https://push.example/dead": http.StatusGone,
	}}
	svc.sender = sender

	result, err := svc.SendToUser(context.Background(), "user-1", port.Message{
		Title: "เตือนจอดรถ", Body: "ถึงกำหนดแล้ว",
	})
	if err != nil {
		t.Fatalf("ไม่ควร error: %v", err)
	}
	if result.Sent != 1 || result.Pruned != 1 {
		t.Fatalf("คาดว่า sent=1 pruned=1 ได้ sent=%d pruned=%d", result.Sent, result.Pruned)
	}
	if len(repo.deletedID) != 1 || repo.deletedID[0] != deadID {
		t.Fatalf("ต้องลบเฉพาะ subscription ที่ตาย (%s) ได้ %v", deadID, repo.deletedID)
	}
	remaining, _ := repo.ListByUser(context.Background(), "user-1")
	if len(remaining) != 1 || remaining[0].ID != aliveID {
		t.Fatalf("subscription ที่ยังไม่ตายต้องเหลืออยู่")
	}
}

func TestSendToUser_RejectsEmptyMessage(t *testing.T) {
	svc := NewPushService(newFakeRepo(), &fakeLogRepo{}, &fakeSender{})
	_, err := svc.SendToUser(context.Background(), "user-1", port.Message{})
	if err == nil {
		t.Fatal("ต้อง reject title/body ว่าง")
	}
}

func TestSendToUser_AlwaysLogsForInAppFeed(t *testing.T) {
	// ไม่มี subscription เลยสักตัว — ยังต้องบันทึก log เพราะ in-app feed
	// ควรเห็นว่า "ระบบพยายามแจ้งแล้ว" แม้ผู้ใช้ไม่เคยเปิด push
	logRepo := &fakeLogRepo{}
	svc := NewPushService(newFakeRepo(), logRepo, &fakeSender{})

	_, err := svc.SendToUser(context.Background(), "user-1", port.Message{
		Title: "เตือนจอดรถ", Body: "ถึงกำหนดแล้ว",
	})
	if err != nil {
		t.Fatalf("ไม่ควร error: %v", err)
	}
	if len(logRepo.created) != 1 {
		t.Fatalf("ต้องบันทึก notification log 1 รายการแม้ไม่มี subscription ได้ %d", len(logRepo.created))
	}
	if logRepo.created[0].UserID != "user-1" || logRepo.created[0].Title != "เตือนจอดรถ" {
		t.Errorf("เนื้อหา log ผิด: %+v", logRepo.created[0])
	}
}
