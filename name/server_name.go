package name

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"strings"

	"golang.org/x/net/idna"
)

// ServerName is a normalized TLS server name.
type ServerName struct {
	Unicode  string
	Punycode string
}

// Parse produces a ServerName value from a string, or panics if
// it is unable to do so.
func Parse(name string) ServerName {
	normalized, err := TryParse(name)
	if err != nil {
		panic(err)
	}

	return normalized
}

// TryParse attempts to produce a ServerName value from a string.
func TryParse(name string) (ServerName, error) {
	var normalized ServerName
	var err error

	lowercase := strings.ToLower(name)
	normalized.Punycode, err = idna.ToASCII(lowercase)
	if err != nil {
		return normalized, err
	} else if !isDomainName(normalized.Punycode) {
		return normalized, fmt.Errorf("invalid server name '%s'", name)
	}

	normalized.Unicode, err = idna.ToUnicode(lowercase)

	return normalized, err
}

// FromHTTP attempts to parse a server name from an HTTP request.
func FromHTTP(request *http.Request) (ServerName, error) {
	host, _, err := net.SplitHostPort(request.Host)
	if err != nil {
		host = request.Host
	}

	return TryParse(host)
}

// FromTLS attempts to parse a server name from a TLS request.
func FromTLS(info *tls.ClientHelloInfo) (ServerName, error) {
	return TryParse(info.ServerName)
}

// isDomainName checks if the given domain name is valid.
func isDomainName(domainName string) bool {
	if len(domainName) == 0 || len(domainName) > 255 {
		return false
	}

	hasLetter := false
	atomLength := 0
	previousChar := byte('.')

	for index := 0; index < len(domainName); index++ {
		char := domainName[index]

		switch {
		case 'a' <= char && char <= 'z':
			fallthrough
		case 'A' <= char && char <= 'Z':
			fallthrough
		case char == '_':
			hasLetter = true
			fallthrough
		case '0' <= char && char <= '9':
			atomLength++
		case char == '-':
			// Byte before dash cannot be dot.
			if previousChar == '.' {
				return false
			}
			atomLength++
		case char == '.':
			// Byte before dot cannot be dot, dash.
			if previousChar == '.' || previousChar == '-' {
				return false
			} else if atomLength > 63 || atomLength == 0 {
				return false
			}
			atomLength = 0
		default:
			return false
		}

		previousChar = char
	}

	return hasLetter &&
		previousChar != '-' &&
		previousChar != '.' &&
		atomLength < 64
}
