package bootstrap

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"gorm.io/gorm"

	"github.com/vertex/notification-service/internal/adapter/handler"
	"github.com/vertex/notification-service/internal/adapter/pushsender"
	"github.com/vertex/notification-service/internal/adapter/repository"
	"github.com/vertex/notification-service/internal/application"
	"github.com/vertex/notification-service/internal/config"
	"github.com/vertex/notification-service/pkg/middleware"
)

// bodyLimit จำกัดขนาด request — subscription payload เล็กมาก (endpoint + สองคีย์)
const bodyLimit = 64 << 10 // 64KB

// NewApp ประกอบ HTTP layer ทั้งหมด
func NewApp(db *gorm.DB, cfg config.Config, auth middleware.AuthConfig) (*fiber.App, *Health) {
	app := fiber.New(fiber.Config{
		BodyLimit:             bodyLimit,
		ErrorHandler:          middleware.ErrorHandler,
		DisableStartupMessage: true,
	})

	app.Use(recover.New())
	app.Use(middleware.NewRequestID())
	app.Use(middleware.NewMetrics())
	app.Use(middleware.NewAccessLog())
	app.Use(cors.New())

	health := NewHealth(db)
	app.Get("/livez", health.Liveness)
	app.Get("/readyz", health.Readiness)
	app.Get("/health", health.Liveness)
	app.Get("/metrics", middleware.MetricsHandler())

	repo := repository.NewGORMSubscriptionRepository(db)
	sender := pushsender.NewWebPushSender(cfg.VAPID.PublicKey, cfg.VAPID.PrivateKey, cfg.VAPID.Subscriber)
	svc := application.NewPushService(repo, sender)
	h := handler.NewPushHandler(svc)

	// ── ผู้ใช้ผ่าน PWA: subscribe/unsubscribe ต้องมี JWT ────────────────────
	h.RegisterUserRoutes(app.Group("/api/v1/push", middleware.NewAuth(auth)))

	// ── service อื่นเรียกสั่งส่ง: ต้องมี service token ไม่ใช่ JWT ของผู้ใช้ ──
	//
	// เหตุผลเดียวกับ event-service POST /events — ผู้เรียกคือ service อื่น
	// (เช่น pet-service ตอนถึงกำหนดเตือนจอดรถซ้อน) ไม่ใช่ผู้ใช้โดยตรง
	//
	// ⚠️ ห้ามตั้งชื่อ path ที่ขึ้นต้นด้วย "/api/v1/push" (เช่น "/api/v1/push-internal")
	//
	//    Fiber ผูก middleware ของ group ด้วย string-prefix match ไม่ใช่ตาม
	//    segment ของ path — "/api/v1/push-internal/send" ขึ้นต้นด้วยตัวอักษร
	//    เดียวกับ "/api/v1/push" ทุกตัว จึงโดน middleware ของ group ด้านบน
	//    (NewAuth ที่ต้องการ JWT) ครอบไปด้วย ทำให้ตอบ 401 "ต้องมี Authorization:
	//    Bearer" แทนที่จะเช็ค X-Service-Token เหมือนที่ตั้งใจ — เจอจริงตอนเทส
	//    local (ดู event-service pkg/bootstrap/app.go ที่เตือนเรื่องเดียวกันไว้
	//    กับ "/api/v1" ครอบ "/api/v1/admin")
	//
	//    "/api/v1/internal/push" ปลอดภัยเพราะอักษรตัวที่ 9 ต่างกัน (i ≠ p)
	//    จึงไม่มีทางเป็น prefix ของกันและกันได้ไม่ว่าจะเพิ่ม path ย่อยอะไรต่อ
	h.RegisterServiceRoutes(app.Group("/api/v1/internal/push",
		middleware.NewServiceToken(cfg.Service.Token)))

	return app, health
}
