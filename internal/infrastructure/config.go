package infrastructure

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	DBUrl                string `mapstructure:"db_url"`
	DailyTieringSchedule string `mapstructure:"daily_tiering_schedule"`
	OutboxProcessor      struct {
		BatchSize uint64 `mapstructure:"batch_size"`
		Schedule  string `mapstructure:"schedule"`
	} `mapstructure:"outbox_processor"`
}

// LoadConfig parses config file from path.
func LoadConfig(path string) (Config, error) {
	v := viper.New()
	v.SetConfigFile(path)

	// Replace dots in nested keys (outbox_processor.batch_size) with
	// underscores for the environment (OUTBOX_PROCESSOR_BATCH_SIZE).
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		return Config{}, fmt.Errorf("could not read config: %w", err)
	}

	var s Config
	if err := v.Unmarshal(&s); err != nil {
		return Config{}, fmt.Errorf("unable to parse config: %w", err)
	}
	if s.DBUrl == "" {
		return Config{}, fmt.Errorf("DB_URL is required")
	}
	if s.DailyTieringSchedule == "" {
		return Config{}, fmt.Errorf("DAILY_TIERING_SCHEDULE is required")
	}
	if s.OutboxProcessor.BatchSize < 1 || s.OutboxProcessor.BatchSize > 1024 {
		return Config{}, fmt.Errorf("OUTBOX_PROCESSOR_BATCH_SIZE must be between 1 and 1024")
	}
	if s.DailyTieringSchedule == "" {
		return Config{}, fmt.Errorf("OUTBOX_PROCESSOR_SCHEDULE is required")
	}
	return s, nil
}
