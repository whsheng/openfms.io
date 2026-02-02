package server

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/redis/go-redis/v9"

	"openfms/gateway/internal/adapter"
	"openfms/gateway/internal/config"
	"openfms/gateway/internal/protocol"
)

// TCPServer handles TCP connections from GPS devices
type TCPServer struct {
	config    *config.Config
	redis     *redis.Client
	nats      *nats.Conn
	listener  net.Listener
	adapters  []protocol.ProtocolAdapter
	detector  *adapter.JT808Detector
	sessions  sync.Map // map[string]*Session
	ctx       context.Context
	cancel    context.CancelFunc
}

// Session represents a device connection
type Session struct {
	ConnID     string
	DeviceID   string
	Conn       net.Conn
	Adapter    protocol.ProtocolAdapter
	GatewayID  string
	ClientIP   string
	LastActive time.Time
	mu         sync.RWMutex
}

// NewTCPServer creates a new TCP server
func NewTCPServer(cfg *config.Config, redisClient *redis.Client, natsConn *nats.Conn) *TCPServer {
	ctx, cancel := context.WithCancel(context.Background())
	return &TCPServer{
		config:   cfg,
		redis:    redisClient,
		nats:     natsConn,
		detector: adapter.NewJT808Detector(),
		ctx:      ctx,
		cancel:   cancel,
	}
}

// Start starts the TCP server
func (s *TCPServer) Start() error {
	addr := fmt.Sprintf(":%d", s.config.GatewayPort)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}
	s.listener = listener

	log.Printf("[Gateway] TCP server listening on %s", addr)

	// Start HTTP server for gateway management
	go s.startHTTPServer()

	// Start downlink consumer
	go s.startDownlinkConsumer()

	// Accept connections
	go s.acceptLoop()

	return nil
}

// Stop stops the TCP server
func (s *TCPServer) Stop() {
	s.cancel()
	if s.listener != nil {
		s.listener.Close()
	}
	s.sessions.Range(func(key, value interface{}) bool {
		if session, ok := value.(*Session); ok {
			session.Conn.Close()
		}
		return true
	})
}

func (s *TCPServer) acceptLoop() {
	connID := 0
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.ctx.Done():
				return
			default:
				log.Printf("[Gateway] Accept error: %v", err)
				continue
			}
		}

		connID++
		session := &Session{
			ConnID:     fmt.Sprintf("%s-%d", s.config.GatewayID, connID),
			Conn:       conn,
			GatewayID:  s.config.GatewayID,
			ClientIP:   conn.RemoteAddr().String(),
			LastActive: time.Now(),
		}

		go s.handleConnection(session)
	}
}

func (s *TCPServer) handleConnection(session *Session) {
	defer func() {
		s.cleanupSession(session)
		session.Conn.Close()
	}()

	log.Printf("[Gateway] New connection: %s from %s", session.ConnID, session.ClientIP)

	reader := bufio.NewReader(session.Conn)
	buffer := make([]byte, 4096)
	var pending []byte

	for {
		select {
		case <-s.ctx.Done():
			return
		default:
		}

		session.Conn.SetReadDeadline(time.Now().Add(300 * time.Second))
		n, err := reader.Read(buffer)
		if err != nil {
			if err != io.EOF {
				log.Printf("[Gateway] Read error from %s: %v", session.ConnID, err)
			}
			return
		}

		pending = append(pending, buffer[:n]...)
		session.LastActive = time.Now()

		// Process packets
		for len(pending) > 0 {
			packet, rest, err := s.extractPacket(pending)
			if err != nil {
				log.Printf("[Gateway] Packet extraction error: %v", err)
				pending = rest
				continue
			}
			if packet == nil {
				// Incomplete packet, wait for more data
				break
			}

			pending = rest
			s.handlePacket(session, packet)
		}
	}
}

func (s *TCPServer) extractPacket(data []byte) ([]byte, []byte, error) {
	// JT808 packet: starts with 0x7E, ends with 0x7E
	// Find start marker
	startIdx := -1
	for i := 0; i < len(data); i++ {
		if data[i] == 0x7E {
			startIdx = i
			break
		}
	}
	if startIdx == -1 {
		// No start marker found, discard all
		return nil, nil, nil
	}

	// Find end marker
	endIdx := -1
	for i := startIdx + 1; i < len(data); i++ {
		if data[i] == 0x7E {
			endIdx = i
			break
		}
	}
	if endIdx == -1 {
		// Incomplete packet
		return nil, data[startIdx:], nil
	}

	packet := data[startIdx : endIdx+1]
	rest := data[endIdx+1:]
	return packet, rest, nil
}

func (s *TCPServer) handlePacket(session *Session, packet []byte) {
	// Detect protocol if not set
	if session.Adapter == nil {
		adapter, matched := s.detector.Match(packet)
		if !matched {
			log.Printf("[Gateway] Unknown protocol from %s", session.ConnID)
			return
		}
		session.Adapter = adapter
		log.Printf("[Gateway] Protocol detected: %s for %s", adapter.Protocol(), session.ConnID)
	}

	// Decode packet
	msg, err := session.Adapter.Decode(packet)
	if err != nil {
		log.Printf("[Gateway] Decode error: %v", err)
		return
	}

	// Update session with device ID
	if msg.DeviceID != "" && session.DeviceID == "" {
		session.DeviceID = msg.DeviceID
		s.sessions.Store(msg.DeviceID, session)
		s.registerSession(session)
	}

	// Handle heartbeat
	if session.Adapter.IsHeartbeat(packet) {
		ack, err := session.Adapter.GenerateHeartbeatAck(packet)
		if err == nil && ack != nil {
			session.Conn.Write(ack)
		}
		s.updateSessionTTL(session)
	}

	// Publish to NATS for processing
	if msg != nil {
		msgData, _ := json.Marshal(msg)
		subject := fmt.Sprintf("fms.uplink.%s", msg.Type)
		s.nats.Publish(subject, msgData)
		s.nats.Publish("fms.uplink.all", msgData)
		log.Printf("[Gateway] Published %s message from device %s", msg.Type, msg.DeviceID)
	}
}

