package main

import (
	"os"

	"go.uber.org/zap"

	"github.com/cldmnky/hyperdhcp/cmd"
)

func main() {
	// Initilize logger
	loggerConfig := zap.Config{
		Encoding:         "json",
		Level:            zap.NewAtomicLevelAt(zap.DebugLevel),
		OutputPaths:      []string{"stdout", "log.txt"},
		ErrorOutputPaths: []string{"stderr"},
	}
	logger, err := loggerConfig.Build()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()
	zap.ReplaceGlobals(logger)
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
