package config

import (
	"os"
	"strconv"
)

// Config holds all configuration for the application
type Config struct {
	// App
	AppEnv      string
	AppPort     string
	AppURL      string
	FrontendURL string

	// Google OAuth
	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURL  string

	// JWT
	JWTSecret              string
	JWTExpiryHours         int
	RefreshTokenExpiryDays int

	// PostgreSQL
	PostgresHost     string
	PostgresPort     string
	PostgresUser     string
	PostgresPassword string
	PostgresDB       string
	PostgresSSLMode  string
	DatabaseURL      string

	// Redis
	RedisHost     string
	RedisPort     string
	RedisPassword string
	RedisDB       int
	RedisURL      string

	// Gemini AI
	GeminiAPIKey string
	GeminiModel  string

	// SMTP / Email
	SMTPHost      string
	SMTPPort      int
	SMTPUser      string
	SMTPPassword  string
	SMTPFromName  string
	SMTPFromEmail string

	// Storage
	StorageType      string
	StorageLocalPath string

	// Session
	SessionSecret string

	// Rate Limiting
	RateLimitRequests int
	RateLimitWindow   int

	// Jitsi
	JitsiDomain string

	// Security
	InvitationExpiryHours       int
	MaxLoginAttempts            int
	SuspiciousActivityThreshold int
}

// Load reads config from environment variables
func Load() *Config {
	return &Config{
		// App
		AppEnv:      getEnv("APP_ENV", "development"),
		AppPort:     getEnv("APP_PORT", "8080"),
		AppURL:      getEnv("APP_URL", "http://localhost:8080"),
		FrontendURL: getEnv("FRONTEND_URL", "http://localhost:3000"),

		// Google OAuth
		GoogleClientID:     getEnv("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret: getEnv("GOOGLE_CLIENT_SECRET", ""),
		GoogleRedirectURL:  getEnv("GOOGLE_REDIRECT_URL", "http://localhost:8080/api/auth/google/callback"),

		// JWT
		JWTSecret:              getEnv("JWT_SECRET", "default-secret-change-in-production"),
		JWTExpiryHours:         getEnvInt("JWT_EXPIRY_HOURS", 24),
		RefreshTokenExpiryDays: getEnvInt("REFRESH_TOKEN_EXPIRY_DAYS", 30),

		// PostgreSQL
		PostgresHost:     getEnv("POSTGRES_HOST", "localhost"),
		PostgresPort:     getEnv("POSTGRES_PORT", "5432"),
		PostgresUser:     getEnv("POSTGRES_USER", "kanban_user"),
		PostgresPassword: getEnv("POSTGRES_PASSWORD", "kanban_password"),
		PostgresDB:       getEnv("POSTGRES_DB", "kanban_db"),
		PostgresSSLMode:  getEnv("POSTGRES_SSL_MODE", "disable"),
		DatabaseURL:      getEnv("DATABASE_URL", ""),

		// Redis
		RedisHost:     getEnv("REDIS_HOST", "localhost"),
		RedisPort:     getEnv("REDIS_PORT", "6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       getEnvInt("REDIS_DB", 0),
		RedisURL:      getEnv("REDIS_URL", "redis://localhost:6379"),

		// Gemini AI
		GeminiAPIKey: getEnv("GEMINI_API_KEY", ""),
		GeminiModel:  getEnv("GEMINI_MODEL", "gemini-1.5-flash"),

		// SMTP
		SMTPHost:      getEnv("SMTP_HOST", "smtp.gmail.com"),
		SMTPPort:      getEnvInt("SMTP_PORT", 587),
		SMTPUser:      getEnv("SMTP_USER", ""),
		SMTPPassword:  getEnv("SMTP_PASSWORD", ""),
		SMTPFromName:  getEnv("SMTP_FROM_NAME", "Kanban Platform"),
		SMTPFromEmail: getEnv("SMTP_FROM_EMAIL", ""),

		// Storage
		StorageType:      getEnv("STORAGE_TYPE", "local"),
		StorageLocalPath: getEnv("STORAGE_LOCAL_PATH", "./uploads"),

		// Session
		SessionSecret: getEnv("SESSION_SECRET", "default-session-secret"),

		// Rate Limiting
		RateLimitRequests: getEnvInt("RATE_LIMIT_REQUESTS", 100),
		RateLimitWindow:   getEnvInt("RATE_LIMIT_WINDOW_SECONDS", 60),

		// Jitsi
		JitsiDomain: getEnv("JITSI_DOMAIN", "meet.jit.si"),

		// Security
		InvitationExpiryHours:       getEnvInt("INVITATION_EXPIRY_HOURS", 72),
		MaxLoginAttempts:            getEnvInt("MAX_LOGIN_ATTEMPTS", 10),
		SuspiciousActivityThreshold: getEnvInt("SUSPICIOUS_ACTIVITY_THRESHOLD", 20),
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultVal
}
