package httpserver

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
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
