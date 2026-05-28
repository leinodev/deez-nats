package natsevents

import (
	"strings"
	"unicode"
)

// jetStreamPushDurable возвращает имя durable push-консьюмера.
// base — общий префикс из JetStreamEventsOptions.ConsumerDurable; при пустом base — ephemeral ("").
// nRoutes — число маршрутов JetStream на этом роутере (>1 требует уникального суффикса на маршрут).
func jetStreamPushDurable(base, routeSubject string, nRoutes int) string {
	base = strings.TrimSpace(base)
	if base == "" {
		return ""
	}
	key := sanitizeJetStreamDurable(base)
	if nRoutes <= 1 {
		return key
	}
	return sanitizeJetStreamDurable(key + "_" + subjectToDurableToken(routeSubject))
}

func subjectToDurableToken(subject string) string {
	var b strings.Builder
	for _, r := range subject {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		} else {
			b.WriteRune('_')
		}
	}
	s := b.String()
	for strings.Contains(s, "__") {
		s = strings.ReplaceAll(s, "__", "_")
	}
	return strings.Trim(s, "_")
}

func sanitizeJetStreamDurable(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	var b strings.Builder
	for _, r := range s {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
		case r == '-' || r == '_':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	out := strings.Trim(b.String(), "_")
	if out == "" {
		return "consumer"
	}
	if len(out) > 200 {
		out = out[:200]
	}
	return out
}
