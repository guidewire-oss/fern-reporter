package config

import (
	"github.com/go-yaml/yaml"
	"io/ioutil"
)

type config struct {
	Db     *dbCofnig
	Server *serverConfig
}

type dbCofnig struct {
	Driver       string `yaml:"driver"`
	Host         string `yaml:"host"`
	Port         string `yaml:"port"`
	Username     string `yaml:"username"`
	Password     string `yaml:"password"`
	Database     string `yaml:"database"`
	DetailLog    bool   `yaml:"detail-log"`
	MaxOpenConns int    `yaml:"max-open-conns"`
	MaxIdleConns int    `yaml:"max-idle-conns"`
}

type serverConfig struct {
	Port string `yaml:"port"`
}

var configuration *config

func LoadConfig(path string) error {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(data, &configuration)
	if err != nil {
		return err
	}
	return err
}

func GetDb() *dbCofnig {
	return configuration.Db
}

func GetServer() *serverConfig {
	return configuration.Server
}