func (s *TCPServer) registerSession(session *Session) {
	key := fmt.Sprintf("fms:sess:%s", session.DeviceID)
	value := fmt.Sprintf("%s:%s:%s", session.GatewayID, session.ConnID, session.ClientIP)

	err := s.redis.Set(s.ctx, key, value, 300*time.Second).Err()
	if err != nil {
		log.Printf("[Gateway] Failed to register session: %v", err)
		return
	}

	log.Printf("[Gateway] Session registered: %s -> %s", session.DeviceID, value)
}

func (s *TCPServer) updateSessionTTL(session *Session) {
	if session.DeviceID == "" {
		return
	}

	key := fmt.Sprintf("fms:sess:%s", session.DeviceID)
	s.redis.Expire(s.ctx, key, 300*time.Second)

	// Update device shadow
	shadowKey := fmt.Sprintf("fms:shadow:%s", session.DeviceID)
	s.redis.HSet(s.ctx, shadowKey, "ts", time.Now().Unix())
	s.redis.Expire(s.ctx, shadowKey, 24*time.Hour)
}

func (s *TCPServer) cleanupSession(session *Session) {
	log.Printf("[Gateway] Connection closed: %s", session.ConnID)

	if session.DeviceID != "" {
		s.sessions.Delete(session.DeviceID)

		// Remove from Redis
		key := fmt.Sprintf("fms:sess:%s", session.DeviceID)
		s.redis.Del(s.ctx, key)
	}
}

func (s *TCPServer) startHTTPServer() {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/sessions", s.handleSessions)
	mux.HandleFunc("/send-command", s.handleSendCommand)

	addr := fmt.Sprintf(":%d", s.config.HTTPPort)
	log.Printf("[Gateway] HTTP server listening on %s", addr)

	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("[Gateway] HTTP server error: %v", err)
		}
	}()

	<-s.ctx.Done()
	server.Shutdown(context.Background())
}

func (s *TCPServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":     "ok",
		"gateway_id": s.config.GatewayID,
	})
}

func (s *TCPServer) handleSessions(w http.ResponseWriter, r *http.Request) {
	sessions := make([]map[string]interface{}, 0)

	s.sessions.Range(func(key, value interface{}) bool {
		if session, ok := value.(*Session); ok {
			sessions = append(sessions, map[string]interface{}{
				"conn_id":     session.ConnID,
				"device_id":   session.DeviceID,
				"client_ip":   session.ClientIP,
				"protocol":    session.Adapter.Protocol(),
				"last_active": session.LastActive,
			})
		}
		return true
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sessions)
}

func (s *TCPServer) handleSendCommand(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		DeviceID string                 `json:"device_id"`
		Type     string                 `json:"type"`
		Params   map[string]interface{} `json:"params"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Find session
	value, ok := s.sessions.Load(req.DeviceID)
	if !ok {
		http.Error(w, "Device not connected", http.StatusNotFound)
		return
	}

	session := value.(*Session)
	if session.Adapter == nil {
		http.Error(w, "Protocol not determined", http.StatusBadRequest)
		return
	}

	cmd := protocol.StandardCommand{
		Type:   req.Type,
		Params: req.Params,
	}

	data, err := session.Adapter.Encode(cmd)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = session.Conn.Write(data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "sent",
	})
}

func (s *TCPServer) startDownlinkConsumer() {
	subject := fmt.Sprintf("gateway.downlink.%s", s.config.GatewayID)
	sub, err := s.nats.Subscribe(subject, func(msg *nats.Msg) {
		var cmd struct {
			DeviceID string                 `json:"device_id"`
			Type     string                 `json:"type"`
			Params   map[string]interface{} `json:"params"`
		}

		if err := json.Unmarshal(msg.Data, &cmd); err != nil {
			log.Printf("[Gateway] Failed to unmarshal command: %v", err)
			return
		}

		value, ok := s.sessions.Load(cmd.DeviceID)
		if !ok {
			log.Printf("[Gateway] Device not connected: %s", cmd.DeviceID)
			return
		}

		session := value.(*Session)
		if session.Adapter == nil {
			log.Printf("[Gateway] Protocol not determined for: %s", cmd.DeviceID)
			return
		}

		data, err := session.Adapter.Encode(protocol.StandardCommand{
			Type:   cmd.Type,
			Params: cmd.Params,
		})
		if err != nil {
			log.Printf("[Gateway] Failed to encode command: %v", err)
			return
		}

		_, err = session.Conn.Write(data)
		if err != nil {
			log.Printf("[Gateway] Failed to send command: %v", err)
			return
		}

		log.Printf("[Gateway] Command sent to %s: %s", cmd.DeviceID, cmd.Type)
	})

	if err != nil {
		log.Printf("[Gateway] Failed to subscribe to downlink: %v", err)
		return
	}

	<-s.ctx.Done()
	sub.Unsubscribe()
}
