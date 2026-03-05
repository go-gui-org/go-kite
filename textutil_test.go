package main

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestSanitizeText(t *testing.T) {
	in := "hello\x00 world this_is_a_very_long_token_that_should_be_truncated"
	got := sanitizeText(in)
	if got == in {
		t.Fatalf("expected sanitize to modify input")
	}
	if got != "hello world this_is_a_very_long_..." {
		t.Fatalf("unexpected sanitize output: %q", got)
	}
}

func TestIndexesInString(t *testing.T) {
	s := "a😀b"
	if !indexesInString(s, 1, 5) {
		t.Fatalf("expected valid UTF-8 boundaries")
	}
	if indexesInString(s, 2, 5) {
		t.Fatalf("expected invalid start boundary")
	}
	if indexesInString(s, 1, 4) {
		t.Fatalf("expected invalid end boundary")
	}
}

func TestSanitizeTextPreservesUTF8(t *testing.T) {
	in := strings.Repeat("😀", 40)
	got := sanitizeText(in)
	if !utf8.ValidString(got) {
		t.Fatalf("expected valid UTF-8 output, got %q", got)
	}
	want := strings.Repeat("😀", 20) + "..."
	if got != want {
		t.Fatalf("unexpected sanitize output: got %q want %q", got, want)
	}
}

func TestIsSafeURI(t *testing.T) {
	if !isSafeURI("https://example.com") {
		t.Fatal("https should be safe")
	}
	if !isSafeURI("HTTP://example.com") {
		t.Fatal("uppercase scheme should be safe")
	}
	if isSafeURI("javascript:alert(1)") {
		t.Fatal("javascript URI should be unsafe")
	}
}
