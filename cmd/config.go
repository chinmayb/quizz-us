package cmd

import (
	log "log/slog"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	port     *int
	grpcPort *int
	logLevel *string
)

func setConfig(cmd *cobra.Command) {
	// Define flags before parsing
	port = pflag.Int("port", 8080, "http port")
	grpcPort = pflag.Int("grpc-port", 9090, "grpc port")
	logLevel = pflag.String("log-level", "info", "log level (debug, info, warn, error)")
	
	cmd.HasPersistentFlags()
	pflag.Parse()
}

func logParser(level string) log.Leveler {
	switch level {
	case "debug":
		return log.LevelDebug
	case "info":
		return log.LevelInfo
	case "warn":
		return log.LevelWarn
	case "error":
		return log.LevelError
	default:
		return log.LevelInfo
	}
}
