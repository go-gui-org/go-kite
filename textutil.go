package main

import (
	"fmt"
	"log"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/mike-ward/go-gui/gui"
)

const (
	maxFieldLen = 35
	truncateAt  = 20
)

func truncateLongFields(s string) string {
	fields := strings.Fields(s)
	for i, f := range fields {
		if len(f) > maxFieldLen {
			fields[i] = f[:truncateAt] + "..."
		}
	}
	return strings.Join(fields, " ")
}

func removeControlChars(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r == '\t' || r == '\n' || r == '\r' || !unicode.IsControl(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func sanitizeText(s string) string {
	return truncateLongFields(removeControlChars(s))
}

func isUTF8Boundary(s string, idx int) bool {
	if idx <= 0 || idx >= len(s) {
		return true
	}
	return utf8.RuneStart(s[idx])
}

func indexesInString(s string, start, end int) bool {
	return end > 0 && end <= len(s) && start >= 0 && start < end &&
		isUTF8Boundary(s, start) && isUTF8Boundary(s, end)
}

func logError(msg string) {
	log.Printf("%s > %s", time.Now().Format("15:04:05"), msg)
}

func changeFontSize(delta, minSize, maxSize float32, w *gui.Window) {
	t, err := gui.CurrentTheme().AdjustFontSize(delta, minSize, maxSize)
	if err != nil {
		logError(err.Error())
		return
	}
	w.SetTheme(t)
}

func relativeShort(createdAt time.Time, now time.Time) string {
	if createdAt.IsZero() {
		return "now"
	}
	d := now.Sub(createdAt)
	if d < time.Minute {
		return "now"
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	days := int(d.Hours() / 24)
	if days < 7 {
		return fmt.Sprintf("%dd", days)
	}
	if days < 30 {
		return fmt.Sprintf("%dw", days/7)
	}
	if days < 365 {
		return fmt.Sprintf("%dmo", days/30)
	}
	return fmt.Sprintf("%dy", days/365)
}
