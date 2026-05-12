package realtime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Polqt/gitflowtui/tui"
	"github.com/gorilla/websocket"
)

const defaultWSPath = "/ws"

// Server broadcasts TUI events to websocket clients.
type Server struct {
	addr string
	path string

	mu       sync.RWMutex
	clients  map[*clientConn]struct{}
	upgrader websocket.Upgrader
	httpSrv  *http.Server
	logger   *log.Logger
}

type clientConn struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

// NewServer creates a websocket broadcaster bound to addr/path.
func NewServer(addr, path string) (*Server, error) {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return nil, errors.New("websocket address cannot be empty")
	}
	path, err := normalizePath(path)
	if err != nil {
		return nil, err
	}
	return &Server{
		addr:    addr,
		path:    path,
		clients: make(map[*clientConn]struct{}),
		logger:  log.New(io.Discard, "", 0),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(_ *http.Request) bool { return true },
		},
	}, nil
}

// SetLogger configures startup, connection, and shutdown logging.
func (s *Server) SetLogger(logger *log.Logger) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if logger == nil {
		s.logger = log.New(io.Discard, "", 0)
		return
	}
	s.logger = logger
}

// Start launches the websocket server in the background.
func (s *Server) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc(s.path, s.handleWS)

	listener, err := new(net.ListenConfig).Listen(context.Background(), "tcp", s.addr)
	if err != nil {
		return fmt.Errorf("listen websocket server: %w", err)
	}

	s.httpSrv = &http.Server{
		Addr:              listener.Addr().String(),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	s.addr = listener.Addr().String()
	s.logf("realtime websocket listening on %s", s.URL())

	go func() {
		if err := s.httpSrv.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.logf("realtime websocket stopped unexpectedly: %v", err)
		}
	}()
	return nil
}

// Shutdown stops the websocket server and disconnects active clients.
func (s *Server) Shutdown(ctx context.Context) error {
	var err error
	if s.httpSrv != nil {
		err = s.httpSrv.Shutdown(ctx)
	}

	s.mu.Lock()
	for conn := range s.clients {
		_ = conn.conn.Close()
		delete(s.clients, conn)
	}
	s.mu.Unlock()

	return err
}

// Publish broadcasts a TUI realtime event to all connected clients.
func (s *Server) Publish(event tui.RealtimeEvent) {
	payload, err := json.Marshal(event)
	if err != nil {
		return
	}

	s.mu.RLock()
	clients := make([]*clientConn, 0, len(s.clients))
	for conn := range s.clients {
		clients = append(clients, conn)
	}
	s.mu.RUnlock()

	for _, conn := range clients {
		conn.mu.Lock()
		_ = conn.conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
		err := conn.conn.WriteMessage(websocket.TextMessage, payload)
		conn.mu.Unlock()
		if err != nil {
			s.removeClient(conn)
		}
	}
}

// URL returns the ws endpoint for clients.
func (s *Server) URL() string {
	return "ws://" + s.addr + s.path
}

func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logf("realtime websocket upgrade failed: %v", err)
		return
	}

	client := &clientConn{conn: conn}
	s.mu.Lock()
	s.clients[client] = struct{}{}
	s.mu.Unlock()
	s.logf("realtime websocket client connected: %s", r.RemoteAddr)

	defer s.removeClient(client)
	conn.SetReadLimit(1024)
	_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	})

	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			return
		}
	}
}

func (s *Server) removeClient(conn *clientConn) {
	s.mu.Lock()
	delete(s.clients, conn)
	s.mu.Unlock()
	_ = conn.conn.Close()
	s.logf("realtime websocket client disconnected")
}

func (s *Server) logf(format string, args ...any) {
	s.mu.RLock()
	logger := s.logger
	s.mu.RUnlock()
	if logger != nil {
		logger.Printf(format, args...)
	}
}

func normalizePath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return defaultWSPath, nil
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if strings.ContainsAny(path, " \t\r\n") || strings.Contains(path, ":") {
		return "", fmt.Errorf("invalid websocket path %q: use a URL path like ws or /ws", path)
	}
	return path, nil
}
