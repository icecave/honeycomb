package cmd

import (
	"crypto/tls"
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
	ProxyProtocol      bool
	CheckTimeout       time.Duration
	MinTLSVersion      uint16
	MaxTLSVersion      uint16
	CipherSuite        []uint16
}

type certificateConfig struct {
	RedisAddress      string
	RedisPassword     string
	RedisCacheExpire  time.Duration
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
			RedisAddress:      env("REDIS_ADDR", ""),
			RedisPassword:     env("REDIS_PASSWORD", ""),
			RedisCacheExpire:  envDuration("REDIS_CACHE_EXPIRY", time.Minute),
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
		ProxyProtocol: envBool("PROXY_PROTOCOL", false),
		CheckTimeout:  envDuration("CHECK_TIMEOUT", 500*time.Millisecond),
		MinTLSVersion: envTLSVersion("TLS_MIN_VERSION"),
		MaxTLSVersion: envTLSVersion("TLS_MAX_VERSION"),
		CipherSuite:   envTLSCiphers("TLS_CIPHER_SUITE"),
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

func envTLSVersion(key string) uint16 {
	if value, ok := os.LookupEnv(key); ok {
		switch strings.ToLower(value) {
		case "tlsv1.0", "v1.0", "1.0", "1_0":
			return tls.VersionTLS10
		case "tlsv1.1", "v1.1", "1.1", "1_1":
			return tls.VersionTLS11
		case "tlsv1.2", "v1.2", "1.2", "1_2":
			return tls.VersionTLS12
		case "tlsv1.3", "v1.3", "1.3", "1_3":
			return tls.VersionTLS13
		default:
			return 0
		}
	}

	return 0
}

func envTLSCiphers(key string) []uint16 {
	if value, ok := os.LookupEnv(key); ok {
		o := []uint16{}
		cipherStrings := strings.Split(value, ":")
		cipherSuites := tls.CipherSuites()
		for _, cipherString := range cipherStrings {
			for _, cipherSuite := range cipherSuites {
				if strings.EqualFold(cipherString, cipherSuite.Name) {
					o = append(o, cipherSuite.ID)
				}
			}
		}

		if len(o) != 0 {
			return o
		}
	}

	return nil
}

func envBool(key string, def bool) bool {
	if value, ok := os.LookupEnv(key); ok {
		i, _ := strconv.ParseBool(value)
		return i
	}

	return def
}

func envDuration(key string, def time.Duration) time.Duration {
	if value, ok := os.LookupEnv(key); ok {
		d, err := time.ParseDuration(value)
		if err != nil {
			return def
		}
		return d
	}

	return def
}
