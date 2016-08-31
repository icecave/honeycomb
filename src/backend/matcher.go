package backend

import (
	"fmt"
	"strings"

	"golang.org/x/net/idna"
)

// Matcher matches a server name pattern against an incoming TLS request's
// server name.
type Matcher struct {
	Pattern    string
	wildPrefix bool
	wildSuffix bool
	fixedPart  string
}

// NewMatcher returns a new matcher for the given pattern.
func NewMatcher(pattern string) (*Matcher, error) {
	if pattern == "*" {
		return &Matcher{
			Pattern:    pattern,
			wildPrefix: true,
			wildSuffix: true,
		}, nil
	} else if pattern == "*.*" {
		return &Matcher{
			Pattern:    pattern,
			wildPrefix: true,
			wildSuffix: true,
			fixedPart:  ".",
		}, nil
	}

	domainPart := strings.ToLower(pattern)

	matcher := &Matcher{
		Pattern:    pattern,
		wildPrefix: strings.HasPrefix(pattern, "*."),
		wildSuffix: strings.HasSuffix(pattern, ".*"),
		fixedPart:  domainPart,
	}

	if matcher.wildPrefix {
		matcher.fixedPart = matcher.fixedPart[1:]
		domainPart = domainPart[2:]
	}

	if matcher.wildSuffix {
		matcher.fixedPart = matcher.fixedPart[:len(matcher.fixedPart)-1]
		domainPart = domainPart[:len(domainPart)-2]
	}

	if !isDomainName(domainPart) {
		return nil, fmt.Errorf(
			"'%s' is not a valid domain pattern",
			domainPart,
		)
	}

	return matcher, nil
}

// Match checks if the pattern matches the given server name.
func (matcher Matcher) Match(serverName string) bool {
	serverName = strings.ToLower(serverName)

	if matcher.wildPrefix && matcher.wildSuffix {
		return matcher.fixedPart == "" ||
			strings.Contains(serverName, matcher.fixedPart)
	} else if matcher.wildPrefix {
		return strings.HasSuffix(serverName, matcher.fixedPart)
	} else if matcher.wildSuffix {
		return strings.HasPrefix(serverName, matcher.fixedPart)
	}

	return serverName == matcher.fixedPart
}

// isDomainName checks if the given domain name is valid.
func isDomainName(domainName string) bool {
	domainName, err := idna.ToASCII(domainName)
	if err != nil || len(domainName) == 0 || len(domainName) > 255 {
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
