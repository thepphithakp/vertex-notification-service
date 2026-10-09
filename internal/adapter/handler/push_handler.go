package handler

import (
	"errors"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/vertex/notification-service/internal/application"
	"github.com/vertex/notification-service/internal/domain"
	"github.com/vertex/notification-service/internal/port"
	"github.com/vertex/notification-service/pkg/middleware"
)

// notificationLogResponse คือรูปแบบ JSON ที่ตอบให้ client — เดิมใช้
// domain.NotificationLog ตรงๆ ผ่าน port.NotificationFeed.Items (มี json tag
// ติดอยู่กับ domain type เอง) ย้ายมาที่นี่ไม่ให้ domain ต้องรู้จัก wire
// format คงชื่อ field และรูปแบบเดิมทุกตัวอักษรไว้ ไม่ให้ PWA client พัง
type notificationLogResponse struct {
	ID        uuid.UUID  `json:"id"`
	UserID    string     `json:"userId"`
	Title     string     `json:"title"`
	Body      string     `json:"body"`
	URL       string     `json:"url"`
	CreatedAt time.Time  `json:"createdAt"`
	ReadAt    *time.Time `json:"readAt,omitempty"`
}

func toNotificationLogResponse(n domain.NotificationLog) notificationLogResponse {
	return notificationLogResponse{
		ID:        n.ID,
		UserID:    n.UserID,
		Title:     n.Title,
		Body:      n.Body,
		URL:       n.URL,
		CreatedAt: n.CreatedAt,
		ReadAt:    n.ReadAt,
	}
}

// notificationFeedResponse คือรูปแบบ JSON ของ port.NotificationFeed — เดิม
// c.JSON(feed) ตรงๆ พึ่ง json tag ของ domain.NotificationLog ใน Items
type notificationFeedResponse struct {
	Items       []notificationLogResponse `json:"items"`
	UnreadCount int64                     `json:"unreadCount"`
}

func toNotificationFeedResponse(f port.NotificationFeed) notificationFeedResponse {
	items := make([]notificationLogResponse, len(f.Items))
	for i, n := range f.Items {
		items[i] = toNotificationLogResponse(n)
	}
	return notificationFeedResponse{Items: items, UnreadCount: f.UnreadCount}
}

type PushHandler struct {
	useCase port.PushUseCase
}

func NewPushHandler(useCase port.PushUseCase) *PushHandler {
	return &PushHandler{useCase: useCase}
}

// RegisterUserRoutes ผูก endpoint ที่ตัวแอป (PWA) เรียกแทนผู้ใช้ ต้องมี JWT
func (h *PushHandler) RegisterUserRoutes(r fiber.Router) {
	r.Post("/subscribe", h.Subscribe)
	r.Delete("/subscribe", h.Unsubscribe)
	r.Get("/notifications", h.ListNotifications)
	r.Post("/notifications/read", h.MarkAllRead)
}

// RegisterServiceRoutes ผูก endpoint ที่ service อื่นเรียกเพื่อสั่งส่ง push
// ต้องมี service token — ไม่ใช่ JWT ของผู้ใช้ เพราะผู้เรียกคือ service ไม่ใช่คน
func (h *PushHandler) RegisterServiceRoutes(r fiber.Router) {
	r.Post("/send", h.Send)
}

type subscribeRequest struct {
	Endpoint string `json:"endpoint"`
	Keys     struct {
		P256dh string `json:"p256dh"`
		Auth   string `json:"auth"`
	} `json:"keys"`
}

func (h *PushHandler) Subscribe(c *fiber.Ctx) error {
	actor, ok := middleware.ActorFrom(c)
	if !ok {
		return unauthorized(c)
	}

	var req subscribeRequest
	if err := c.BodyParser(&req); err != nil {
		return badRequest(c, "อ่าน request body ไม่ได้")
	}

	err := h.useCase.Subscribe(c.UserContext(), actor.UserID, port.SubscribeInput{
		Endpoint:  req.Endpoint,
		P256dh:    req.Keys.P256dh,
		Auth:      req.Keys.Auth,
		UserAgent: c.Get(fiber.HeaderUserAgent),
	})
	if err != nil {
		return handleUseCaseError(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

type unsubscribeRequest struct {
	Endpoint string `json:"endpoint"`
}

func (h *PushHandler) Unsubscribe(c *fiber.Ctx) error {
	actor, ok := middleware.ActorFrom(c)
	if !ok {
		return unauthorized(c)
	}

	var req unsubscribeRequest
	if err := c.BodyParser(&req); err != nil {
		return badRequest(c, "อ่าน request body ไม่ได้")
	}

	if err := h.useCase.Unsubscribe(c.UserContext(), actor.UserID, req.Endpoint); err != nil {
		return handleUseCaseError(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

type sendRequest struct {
	UserID string `json:"userId"`
	Title  string `json:"title"`
	Body   string `json:"body"`
	Tag    string `json:"tag"`
	URL    string `json:"url"`
}

func (h *PushHandler) Send(c *fiber.Ctx) error {
	var req sendRequest
	if err := c.BodyParser(&req); err != nil {
		return badRequest(c, "อ่าน request body ไม่ได้")
	}
	if req.UserID == "" {
		return badRequest(c, "userId ต้องไม่ว่าง")
	}

	result, err := h.useCase.SendToUser(c.UserContext(), req.UserID, port.Message{
		Title: req.Title,
		Body:  req.Body,
		Tag:   req.Tag,
		URL:   req.URL,
	})
	if err != nil {
		return handleUseCaseError(c, err)
	}
	return c.JSON(result)
}

func (h *PushHandler) ListNotifications(c *fiber.Ctx) error {
	actor, ok := middleware.ActorFrom(c)
	if !ok {
		return unauthorized(c)
	}
	feed, err := h.useCase.ListNotifications(c.UserContext(), actor.UserID)
	if err != nil {
		return handleUseCaseError(c, err)
	}
	return c.JSON(toNotificationFeedResponse(feed))
}

func (h *PushHandler) MarkAllRead(c *fiber.Ctx) error {
	actor, ok := middleware.ActorFrom(c)
	if !ok {
		return unauthorized(c)
	}
	if err := h.useCase.MarkAllRead(c.UserContext(), actor.UserID); err != nil {
		return handleUseCaseError(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func handleUseCaseError(c *fiber.Ctx, err error) error {
	var ve *application.ValidationError
	if errors.As(err, &ve) {
		return badRequest(c, ve.Error())
	}
	return fiber.NewError(fiber.StatusInternalServerError, err.Error())
}

func badRequest(c *fiber.Ctx, msg string) error {
	return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
		"error":     msg,
		"requestId": c.Get(middleware.HeaderRequestID),
	})
}

func unauthorized(c *fiber.Ctx) error {
	return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
		"error":     "ยังไม่ได้ยืนยันตัวตน",
		"requestId": c.Get(middleware.HeaderRequestID),
	})
}
