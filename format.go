package stock

import (
	"fmt"
	"strings"
)

func money(v float64) string {
	return fmt.Sprintf("$%.2f", v)
}

func percent(v float64) string {
	return fmt.Sprintf("%+.2f%%", v)
}

func millions(v float64) string {
	switch {
	case v >= 1_000_000:
		return fmt.Sprintf("$%.2fT", v/1_000_000)
	case v >= 1_000:
		return fmt.Sprintf("$%.2fB", v/1_000)
	default:
		return fmt.Sprintf("$%.2fM", v)
	}
}

func thousands(v int64) string {
	s := fmt.Sprintf("%d", v)
	sign := ""
	if strings.HasPrefix(s, "-") {
		sign, s = "-", s[1:]
	}
	var parts []string
	for len(s) > 3 {
		parts = append([]string{s[len(s)-3:]}, parts...)
		s = s[:len(s)-3]
	}
	return sign + strings.Join(append([]string{s}, parts...), ",")
}

func deref(v *float32) float64 {
	if v == nil {
		return 0
	}
	return float64(*v)
}

func derefString(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func derefInt(v *int64) int64 {
	if v == nil {
		return 0
	}
	return *v
}
