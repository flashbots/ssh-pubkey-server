package httpserver

import (
	"bytes"
	"net/http"
	"time"

	"github.com/flashbots/ssh-pubkey-server/metrics"
)

// handleGetPubkey serves every currently-available pubkey, newline-joined.
// Files are read on each request, and any path that is missing or not yet
// readable is skipped — so the response grows as keys become available (e.g. a
// key behind an encrypted disk that is unlocked later) without a restart. When
// no key is available yet, it responds 503.
func (s *Server) handleGetPubkey(w http.ResponseWriter, r *http.Request) {
	m := s.metricsSrv.Float64Histogram(
		"request_duration_api",
		"API request handling duration",
		metrics.UomMicroseconds,
		metrics.BucketsRequestDuration...,
	)
	defer func(start time.Time) {
		m.Record(r.Context(), float64(time.Since(start).Microseconds()))
	}(time.Now())

	var keys [][]byte
	for _, path := range s.cfg.SSHPubkeyPaths {
		pubkey, err := readAndFormatPubkey(path)
		if err != nil {
			s.log.Debug("pubkey not available, skipping", "path", path, "err", err)
			continue
		}
		if pubkey != nil {
			keys = append(keys, pubkey)
		}
	}

	if len(keys) == 0 {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	if _, err := w.Write(bytes.Join(keys, []byte("\n"))); err != nil {
		s.log.Error("could not serve pubkey", "err", err)
	}
}

func (s *Server) handleLivenessCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleReadinessCheck(w http.ResponseWriter, r *http.Request) {
	if !s.isReady.Load() {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleDrain(w http.ResponseWriter, r *http.Request) {
	if wasReady := s.isReady.Swap(false); !wasReady {
		return
	}
	// l := logutils.ZapFromRequest(r)
	s.log.Info("Server marked as not ready")
	time.Sleep(s.cfg.DrainDuration) // Give LB enough time to detect us not ready
}

func (s *Server) handleUndrain(w http.ResponseWriter, r *http.Request) {
	if wasReady := s.isReady.Swap(true); wasReady {
		return
	}
	// l := logutils.ZapFromRequest(r)
	s.log.Info("Server marked as ready")
}
