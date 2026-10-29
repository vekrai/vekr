package config

import "os"

var DefaultEnv = "development"

func Env() string {
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = DefaultEnv
	}
	return env
}
