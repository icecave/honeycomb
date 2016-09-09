package cmd

import (
	"os"
	"strconv"
	"time"
)

// Config holds configuration values for commands.
type Config struct {
	Port               string
	CACertificate      string
	CAKey              string
	ServerCertificate  string
	ServerKey          string
	DockerPollInterval time.Duration
}

// GetConfigFromEnvironment creates Config object based on the shell environment.
func GetConfigFromEnvironment() *Config {
	poll := envInt("DOCKER_POLL_INTERVAL", 0)

	return &Config{
		Port:               env("PORT", "8443"),
		CACertificate:      env("CA_CERT", "ca.crt"),
		CAKey:              env("CA_KEY", "ca.key"),
		ServerCertificate:  env("SERVER_CERT", "server.crt"),
		ServerKey:          env("SERVER_KEY", "server.key"),
		DockerPollInterval: time.Duration(poll) * time.Second,
	}
}

func env(key string, def string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}

	return def
}

func envInt(key string, def int64) int64 {
	if value, ok := os.LookupEnv(key); ok {
		i, _ := strconv.ParseInt(value, 10, 64)
		return i
	}

	return def
}
