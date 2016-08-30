package docker

import "golang.org/x/net/idna"

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
