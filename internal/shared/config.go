package shared

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"cloud.google.com/go/spanner"
)

type Config struct {
	SpannerProjectID   string
	SpannerInstanceID  string
	SpannerDatabaseID  string
	Port               int
	WebhookSecret      string
	CORSAllowedOrigins string
}

func LoadConfig() (*Config, error) {
	port, err := strconv.Atoi(GetEnv("PORT", "8080"))
	if err != nil {
		return nil, fmt.Errorf("invalid PORT: %w", err)
	}

	return &Config{
		SpannerProjectID:   GetEnv("SPANNER_PROJECT_ID", ""),
		SpannerInstanceID:  GetEnv("SPANNER_INSTANCE_ID", ""),
		SpannerDatabaseID:  GetEnv("SPANNER_DATABASE_ID", ""),
		Port:               port,
		WebhookSecret:      GetEnv("WEBHOOK_SECRET", ""),
		CORSAllowedOrigins: GetEnv("CORS_ALLOWED_ORIGINS", "*"),
	}, nil
}

func (c *Config) SpannerDatabasePath() string {
	return fmt.Sprintf("projects/%s/instances/%s/databases/%s",
		c.SpannerProjectID, c.SpannerInstanceID, c.SpannerDatabaseID)
}

func NewSpannerClient(ctx context.Context, cfg *Config) (*spanner.Client, error) {
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
