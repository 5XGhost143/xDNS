package main

import (
	"net"
	"sync"
)

type Server struct {
	listenAddr string
	forwarder  *Forwarder
	logger     *Logger
	conn       *net.UDPConn
	wg         sync.WaitGroup
	stopChan   chan struct{}
}

func NewServer(listenAddr string, forwarder *Forwarder, logger *Logger) *Server {
	return &Server{
		listenAddr: listenAddr,
		forwarder:  forwarder,
		logger:     logger,
		stopChan:   make(chan struct{}),
	}
}

func (s *Server) Start() error {
	addr, err := net.ResolveUDPAddr("udp", s.listenAddr)
	if err != nil {
		s.logger.Error("Error resolving the list address: %v", err)
		return err
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		s.logger.Error("Error while starting dns: %v", err)
		return err
	}

	s.conn = conn

	err = s.conn.SetReadBuffer(8 * 1024 * 1024)
	if err != nil {
		s.logger.Error("Error setting the read buffer: %v", err)
	}

	err = s.conn.SetWriteBuffer(8 * 1024 * 1024)
	if err != nil {
		s.logger.Error("Error setting the write buffer: %v", err)
	}

	bufferPool := &sync.Pool{
		New: func() interface{} {
			b := make([]byte, 4096)
			return &b
		},
	}

	semaphore := make(chan struct{}, 5000)

	for {
		select {
		case <-s.stopChan:
			return nil
		default:
			bufferPtr := bufferPool.Get().(*[]byte)
			buffer := *bufferPtr

			n, clientAddr, err := s.conn.ReadFromUDP(buffer)
			if err != nil {
				bufferPool.Put(bufferPtr)
				continue
			}

			query := make([]byte, n)
			copy(query, buffer[:n])
			bufferPool.Put(bufferPtr)

			semaphore <- struct{}{}
			s.wg.Add(1)
			go func() {
				s.handleQuery(query, clientAddr)
				<-semaphore
			}()
		}
	}
}

func (s *Server) handleQuery(query []byte, clientAddr *net.UDPAddr) {
	defer s.wg.Done()

	response, err := s.forwarder.Forward(query, clientAddr)
	if err != nil {
		s.logger.Error("Error during forwarding: %v", err)
		return
	}

	_, err = s.conn.WriteToUDP(response, clientAddr)
	if err != nil {
		s.logger.Error("Error sending response: %v", err)
	}
}

func (s *Server) Stop() {
	close(s.stopChan)
	if s.conn != nil {
		s.conn.Close()
	}
	s.wg.Wait()
	s.forwarder.Close()
	s.logger.Info("DNS stopped.")
}
