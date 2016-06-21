package config

import (
	"github.com/kelseyhightower/envconfig"
	"github.com/pkg/errors"
)

type Config struct {
	ListenAddress string `envconfig:"LISTEN_ADDRESS" default:"127.0.0.1:8080"`
	CanonicalURL  string `envconfig:"CANONICAL_URL"  default:"http://localhost:8080"`

	FacebookClientId     string `envconfig:"FACEBOOK_CLIENT_ID"     required:"true"`
	FacebookClientSecret string `envconfig:"FACEBOOK_CLIENT_SECRET" required:"true"`

	GoogleClientId     string `envconfig:"GOOGLE_CLIENT_ID"     required:"true"`
	GoogleClientSecret string `envconfig:"GOOGLE_CLIENT_SECRET" required:"true"`

	GitHubClientId     string `envconfig:"GITHUB_CLIENT_ID"     required:"true"`
	GitHubClientSecret string `envconfig:"GITHUB_CLIENT_SECRET" required:"true"`

	MongoURL string `envconfig:"MONGO_URL" default:"localhost"`

	SteemdRPCEndpointAddress string `envconfig:"STEEMD_RPC_ENDPOINT_ADDRESS" default:"ws://localhost:8090"`
}

func Load() (*Config, error) {
	var config Config
	if err := envconfig.Process("STEEMWATCH", &config); err != nil {
		return nil, errors.Wrap(err, "failed to load config from the environment")
	}
	return &config, nil
}
