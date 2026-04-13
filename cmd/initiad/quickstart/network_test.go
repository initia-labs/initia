package quickstart

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/initia-labs/initia/cmd/initiad/quickstart/providers"
)

func TestBuildAddrbookFromRPC(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/net_info":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"result":{"peers":[{"node_info":{"id":"abc123def456","listen_addr":"tcp://0.0.0.0:26656"},"remote_ip":"1.2.3.4"},{"node_info":{"id":"def789abc012","listen_addr":"tcp://0.0.0.0:26656"},"remote_ip":"5.6.7.8"}]}}`))
		case "/status":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"result":{"node_info":{"id":"src123node456","listen_addr":"tcp://0.0.0.0:26656"}}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	origClient := providers.HTTPClient
	providers.HTTPClient = srv.Client()
	defer func() { providers.HTTPClient = origClient }()

	tmpDir := t.TempDir()
	destPath := filepath.Join(tmpDir, "addrbook.json")

	err := buildAddrbookFromRPC(srv.URL, destPath)
	require.NoError(t, err)

	info, err := os.Stat(destPath)
	require.NoError(t, err)
	require.Greater(t, info.Size(), int64(0), "addrbook.json should be non-empty")
}

func TestParsePortFromListenAddr(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  uint16
		expectErr bool
	}{
		{
			name:     "tcp scheme with wildcard host",
			input:    "tcp://0.0.0.0:26656",
			expected: 26656,
		},
		{
			name:     "no scheme with wildcard host",
			input:    "0.0.0.0:26656",
			expected: 26656,
		},
		{
			name:     "tcp scheme with loopback",
			input:    "tcp://127.0.0.1:33756",
			expected: 33756,
		},
		{
			name:      "invalid format",
			input:     "invalid",
			expectErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			port, err := parsePortFromListenAddr(tc.input)
			if tc.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expected, port)
			}
		})
	}
}
