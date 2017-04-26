package cmd

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds configuration values for commands.
type Config struct {
	Port               string
	InsecurePort       string
	DockerPollInterval time.Duration
	Certificates       certificateConfig
}

type certificateConfig struct {
	BasePath          string
	IssuerCertificate string
	IssuerKey         string
	ServerCertificate string
	ServerKey         string
	CABundles         []string
}

// GetConfigFromEnvironment creates Config object based on the shell environment.
func GetConfigFromEnvironment() *Config {
	return &Config{
		Port:               env("PORT", "8443"),
		InsecurePort:       env("REDIRECT_PORT", "8080"),
		DockerPollInterval: time.Duration(envInt("DOCKER_POLL_INTERVAL", 0)) * time.Second,
		Certificates: certificateConfig{
			BasePath:          env("CERTIFICATE_PATH", "/run/secrets/"),
			IssuerCertificate: env("ISSUER_CERT", "honeycomb-ca.crt"),
			IssuerKey:         env("ISSUER_KEY", "honeycomb-ca.key"),
			ServerCertificate: env("SERVER_CERT", "honeycomb-server.crt"),
			ServerKey:         env("SERVER_KEY", "honeycomb-server.key"),
			CABundles: strings.Split(
				env("CA_PATH", "/app/etc/ca-bundle.pem,/run/secrets/ca-bundle.pem"),
				",",
			),
		},
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
