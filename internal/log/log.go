// Package log wires zerolog as the project-wide structured logger.
//
// Production: JSON output to stdout. Debug: console pretty-print.
// Standard fields used across the pipeline:
//   - chain:      human-readable chain name (e.g. "base")
//   - chain_id:   numeric EVM chain id
//   - block:      block number
//   - component:  service/subsystem ("indexer", "processor", "kafka_producer")
package log

import (
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Standard structured field names. Use these instead of free-form keys so
// downstream filters (Loki, Grafana) stay consistent.
const (
	FieldChain     = "chain"
	FieldChainID   = "chain_id"
	FieldBlock     = "block"
	FieldComponent = "component"
)

// Init configures the global zerolog logger. levelStr accepts "debug",
// "info", "warn", "error" (case-insensitive). Unknown values fall back to
// info. When the level is debug, output is rendered as console pretty-print
// for local development; otherwise JSON to stdout.
func Init(levelStr string) {
	level := parseLevel(levelStr)
	zerolog.SetGlobalLevel(level)
	zerolog.TimeFieldFormat = time.RFC3339Nano

	if level == zerolog.DebugLevel {
		log.Logger = zerolog.New(zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
		}).With().Timestamp().Logger()
		return
	}

	log.Logger = zerolog.New(os.Stdout).With().Timestamp().Logger()
}

// Component returns a child logger pre-tagged with a component name.
func Component(name string) zerolog.Logger {
	return log.With().Str(FieldComponent, name).Logger()
}

// Chain returns a child logger pre-tagged with chain name + id.
func Chain(name string, id uint64) zerolog.Logger {
	return log.With().Str(FieldChain, name).Uint64(FieldChainID, id).Logger()
}

func parseLevel(s string) zerolog.Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return zerolog.DebugLevel
	case "warn", "warning":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	case "info", "":
		return zerolog.InfoLevel
	default:
		return zerolog.InfoLevel
	}
}
