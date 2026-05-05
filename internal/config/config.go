// Package config loads the ChainPulse TOML configuration with ${VAR}
// environment variable substitution.
package config

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/joho/godotenv"
	"github.com/pelletier/go-toml/v2"

	"github.com/0xfandom/chainpulse/internal/types"
)

// envVarPattern matches ${VAR_NAME} placeholders. Names accept letters,
// digits, and underscores; they must not start with a digit.
var envVarPattern = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)\}`)

// ErrMissingEnvVar is returned by Substitute when a referenced env var has
// no value in the environment.
var ErrMissingEnvVar = errors.New("missing environment variable")

// Load reads the TOML file at path, loads .env via godotenv (best-effort,
// no error if .env is missing), substitutes ${VAR} references from the
// process environment, parses the result into AppConfig, and validates
// required fields.
func Load(path string) (*types.AppConfig, error) {
	// godotenv populates os.Environ() so the env-var lookups below find
	// values from .env. Failure (no .env file) is fine in production.
	_ = godotenv.Load()

	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}

	substituted, err := Substitute(string(raw))
	if err != nil {
		return nil, fmt.Errorf("substitute env vars in %s: %w", path, err)
	}

	var cfg types.AppConfig
	if err := toml.Unmarshal([]byte(substituted), &cfg); err != nil {
		return nil, fmt.Errorf("parse toml %s: %w", path, err)
	}

	if err := validate(&cfg); err != nil {
		return nil, fmt.Errorf("validate config %s: %w", path, err)
	}

	return &cfg, nil
}

// Substitute replaces every ${VAR} reference in src with the value of the
// matching environment variable. Returns ErrMissingEnvVar if any reference
// is unset.
func Substitute(src string) (string, error) {
	var missing []string
	out := envVarPattern.ReplaceAllStringFunc(src, func(match string) string {
		name := match[2 : len(match)-1]
		val, ok := os.LookupEnv(name)
		if !ok {
			missing = append(missing, name)
			return match
		}
		return val
	})
	if len(missing) > 0 {
		return "", fmt.Errorf("%w: %s", ErrMissingEnvVar, strings.Join(missing, ", "))
	}
	return out, nil
}

func validate(cfg *types.AppConfig) error {
	if len(cfg.Kafka.Brokers) == 0 {
		return errors.New("kafka.brokers must not be empty")
	}
	if cfg.Kafka.TopicRawEvents == "" {
		return errors.New("kafka.topic_raw_events must be set")
	}
	if len(cfg.Chains) == 0 {
		return errors.New("at least one [[chains]] block is required")
	}
	for i, c := range cfg.Chains {
		if c.ChainID == 0 {
			return fmt.Errorf("chains[%d].chain_id must be non-zero", i)
		}
		if c.Name == "" {
			return fmt.Errorf("chains[%d].name must be set", i)
		}
		if c.RPCWSS == "" {
			return fmt.Errorf("chains[%d].rpc_wss must be set (chain=%s)", i, c.Name)
		}
	}
	return nil
}
