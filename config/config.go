package config

import (
	"bytes"
	"embed"
	"fmt"
	"os"

	"github.com/spf13/viper"
)

type config struct {
	Db     *dbConfig
	Server *serverConfig
	Auth   *authConfig
	Header string
}

type dbConfig struct {
	Driver       string `mapstructure:"driver"`
	Host         string `mapstructure:"host"`
	Port         string `mapstructure:"port"`
	Username     string `mapstructure:"username"`
	Password     string `mapstructure:"password"`
	Database     string `mapstructure:"database"`
	DetailLog    bool   `mapstructure:"detail-log"`
	MaxOpenConns int    `mapstructure:"max-open-conns"`
	MaxIdleConns int    `mapstructure:"max-idle-conns"`
}

type serverConfig struct {
	Port string `mapstructure:"port"`
}

type authConfig struct {
	KeysEndpoint string `mapstructure:"keys-endpoint"`
}

var configuration *config

//go:embed config.yaml
var configPath embed.FS

func LoadConfig() (*config, error) {
	v := viper.New()
	v.SetConfigType("yaml")
	data, err := configPath.ReadFile("config.yaml")
	if err != nil {
		return nil, fmt.Errorf("error reading embedded config file:: %w", err)
	}
	err = v.ReadConfig(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("fatal error config file: %w", err)
	}

	err = v.Unmarshal(&configuration)
	if err != nil {
		return nil, err
	}

	fmt.Println("Successfully loaded config file - ", v.ConfigFileUsed())

	if os.Getenv("FERN_USERNAME") != "" {
		configuration.Db.Username = os.Getenv("FERN_USERNAME")
	}
	if os.Getenv("FERN_PASSWORD") != "" {
		configuration.Db.Password = os.Getenv("FERN_PASSWORD")
	}
	if os.Getenv("FERN_HOST") != "" {
		configuration.Db.Host = os.Getenv("FERN_HOST")
	}
	if os.Getenv("FERN_PORT") != "" {
		configuration.Db.Port = os.Getenv("FERN_PORT")
	}
	if os.Getenv("FERN_DATABASE") != "" {
		configuration.Db.Database = os.Getenv("FERN_DATABASE")
	}
	if os.Getenv("AUTH_KEYS_ENDPOINT") != "" {
		configuration.Auth.KeysEndpoint = os.Getenv("AUTH_KEYS_ENDPOINT")
	}
	if os.Getenv("FERN_HEADER_NAME") != "" {
		configuration.Header = os.Getenv("FERN_HEADER_NAME")
	}

	return configuration, nil
}

func GetDb() *dbConfig {
	return configuration.Db
}

func GetServer() *serverConfig {
	return configuration.Server
}

func GetHeaderName() string {
	return configuration.Header
}
