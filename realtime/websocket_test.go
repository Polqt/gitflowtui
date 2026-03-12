package realtime_test

import (
	"strings"
	"testing"

	"github.com/Polqt/gitflowtui/realtime"
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
