package log

import (
	"testing"

	"github.com/rs/zerolog"
)

func TestParseLevel(t *testing.T) {
	cases := map[string]zerolog.Level{
		"debug":   zerolog.DebugLevel,
		"DEBUG":   zerolog.DebugLevel,
		"info":    zerolog.InfoLevel,
		"":        zerolog.InfoLevel,
		"warn":    zerolog.WarnLevel,
		"warning": zerolog.WarnLevel,
		"error":   zerolog.ErrorLevel,
		"bogus":   zerolog.InfoLevel,
	}
	for in, want := range cases {
		if got := parseLevel(in); got != want {
			t.Errorf("parseLevel(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestInitDoesNotPanic(t *testing.T) {
	Init("info")
	Init("debug")
	Init("")
}
