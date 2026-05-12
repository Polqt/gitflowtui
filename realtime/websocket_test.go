package realtime_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/Polqt/gitflowtui/realtime"
	"github.com/Polqt/gitflowtui/tui"
)

func TestNewServer_NormalizesPathInURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "empty defaults", in: "", want: "/ws"},
		{name: "spaces defaults", in: "   ", want: "/ws"},
		{name: "already rooted", in: "/events", want: "/events"},
		{name: "without slash", in: "events", want: "/events"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server, err := realtime.NewServer("127.0.0.1:7777", tt.in)
			if err != nil {
				t.Fatalf("NewServer returned error: %v", err)
			}

			if got := server.URL(); !strings.HasSuffix(got, tt.want) {
				t.Fatalf("URL() = %q, expected suffix %q", got, tt.want)
			}
		})
	}
}

func TestNewServer_RejectsInvalidPath(t *testing.T) {
	t.Parallel()

	_, err := realtime.NewServer("127.0.0.1:7777", "/C:/Program Files/Git/ws")
	if err == nil {
		t.Fatal("NewServer returned nil error for invalid path")
	}
}

func TestServerBroadcastsToConnectedClient(t *testing.T) {
	t.Parallel()

	server, err := realtime.NewServer("127.0.0.1:0", "/ws")
	if err != nil {
		t.Fatalf("NewServer returned error: %v", err)
	}
	if err := server.Start(); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			t.Fatalf("Shutdown returned error: %v", err)
		}
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	events, errs := realtime.Client{
		URL:        server.URL(),
		MinBackoff: 10 * time.Millisecond,
		MaxBackoff: 50 * time.Millisecond,
	}.Listen(ctx)

	want := tui.RealtimeEvent{
		Type:      "status",
		Timestamp: time.Now().UTC(),
		RepoRoot:  "repo",
		Branch:    "feature/ws",
		Status: &tui.RealtimeStatusMsg{
			Message: "connected",
			Error:   false,
		},
	}

	deadline := time.After(2 * time.Second)
	for {
		server.Publish(want)
		select {
		case got := <-events:
			if got.Type != want.Type ||
				got.Branch != want.Branch ||
				got.Status == nil ||
				got.Status.Message != want.Status.Message {
				t.Fatalf("received event = %#v, want %#v", got, want)
			}
			return
		case err := <-errs:
			if err != nil {
				t.Logf("client reconnect event: %v", err)
			}
		case <-time.After(25 * time.Millisecond):
		case <-deadline:
			t.Fatal("timed out waiting for websocket broadcast")
		}
	}
}
