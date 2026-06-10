package httpserver

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/flashbots/ssh-pubkey-server/common"
	"github.com/stretchr/testify/require"
)

func getTestLogger() *slog.Logger {
	return common.SetupLogger(&common.LoggingOpts{
		Debug:   true,
		JSON:    false,
		Service: "test",
		Version: "test",
	})
}

func Test_Handlers_Healthcheck_Drain_Undrain(t *testing.T) {
	const (
		latency    = 200 * time.Millisecond
		listenAddr = ":8080"
	)

	//nolint: exhaustruct
	s, err := New(&HTTPServerConfig{
		DrainDuration:  latency,
		ListenAddr:     listenAddr,
		Log:            getTestLogger(),
		SSHPubkeyPaths: []string{"./test_key.pub", "./test_key.pub"},
	})
	require.NoError(t, err)

	{ // Check pubkey
		req := httptest.NewRequest(http.MethodGet, "http://localhost"+listenAddr+"/pubkey", nil) //nolint:goconst,nolintlint
		w := httptest.NewRecorder()
		s.handleGetPubkey(w, req)
		resp := w.Result()
		defer resp.Body.Close()
		data, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		expectedKey := []byte("ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIFGGVd5nQewq0hETk2Tr/P7OZxTW/4aftdfh9/cAe7FC")
		expectedOutput := append(expectedKey, '\n')
		expectedOutput = append(expectedOutput, expectedKey...)
		require.Equal(t, expectedOutput, data)
	}

	{ // Check health
		req := httptest.NewRequest(http.MethodGet, "http://localhost"+listenAddr+"/readyz", nil) //nolint:goconst,nolintlint
		w := httptest.NewRecorder()
		s.handleReadinessCheck(w, req)
		resp := w.Result()
		defer resp.Body.Close()
		_, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode, "Healthcheck must return `Ok` before draining")
	}

	{ // Drain
		req := httptest.NewRequest(http.MethodGet, "http://localhost"+listenAddr+"/drain", nil)
		w := httptest.NewRecorder()
		start := time.Now()
		s.handleDrain(w, req)
		duration := time.Since(start)
		resp := w.Result()
		defer resp.Body.Close()
		_, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode, "Must return `Ok` for calls to `/drain`")
		require.GreaterOrEqual(t, duration, latency, "Must wait long enough during draining")
	}

	{ // Check health
		req := httptest.NewRequest(http.MethodGet, "http://localhost"+listenAddr+"/readyz", nil)
		w := httptest.NewRecorder()
		s.handleReadinessCheck(w, req)
		resp := w.Result()
		defer resp.Body.Close()
		_, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		require.Equal(t, http.StatusServiceUnavailable, resp.StatusCode, "Healthcheck must return `Service Unavailable` after draining")
	}

	{ // Undrain
		req := httptest.NewRequest(http.MethodGet, "http://localhost"+listenAddr+"/undrain", nil)
		w := httptest.NewRecorder()
		s.handleUndrain(w, req)
		resp := w.Result()
		defer resp.Body.Close()
		_, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode, "Must return `Ok` for calls to `/undrain`")
		time.Sleep(latency)
	}

	{ // Check health
		req := httptest.NewRequest(http.MethodGet, "http://localhost"+listenAddr+"/readyz", nil)
		w := httptest.NewRecorder()
		s.handleReadinessCheck(w, req)
		resp := w.Result()
		defer resp.Body.Close()
		_, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode, "Healthcheck must return `Ok` after undraining")
	}
}

func Test_Handlers_Pubkey_LazyAvailability(t *testing.T) {
	dir := t.TempDir()
	pathA := filepath.Join(dir, "a.pub")
	pathB := filepath.Join(dir, "b.pub")

	// The host field is dropped by readAndFormatPubkey, so the served output is
	// just "<type> <key>".
	require.NoError(t, os.WriteFile(pathA, []byte("ssh-ed25519 AAAAKEYA comment"), 0o600))
	expectedA := []byte("ssh-ed25519 AAAAKEYA")
	expectedB := []byte("ssh-ed25519 AAAAKEYB")

	//nolint: exhaustruct
	s, err := New(&HTTPServerConfig{
		ListenAddr:     ":8080",
		Log:            getTestLogger(),
		SSHPubkeyPaths: []string{pathA, pathB}, // pathB does not exist yet
	})
	require.NoError(t, err)

	get := func() (int, []byte) {
		req := httptest.NewRequest(http.MethodGet, "http://localhost/pubkey", nil)
		w := httptest.NewRecorder()
		s.handleGetPubkey(w, req)
		resp := w.Result()
		defer resp.Body.Close()
		body, readErr := io.ReadAll(resp.Body)
		require.NoError(t, readErr)
		return resp.StatusCode, body
	}

	// Only the first key exists yet: /pubkey serves the available subset.
	code, body := get()
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, expectedA, body, "/pubkey must return only the available key")

	// The second key appears later (e.g. after the disk is unlocked): served
	// with no restart, thanks to per-request reads.
	require.NoError(t, os.WriteFile(pathB, []byte("ssh-ed25519 AAAAKEYB comment"), 0o600))
	code, body = get()
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, []byte(string(expectedA)+"\n"+string(expectedB)), body, "/pubkey must return both keys once available")

	// A half-written (empty) file is skipped rather than panicking.
	require.NoError(t, os.WriteFile(pathA, []byte(""), 0o600))
	code, body = get()
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, expectedB, body, "/pubkey must skip an empty/malformed key file")

	// When no key is available yet, /pubkey reports not-ready.
	require.NoError(t, os.Remove(pathA))
	require.NoError(t, os.Remove(pathB))
	code, _ = get()
	require.Equal(t, http.StatusServiceUnavailable, code, "/pubkey must return 503 when no key is available")
}
