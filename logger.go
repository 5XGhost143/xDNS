package main

import (
	"fmt"
	"log"
	"sync"
)

type Logger struct {
	mu sync.Mutex
}

func NewLogger() *Logger {
	return &Logger{}
}

func (l *Logger) Info(format string, args ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	log.Printf("[INFO] "+format, args...)
}

func (l *Logger) Error(format string, args ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	log.Printf("[ERROR] "+format, args...)
}

func (l *Logger) Query(clientIP, domain string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	log.Printf("[QUERY] Client: %s | Domain: %s", clientIP, domain)
}

func (l *Logger) Blocked(clientIP, domain string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	log.Printf("[BLOCKED] Client: %s | Domain: %s", clientIP, domain)
}

func (l *Logger) CacheHit(domain string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	log.Printf("[CACHE] Hit for domain: %s", domain)
}

func (l *Logger) Debug(format string, args ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	fmt.Printf("[DEBUG] "+format+"\n", args...)
}
