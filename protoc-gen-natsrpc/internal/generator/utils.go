package generator

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

func firstToLower(s string) string {
	r, size := utf8.DecodeRuneInString(s)
	if r == utf8.RuneError && size <= 1 {
		return s
	}
	lc := unicode.ToLower(r)
	if r == lc {
		return s
	}
	return string(lc) + s[size:]
}

func firstToUpper(s string) string {
	r, size := utf8.DecodeRuneInString(s)
	if r == utf8.RuneError && size <= 1 {
		return s
	}
	lc := unicode.ToUpper(r)
	if r == lc {
		return s
	}
	return string(lc) + s[size:]
}

func toRoute(s string) string {
	if len(s) == 0 {
		return ""
	}

	var result strings.Builder
	// Предварительно выделяем память (примерный размер)
	result.Grow(len(s) + len(s)/2)

	// Обрабатываем первый символ
	first := s[0]
	if first >= 'A' && first <= 'Z' {
		// Первая заглавная -> строчная
		result.WriteByte(first + 32) // 'a' - 'A' = 32
	} else {
		result.WriteByte(first)
	}

	// Обрабатываем остальные символы
	for i := 1; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			// Перед заглавной буквой добавляем точку
			result.WriteByte('.')
			result.WriteByte(c + 32) // Преобразуем в строчную
		} else {
			result.WriteByte(c)
		}
	}

	return result.String()
}
