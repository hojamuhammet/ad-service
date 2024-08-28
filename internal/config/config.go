package config

import (
	"log"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	HTTP     HTTPConfig     `yaml:"http"`
	Database DatabaseConfig `yaml:"database"`
	Redis    RedisConfig    `yaml:"redis"`
	Tracing  TracingConfig  `yaml:"tracing"`
	Logger   LoggerConfig   `yaml:"logger"`
}

type HTTPConfig struct {
	Port    int           `yaml:"port"`
	Timeout time.Duration `yaml:"timeout"`
}

type DatabaseConfig struct {
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Name     string `yaml:"name"`
}

type RedisConfig struct {
	Addr     string `yaml:"addr"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

type TracingConfig struct {
	Endpoint    string `yaml:"endpoint"`
	ServiceName string `yaml:"service_name"`
	Environment string `yaml:"environment"`
	Version     string `yaml:"version"`
}

type LoggerConfig struct {
	Level string `yaml:"level"`
}

func LoadConfig() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	viper.AddConfigPath(".") // if its current directory

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	if timeoutStr := viper.GetString("http.timeout"); timeoutStr != "" {
		timeout, err := time.ParseDuration(timeoutStr)
		if err != nil {
			return nil, err
		}
		config.HTTP.Timeout = timeout
	}

	return &config, nil
}

func MustLoadConfig() *Config {
	config, err := LoadConfig()
	if err != nil {
		log.Fatalf("Could not load configuration: %v", err)
	}
	return config
}
