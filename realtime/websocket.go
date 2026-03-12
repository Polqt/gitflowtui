package realtime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Polqt/gitflowtui/tui"
	"github.com/gorilla/websocket"
)

// Server broadcasts TUI events to websocket clients.
type Server struct {
	addr string
	path string

	mu       sync.RWMutex
	clients  map[*websocket.Conn]struct{}
	upgrader websocket.Upgrader
	httpSrv  *http.Server
}

// NewServer creates a websocket broadcaster bound to addr/path.
func NewServer(addr, path string) (*Server, error) {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return nil, errors.New("websocket address cannot be empty")
	}
	path = normalizePath(path)
	return &Server{
		addr:    addr,
		path:    path,
		clients: make(map[*websocket.Conn]struct{}),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(_ *http.Request) bool { return true },
		},
	}, nil
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
		Addr:              s.addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		if err := s.httpSrv.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			// Non-fatal for the TUI process; websocket streaming can fail independently.
			log.Printf("websocket server stopped: %v", err)
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
		_ = conn.Close()
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
	clients := make([]*websocket.Conn, 0, len(s.clients))
	for conn := range s.clients {
		clients = append(clients, conn)
	}
	s.mu.RUnlock()

	for _, conn := range clients {
		_ = conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
		if err := conn.WriteMessage(websocket.TextMessage, payload); err != nil {
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
		return
	}

	s.mu.Lock()
	s.clients[conn] = struct{}{}
	s.mu.Unlock()

	defer s.removeClient(conn)
	conn.SetReadLimit(1024)

	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			return
		}
	}
}

func (s *Server) removeClient(conn *websocket.Conn) {
	s.mu.Lock()
	delete(s.clients, conn)
	s.mu.Unlock()
	_ = conn.Close()
}

func normalizePath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return "/ws"
	}
	if !strings.HasPrefix(path, "/") {
		return "/" + path
	}
	return path
}
