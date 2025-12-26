package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

func main() {
	upstreamServer := flag.String("upstream", "1.1.1.1:53", "Upstream DNS IP")
	flag.Parse()

	logger := NewLogger()
	config := NewConfig()

	upstream := *upstreamServer
	if !strings.Contains(upstream, ":") {
		upstream = upstream + ":53"
	}
	config.UpstreamServer = upstream

	blocker, err := NewBlocker("blacklist.ini", logger)
	if err != nil {
		log.Fatalf("Error loading blacklist: %v", err)
	}

	cache := NewCache(config.CacheTTL)
	forwarder := NewForwarder(config.UpstreamServer, logger, cache, blocker)
	server := NewServer(config.ListenAddr, forwarder, logger)

	go server.Start()

	logger.Info("DNS started on port: %s", config.ListenAddr)
	logger.Info("Upstream Server: %s", config.UpstreamServer)
	logger.Info("blacklist entries: %d", blocker.Count())

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	logger.Info("Shutdown signal received, terminating server...")
	server.Stop()
}
