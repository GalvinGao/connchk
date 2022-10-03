package config

import (
	"time"

	_ "github.com/joho/godotenv/autoload"
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	ServerAddr string `required:"true"`

	// The interval between each heartbeat.
	HeartbeatInterval time.Duration `default:"2s" split_words:"true" required:"true"`

	GracePeriod time.Duration `default:"10s" split_words:"true" required:"true"`

	// Temporarily disable the connection check.
	Disabled bool `default:"false"`

	NotifyTelegramBotToken string `split_words:"true" required:"true"`
	NotifyTelegramReceiver int64  `split_words:"true" required:"true"`

	NotifyTwilioAccountSID string `split_words:"true" required:"true"`
	NotifyTwilioAuthToken  string `split_words:"true" required:"true"`
	NotifyTwilioFromPhone  string `split_words:"true" required:"true"`
	NotifyTwilioToPhone    string `split_words:"true" required:"true"`
}

func Parse() (*Config, error) {
	var c Config
	if err := envconfig.Process("CONNCHK", &c); err != nil {
		envconfig.Usage("CONNCHK", &c)
		return nil, err
	}
	return &c, nil
}
