// Command indexer subscribes to chain WebSocket endpoints, decodes raw event
// logs via ABI, and publishes structured events to Kafka.
package main

import (
	"flag"
	"fmt"
	"os"
)

var version = "dev"

func main() {
	configPath := flag.String("config", "config/config.toml", "path to TOML config file")
	flag.Parse()

	fmt.Fprintf(os.Stdout, "chainpulse-indexer %s\n", version)
	fmt.Fprintf(os.Stdout, "config: %s\n", *configPath)
	fmt.Fprintf(os.Stdout, "TODO: load config, start chain listeners, produce to Kafka\n")
}
