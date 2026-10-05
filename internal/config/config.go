package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config รวมค่าตั้งทั้งหมดที่อ่านจาก environment
//
// พอร์ตมาจาก vertex-event-service — ไม่มี fallback สำหรับค่าที่จำเป็น
// ตั้งไม่ครบแล้วไม่ยอม start ดีกว่าไปต่อ database ผิดตัวหรือส่ง push ด้วย
// VAPID key ผิดใบเงียบๆ
type Config struct {
	Port     string
	DB       DBConfig
	JWT      JWTConfig
	VAPID    VAPIDConfig
	Service  ServiceConfig
	Log      LogConfig
	Shutdown ShutdownConfig
}

type DBConfig struct {
	Host       string
	Port       string
	User       string
	Password   string
	Name       string
	SSLMode    string
	SearchPath string
}

type JWTConfig struct {
	PublicKeys string
	Issuer     string
	Audience   string
}

// VAPIDConfig คือคีย์คู่ที่ใช้เซ็น push message ตามมาตรฐาน RFC 8292
//
// Public key ฝังอยู่ใน JS ของ PWA อยู่แล้ว (ไม่ใช่ความลับ)
// แต่ Private key ต้องเก็บเป็นความลับ — ถ้าหลุดใครก็ส่ง push ปลอมไปหาผู้ใช้ได้
type VAPIDConfig struct {
	PublicKey  string
	PrivateKey string
	Subscriber string // "mailto:..." ตามข้อกำหนดของ VAPID
}

// ServiceConfig คุมการยืนยันตัวตนของ POST /send ซึ่งถูกเรียกโดย service อื่น
type ServiceConfig struct {
	Token string
}

type LogConfig struct {
	Level string
}

type ShutdownConfig struct {
	DrainDelay time.Duration
	Timeout    time.Duration
}

func (c DBConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s search_path=%s TimeZone=Asia/Bangkok",
		c.Host, c.Port, c.User, c.Password, c.Name, c.SSLMode, c.SearchPath,
	)
}

func (c DBConfig) Redacted() string {
	return fmt.Sprintf("host=%s port=%s user=%s dbname=%s search_path=%s",
		c.Host, c.Port, c.User, c.Name, c.SearchPath)
}

const minServiceTokenLength = 32

func Load() (Config, error) {
	cfg := Config{
		Port: env("PORT", "4003"),
		DB: DBConfig{
			Host:       env("DB_HOST", "localhost"),
			Port:       env("DB_PORT", "5432"),
			User:       os.Getenv("DB_USER"),
			Password:   os.Getenv("DB_PASSWORD"),
			Name:       env("DB_NAME", "vertex"),
			SSLMode:    env("DB_SSL_MODE", "disable"),
			SearchPath: env("DB_SEARCH_PATH", "notification"),
		},
		JWT: JWTConfig{
			PublicKeys: os.Getenv("JWT_PUBLIC_KEYS"),
			Issuer:     os.Getenv("JWT_ISSUER"),
			Audience:   os.Getenv("JWT_AUDIENCE"),
		},
		VAPID: VAPIDConfig{
			PublicKey:  os.Getenv("VAPID_PUBLIC_KEY"),
			PrivateKey: os.Getenv("VAPID_PRIVATE_KEY"),
			Subscriber: os.Getenv("VAPID_SUBSCRIBER"),
		},
		Service: ServiceConfig{
			Token: os.Getenv("PUSH_SERVICE_TOKEN"),
		},
		Log: LogConfig{Level: env("LOG_LEVEL", "info")},
		Shutdown: ShutdownConfig{
			DrainDelay: envDuration("SHUTDOWN_DRAIN_DELAY", 5*time.Second),
			Timeout:    envDuration("SHUTDOWN_TIMEOUT", 20*time.Second),
		},
	}

	var missing []string
	if cfg.DB.User == "" {
		missing = append(missing, "DB_USER")
	}
	if cfg.DB.Password == "" {
		missing = append(missing, "DB_PASSWORD")
	}
	if cfg.JWT.PublicKeys == "" {
		missing = append(missing, "JWT_PUBLIC_KEYS")
	}
	if cfg.VAPID.PublicKey == "" {
		missing = append(missing, "VAPID_PUBLIC_KEY")
	}
	if cfg.VAPID.PrivateKey == "" {
		missing = append(missing, "VAPID_PRIVATE_KEY")
	}
	if cfg.VAPID.Subscriber == "" {
		missing = append(missing, "VAPID_SUBSCRIBER")
	}
	if cfg.Service.Token == "" {
		missing = append(missing, "PUSH_SERVICE_TOKEN")
	}
	if len(missing) > 0 {
		return Config{}, fmt.Errorf("ไม่ได้ตั้ง environment variable ที่จำเป็น: %s", strings.Join(missing, ", "))
	}

	if len(cfg.Service.Token) < minServiceTokenLength {
		return Config{}, fmt.Errorf(
			"PUSH_SERVICE_TOKEN สั้นเกินไป (%d ตัวอักษร) ต้องอย่างน้อย %d — สร้างด้วย: openssl rand -base64 48",
			len(cfg.Service.Token), minServiceTokenLength)
	}
	if !strings.HasPrefix(cfg.VAPID.Subscriber, "mailto:") && !strings.HasPrefix(cfg.VAPID.Subscriber, "https://") {
		return Config{}, fmt.Errorf(
			"VAPID_SUBSCRIBER ต้องขึ้นต้นด้วย mailto: หรือ https:// ตามข้อกำหนดของ VAPID (ได้ %q)",
			cfg.VAPID.Subscriber)
	}

	return cfg, nil
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envDuration(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	if d, err := time.ParseDuration(v); err == nil {
		return d
	}
	if n, err := strconv.Atoi(v); err == nil {
		return time.Duration(n) * time.Second
	}
	return fallback
}
