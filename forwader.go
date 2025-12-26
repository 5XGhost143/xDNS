package main

import (
	"crypto/sha256"
	"encoding/hex"
	"net"
	"time"
)

type Forwarder struct {
	upstream     string
	logger       *Logger
	cache        *Cache
	blocker      *Blocker
	upstreamAddr *net.UDPAddr
	connPool     chan *net.UDPConn
	poolSize     int
}

func NewForwarder(upstream string, logger *Logger, cache *Cache, blocker *Blocker) *Forwarder {
	upstreamAddr, err := net.ResolveUDPAddr("udp", upstream)
	if err != nil {
		logger.Error("Error resolving upstream address: %v", err)
		return nil
	}

	f := &Forwarder{
		upstream:     upstream,
		logger:       logger,
		cache:        cache,
		blocker:      blocker,
		upstreamAddr: upstreamAddr,
		poolSize:     500,
		connPool:     make(chan *net.UDPConn, 500),
	}

	for i := 0; i < f.poolSize; i++ {
		conn, err := net.DialUDP("udp", nil, upstreamAddr)
		if err != nil {
			logger.Error("Error establishing connection: %v", err)
			continue
		}
		f.connPool <- conn
	}

	return f
}

func (f *Forwarder) getConn() *net.UDPConn {
	select {
	case conn := <-f.connPool:
		return conn
	default:
		conn, _ := net.DialUDP("udp", nil, f.upstreamAddr)
		return conn
	}
}

func (f *Forwarder) returnConn(conn *net.UDPConn) {
	select {
	case f.connPool <- conn:
	default:
		conn.Close()
	}
}

func (f *Forwarder) Forward(query []byte, clientAddr *net.UDPAddr) ([]byte, error) {
	domain := f.extractDomain(query)
	f.logger.Query(clientAddr.IP.String(), domain)

	if f.blocker.IsBlocked(domain) {
		f.logger.Blocked(clientAddr.IP.String(), domain)
		return f.createBlockedResponse(query), nil
	}

	cacheKey := f.getCacheKey(query)
	if cached, found := f.cache.Get(cacheKey); found {
		f.logger.CacheHit(domain)
		return f.adjustTransactionID(cached, query), nil
	}

	conn := f.getConn()
	defer f.returnConn(conn)

	conn.SetDeadline(time.Now().Add(2 * time.Second))

	_, err := conn.Write(query)
	if err != nil {
		return nil, err
	}

	response := make([]byte, 4096)
	n, err := conn.Read(response)
	if err != nil {
		return nil, err
	}

	response = response[:n]
	f.cache.Set(cacheKey, response)

	return response, nil
}

func (f *Forwarder) createBlockedResponse(query []byte) []byte {
	if len(query) < 12 {
		return query
	}

	response := make([]byte, len(query))
	copy(response, query)

	response[2] = 0x81
	response[3] = 0x83

	return response
}

func (f *Forwarder) getCacheKey(query []byte) string {
	if len(query) < 12 {
		return ""
	}
	queryWithoutID := make([]byte, len(query))
	copy(queryWithoutID, query)
	queryWithoutID[0] = 0
	queryWithoutID[1] = 0

	hash := sha256.Sum256(queryWithoutID)
	return hex.EncodeToString(hash[:])
}

func (f *Forwarder) adjustTransactionID(cached []byte, query []byte) []byte {
	if len(cached) < 2 || len(query) < 2 {
		return cached
	}
	response := make([]byte, len(cached))
	copy(response, cached)
	response[0] = query[0]
	response[1] = query[1]
	return response
}

func (f *Forwarder) extractDomain(query []byte) string {
	if len(query) < 13 {
		return "unknown"
	}

	pos := 12
	domain := ""

	for pos < len(query) {
		length := int(query[pos])
		if length == 0 {
			break
		}
		pos++
		if pos+length > len(query) {
			return "invalid"
		}
		if domain != "" {
			domain += "."
		}
		domain += string(query[pos : pos+length])
		pos += length
	}

	if domain == "" {
		return "unknown"
	}
	return domain
}

func (f *Forwarder) Close() {
	close(f.connPool)
	for conn := range f.connPool {
		conn.Close()
	}
}
