package security_helpers

import "unicode/utf8"

func Truncate(s []byte, max int) []byte {
	if max <= 0 {
		return []byte("")
	}

	ss := string(s)

	if utf8.RuneCountInString(ss) < max {
		return s
	}

	return []byte(string([]rune(ss)[:max]))
}
