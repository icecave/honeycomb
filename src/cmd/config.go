package cmd

import (
	"os"
	"strconv"
	"time"
)

// Config holds configuration values for commands.
type Config struct {
	Port               string
	InsecurePort       string
	DockerPollInterval time.Duration

	AWSAccessKeyID     string
	AWSSecretAccessKey string

	Certificates certificateConfig
}

type certificateConfig struct {
	BasePath          string
	IssuerCertificate string
	IssuerKey         string
	ServerCertificate string
	ServerKey         string

	S3Bucket   string
	S3Endpoint string
}

// GetConfigFromEnvironment creates Config object based on the shell environment.
func GetConfigFromEnvironment() *Config {
	return &Config{
		Port:               env("PORT", "8443"),
		InsecurePort:       env("REDIRECT_PORT", "8080"),
		DockerPollInterval: time.Duration(envInt("DOCKER_POLL_INTERVAL", 0)) * time.Second,

		AWSAccessKeyID:     env("AWS_ACCESS_KEY_ID", ""),
		AWSSecretAccessKey: env("AWS_SECRET_ACCESS_KEY", ""),

		Certificates: certificateConfig{
			BasePath:          env("CERTIFICATE_PATH", ""),
			IssuerCertificate: env("ISSUER_CERT", "ca.crt"),
			IssuerKey:         env("ISSUER_KEY", "ca.key"),
			ServerCertificate: env("SERVER_CERT", "server.crt"),
			ServerKey:         env("SERVER_KEY", "server.key"),
			S3Bucket:          env("CERTIFICATE_S3_BUCKET", ""),
			S3Endpoint:        env("CERTIFICATE_S3_ENDPOINT", "s3.amazonaws.com"),
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
