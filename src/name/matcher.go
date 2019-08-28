package name

import (
	"fmt"
	"strings"
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

	if _, err := TryParse(domainPart); err != nil {
		return nil, fmt.Errorf(
			"'%s' is not a valid server name pattern",
			domainPart,
		)
	}

	return matcher, nil
}

// Match checks if the pattern matches the given server name.
//
// It returns a score indicating the strength of the match. A value of 0 or less
// indicates that no match was made.
func (matcher Matcher) Match(serverName ServerName) int {
	score := 1 + len(matcher.fixedPart)

	if matcher.wildPrefix && matcher.wildSuffix {
		if strings.Contains(serverName.Unicode, matcher.fixedPart) {
			return score
		}
	} else if matcher.wildPrefix {
		if strings.HasSuffix(serverName.Unicode, matcher.fixedPart) {
			return score
		}
	} else if matcher.wildSuffix {
		if strings.HasPrefix(serverName.Unicode, matcher.fixedPart) {
			return score
		}
	} else {
		if serverName.Unicode == matcher.fixedPart {
			// an exact match always scores higher that a wildcard match that matches
			// the same number of characters.
			const max = int(^uint(0) >> 1)
			return max
		}
	}

	return 0
}
