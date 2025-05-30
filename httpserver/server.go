package httpserver

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/flashbots/go-utils/httplogger"
	"github.com/flashbots/ssh-pubkey-server/common"
	"github.com/flashbots/ssh-pubkey-server/metrics"
	chi "github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/atomic"
)

type HTTPServerConfig struct {
	ListenAddr  string
	MetricsAddr string
	EnablePprof bool
	Log         *slog.Logger

	DrainDuration            time.Duration
	GracefulShutdownDuration time.Duration
	ReadTimeout              time.Duration
	WriteTimeout             time.Duration

	SSHPubkeyPath string
}

type Server struct {
	cfg     *HTTPServerConfig
	isReady atomic.Bool
	log     *slog.Logger

	sshPubkey []byte

	srv        *http.Server
	metricsSrv *metrics.MetricsServer
}

func New(cfg *HTTPServerConfig) (srv *Server, err error) {
	metricsSrv, err := metrics.New(common.PackageName, cfg.MetricsAddr)
	if err != nil {
		return nil, err
	}

	sshPubkey, err := os.ReadFile(cfg.SSHPubkeyPath)
	if err != nil {
		return nil, err
	}
	// pubkey is in the form <type> <key> <host>. we want to drop the host
	sshPubkey = bytes.Join(bytes.Fields(sshPubkey)[0:2], []byte(" "))

	srv = &Server{
		cfg:        cfg,
		log:        cfg.Log,
		sshPubkey:  sshPubkey,
		srv:        nil,
		metricsSrv: metricsSrv,
	}
	srv.isReady.Swap(true)

	mux := chi.NewRouter()
	mux.With(srv.httpLogger).Get("/pubkey", srv.handleGetPubkey) // Never serve at `/` (root) path
	mux.With(srv.httpLogger).Get("/livez", srv.handleLivenessCheck)
	mux.With(srv.httpLogger).Get("/readyz", srv.handleReadinessCheck)
	mux.With(srv.httpLogger).Get("/drain", srv.handleDrain)
	mux.With(srv.httpLogger).Get("/undrain", srv.handleUndrain)

	if cfg.EnablePprof {
		srv.log.Info("pprof API enabled")
		mux.Mount("/debug", middleware.Profiler())
	}

	srv.srv = &http.Server{
		Addr:         cfg.ListenAddr,
		Handler:      mux,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	}

	return srv, nil
}

func (s *Server) httpLogger(next http.Handler) http.Handler {
	return httplogger.LoggingMiddlewareSlog(s.log, next)
}

func (s *Server) RunInBackground() {
	// metrics
	if s.cfg.MetricsAddr != "" {
		go func() {
			s.log.With("metricsAddress", s.cfg.MetricsAddr).Info("Starting metrics server")
			err := s.metricsSrv.ListenAndServe()
			if err != nil && !errors.Is(err, http.ErrServerClosed) {
				s.log.Error("HTTP server failed", "err", err)
			}
		}()
	}

	// api
	go func() {
		s.log.Info("Starting HTTP server", "listenAddress", s.cfg.ListenAddr)
		if err := s.srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.log.Error("HTTP server failed", "err", err)
		}
	}()
}

func (s *Server) Shutdown() {
	// api
	ctx, cancel := context.WithTimeout(context.Background(), s.cfg.GracefulShutdownDuration)
	defer cancel()
	if err := s.srv.Shutdown(ctx); err != nil {
		s.log.Error("Graceful HTTP server shutdown failed", "err", err)
	} else {
		s.log.Info("HTTP server gracefully stopped")
	}

	// metrics
	if len(s.cfg.MetricsAddr) != 0 {
		ctx, cancel := context.WithTimeout(context.Background(), s.cfg.GracefulShutdownDuration)
		defer cancel()

		if err := s.metricsSrv.Shutdown(ctx); err != nil {
			s.log.Error("Graceful metrics server shutdown failed", "err", err)
		} else {
			s.log.Info("Metrics server gracefully stopped")
		}
	}
}
