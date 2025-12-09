package config

import (
	"os"
	"strings"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env/v2"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/structs"
	"github.com/knadh/koanf/v2"
	log "github.com/sirupsen/logrus"
)

type Application struct {
	Host     string   `koanf:"host"`
	Frontend Frontend `koanf:"frontend"`
	ClickUp  ClickUp  `koanf:"clickup"`
	Google   Google   `koanf:"google"`
	Database Database `koanf:"db"`
}

type Frontend struct {
	Enabled bool `koanf:"enabled"`
}

type ClickUp struct {
	ClientId     string `koanf:"clientid"`
	ClientSecret string `koanf:"clientsecret"`
}

type Google struct {
	ClientId     string `koanf:"clientid"`
	ClientSecret string `koanf:"clientsecret"`
}

type Database struct {
	Host   string `koanf:"host"`
	Port   int    `koanf:"port"`
	User   string `koanf:"user"`
	Pass   string `koanf:"pass"`
	Name   string `koanf:"name"`
	Schema string `koanf:"schema"`
}

func Load(path string) (Application, error) {
	var k = koanf.New(".")

	err := k.Load(structs.Provider(Application{
		Host: "http://localhost:3000",
		Frontend: Frontend{
			Enabled: true,
		},
		Database: Database{
			Host:   "localhost",
			Port:   5432,
			User:   "klokku",
			Pass:   "",
			Name:   "klokku",
			Schema: "klokku",
		},
	}, "koanf"), nil)
	if err != nil {
		log.Errorf("error loading config from structs: %v", err)
		return Application{}, err
	}

	if err := k.Load(file.Provider(path), yaml.Parser()); err != nil {
		if os.IsNotExist(err) {
			log.Infof("Config file not found at %s, using defaults and environment variables", path)
		} else {
			log.Errorf("error loading config from YAML: %v", err)
			return Application{}, err
		}
	} else {
		log.Infof("Loaded configuration from file: %s", path)
	}

	err = k.Load(env.Provider(".", env.Opt{
		Prefix: "KLOKKU_",
		TransformFunc: func(k, v string) (string, any) {
			// Transform the key.
			k = strings.ReplaceAll(strings.ToLower(strings.TrimPrefix(k, "KLOKKU_")), "_", ".")
			return k, v
		},
	}), nil)
	if err != nil {
		log.Errorf("error loading config from envs: %v", err)
		return Application{}, err
	}

	var app Application
	if err := k.Unmarshal("", &app); err != nil {
		return Application{}, err
	}

	return app, nil
}
