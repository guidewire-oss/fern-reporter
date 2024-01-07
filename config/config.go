package config

import (
	"fmt"
	"github.com/spf13/viper"
	"os"
)

type config struct {
	Db     *dbConfig
	Server *serverConfig
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

var configuration *config

func LoadConfig(path string) (*config, error) {
	viper.SetConfigFile(path)
	viper.AutomaticEnv()

	err := viper.ReadInConfig()

	if err != nil {
		return nil, fmt.Errorf("fatal error config file: %w", err)
	}

	err = viper.Unmarshal(&configuration)
	if err != nil {
		return nil, err
	}

	fmt.Println("Successfully loaded config file - ", viper.ConfigFileUsed())

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

	return configuration, nil
}

func GetDb() *dbConfig {
	return configuration.Db
}

func GetServer() *serverConfig {
	return configuration.Server
}
