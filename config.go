package main

import "time"

type Config struct {
	ListenAddr     string
	UpstreamServer string
	CacheTTL       time.Duration
	BufferSize     int
	Timeout        time.Duration
}

func NewConfig() *Config {
	return &Config{
		ListenAddr:     ":53",
		UpstreamServer: "1.1.1.1:53",
		CacheTTL:       15 * time.Minute,
		BufferSize:     4096,
		Timeout:        2 * time.Second,
	}
}
