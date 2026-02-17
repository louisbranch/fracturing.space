// Package main starts the chat real-time service and handles termination.
//
// The process is a transport adapter around chat room lifecycle and message
// streaming so campaign state remains owned by the game domain.
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	chatcmd "github.com/louisbranch/fracturing.space/internal/cmd/chat"
)

func main() {
	cfg, err := chatcmd.ParseConfig(flag.CommandLine, os.Args[1:])
	if err != nil {
		log.Fatalf("parse flags: %v", err)
	}
	log.SetPrefix("[CHAT] ")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := chatcmd.Run(ctx, cfg); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
