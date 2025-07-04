package httpserver

import (
	"net/http"
	"time"

	"github.com/flashbots/ssh-pubkey-server/metrics"
)

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

	_, err := w.Write(s.sshPubkeys)
	if err != nil {
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
