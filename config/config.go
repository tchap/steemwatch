package config

import (
	"github.com/kelseyhightower/envconfig"
	"github.com/pkg/errors"
)

type Config struct {
	Env string `envconfig:"ENVIRONMENT" default:"development"`

	ListenAddress string `envconfig:"LISTEN_ADDRESS" default:"127.0.0.1:8080"`
	CanonicalURL  string `envconfig:"CANONICAL_URL"  default:"http://localhost:8080"`

	FacebookClientId     string `envconfig:"FACEBOOK_CLIENT_ID"     required:"true"`
	FacebookClientSecret string `envconfig:"FACEBOOK_CLIENT_SECRET" required:"true"`

	RedditClientId     string `envconfig:"REDDIT_CLIENT_ID"     required:"true"`
	RedditClientSecret string `envconfig:"REDDIT_CLIENT_SECRET" required:"true"`

	GoogleClientId     string `envconfig:"GOOGLE_CLIENT_ID"     required:"true"`
	GoogleClientSecret string `envconfig:"GOOGLE_CLIENT_SECRET" required:"true"`

	GitHubClientId     string `envconfig:"GITHUB_CLIENT_ID"     required:"true"`
	GitHubClientSecret string `envconfig:"GITHUB_CLIENT_SECRET" required:"true"`

	TelegramBotToken string `envconfig:"TELEGRAM_BOT_TOKEN" required:"true"`

	DiscordBotToken string `envconfig:"DISCORD_BOT_TOKEN" required:"true"`

	MongoURL string `envconfig:"MONGO_URL" default:"localhost"`

	SteemdDisabled             bool     `envconfig:"STEEMD_DISABLED"`
	SteemdRPCEndpointAddresses []string `envconfig:"STEEMD_RPC_ENDPOINT_ADDRESSES" default:"ws://localhost:8090"`

	BlockProcessorWorkerCount uint `envconfig:"BLOCK_PROCESSOR_WORKER_COUNT" default:"10"`
}

func Load() (*Config, error) {
	var config Config
	if err := envconfig.Process("STEEMWATCH", &config); err != nil {
		return nil, errors.Wrap(err, "failed to load config from the environment")
	}
	return &config, nil
}
