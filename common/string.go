package common

import "strings"

func TrimCannotbeseen(src string) (afterTrim string) {
	afterTrim = strings.TrimFunc(src, func(w rune) bool {
		if w < 32 {
			return true
		}
		if w == '\n' {
			return true
		}
		if w == '\t' {
			return true
		}
		if w == '\r' {
			return true
		}
		if w == ' ' {
			return true
		}
		return false
	})
	return
}
