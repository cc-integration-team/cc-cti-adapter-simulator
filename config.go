package main

import (
	"time"

	pkgredis "github.com/cc-integration-team/cc-pkg/pkg/redis"
	"github.com/spf13/viper"
)

type Config struct {
	Redis pkgredis.RedisConfig `mapstructure:"redis"`
	Push  PushConfig           `mapstructure:"push"`
}

type PushConfig struct {
	Interval            time.Duration `mapstructure:"interval"`
	Count               int           `mapstructure:"count"`
	ConnectedInterval   time.Duration `mapstructure:"connectedInterval"`
	TerminatedInterval  time.Duration `mapstructure:"terminatedInterval"`
}

func LoadConfig(path string) (*Config, error) {
	var cfg Config
	v := viper.NewWithOptions(viper.KeyDelimiter(":"))

	v.AddConfigPath(path)
	v.SetConfigName("config")
	v.SetConfigType("yml")

	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}

	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
