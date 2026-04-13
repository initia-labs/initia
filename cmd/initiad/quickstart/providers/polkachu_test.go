package providers

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFetchLatestHeight(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/abci_info", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"result":{"response":{"last_block_height":"12345"}}}`))
	}))
	defer srv.Close()

	origClient := HTTPClient
	HTTPClient = srv.Client()
	defer func() { HTTPClient = origClient }()

	height, err := FetchLatestHeight(srv.URL)
	require.NoError(t, err)
	require.Equal(t, int64(12345), height)
}

func TestFetchBlockHash(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/block", r.URL.Path)
		require.Equal(t, "10000", r.URL.Query().Get("height"))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"result":{"block_id":{"hash":"ABCDEF123456"}}}`))
	}))
	defer srv.Close()

	origClient := HTTPClient
	HTTPClient = srv.Client()
	defer func() { HTTPClient = origClient }()

	hash, err := FetchBlockHash(srv.URL, 10000)
	require.NoError(t, err)
	require.Equal(t, "ABCDEF123456", hash)
}

func TestFetchStateSyncPeer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"polkachu_peer":"abc123@1.2.3.4:26656"}`))
	}))
	defer srv.Close()

	origClient := HTTPClient
	HTTPClient = srv.Client()
	defer func() { HTTPClient = origClient }()

	peer, err := FetchStateSyncPeer(srv.URL)
	require.NoError(t, err)
	require.Equal(t, "abc123@1.2.3.4:26656", peer)
}

func TestDownloadFile(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("hello world"))
	}))
	defer srv.Close()

	origClient := HTTPClient
	HTTPClient = srv.Client()
	defer func() { HTTPClient = origClient }()

	destPath := filepath.Join(t.TempDir(), "downloaded.txt")

	err := DownloadFile(srv.URL+"/file", destPath)
	require.NoError(t, err)

	data, err := os.ReadFile(destPath)
	require.NoError(t, err)
	require.Equal(t, "hello world", string(data))
}

func TestDownloadFileAtomicOnFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	origClient := HTTPClient
	HTTPClient = srv.Client()
	defer func() { HTTPClient = origClient }()

	destPath := filepath.Join(t.TempDir(), "should_not_exist.txt")

	err := DownloadFile(srv.URL+"/file", destPath)
	require.Error(t, err)

	_, statErr := os.Stat(destPath)
	require.True(t, os.IsNotExist(statErr), "dest file should not exist after failed download")
}
