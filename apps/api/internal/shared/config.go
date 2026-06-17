package shared

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"cloud.google.com/go/spanner"
)

type GCPConfig struct {
	SpannerProjectID     string
	SpannerInstanceID    string
	SpannerDatabaseID    string
	SpannerEmulatorHost  string
}

type WAConfig struct {
	WebhookSecret string
}

type WebConfig struct {
	CORSAllowedOrigins string
	TemplatesPath      string
	StaticPath         string
}

type ServerConfig struct {
	Port int
}

type Config struct {
	Server ServerConfig
	GCP    GCPConfig
	WA     WAConfig
	Web    WebConfig
}

func LoadConfig() (*Config, error) {
	port, err := strconv.Atoi(GetEnv("PORT", "8080"))
	if err != nil {
		return nil, fmt.Errorf("invalid PORT: %w", err)
	}

	return &Config{
		Server: ServerConfig{Port: port},
		GCP: GCPConfig{
			SpannerProjectID:    GetEnv("GCP_SPANNER_PROJECT_ID", ""),
			SpannerInstanceID:   GetEnv("GCP_SPANNER_INSTANCE_ID", ""),
			SpannerDatabaseID:   GetEnv("GCP_SPANNER_DATABASE_ID", ""),
			SpannerEmulatorHost: GetEnv("GCP_SPANNER_EMULATOR_HOST", ""),
		},
		WA: WAConfig{
			WebhookSecret: GetEnv("WEBHOOK_SECRET", ""),
		},
		Web: WebConfig{
			CORSAllowedOrigins: GetEnv("CORS_ALLOWED_ORIGINS", "*"),
			TemplatesPath:      GetEnv("TEMPLATES_PATH", "../web/templates"),
			StaticPath:         GetEnv("STATIC_PATH", "../web/static"),
		},
	}, nil
}

func (c *Config) SpannerDatabasePath() string {
	return fmt.Sprintf("projects/%s/instances/%s/databases/%s",
		c.GCP.SpannerProjectID, c.GCP.SpannerInstanceID, c.GCP.SpannerDatabaseID)
}

func NewSpannerClient(ctx context.Context, cfg *Config) (*spanner.Client, error) {
	if cfg.GCP.SpannerEmulatorHost != "" {
		os.Setenv("SPANNER_EMULATOR_HOST", cfg.GCP.SpannerEmulatorHost)
	}
	client, err := spanner.NewClient(ctx, cfg.SpannerDatabasePath())
	if err != nil {
		return nil, fmt.Errorf("failed to create Spanner client: %w", err)
	}
	return client, nil
}

func GetEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
