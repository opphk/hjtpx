package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

type HTTP2Server struct {
	server           *http.Server
	enableHTTP2      bool
	enableTLS        bool
	tlsCertFile      string
	tlsKeyFile       string
	readTimeout      time.Duration
	writeTimeout     time.Duration
	idleTimeout      time.Duration
	maxHeaderBytes   int
	keepAliveTimeout time.Duration
	shutdownTimeout  time.Duration
}

type ServerMetrics struct {
	RequestsTotal      int64
	RequestsSuccess    int64
	RequestsFailed     int64
	BytesSent          int64
	BytesReceived      int64
	ActiveConnections  int64
	TotalConnections   int64
	Uptime             time.Duration
	StartTime          time.Time
}

type RequestMetrics struct {
	Path         string
	Method       string
	Duration     time.Duration
	StatusCode   int
	BytesSent    int64
	Timestamp    time.Time
}

var (
	serverMetrics = &ServerMetrics{}
	metricsMu     sync.RWMutex
	requestBuffer = make(chan *RequestMetrics, 10000)
)

func NewHTTP2Server() *HTTP2Server {
	return &HTTP2Server{
		enableHTTP2:      true,
		enableTLS:        false,
		readTimeout:      15 * time.Second,
		writeTimeout:     15 * time.Second,
		idleTimeout:      60 * time.Second,
		maxHeaderBytes:   1 << 20,
		keepAliveTimeout: 30 * time.Second,
		shutdownTimeout:  30 * time.Second,
	}
}

func (s *HTTP2Server) SetTLS(certFile, keyFile string) {
	s.tlsCertFile = certFile
	s.tlsKeyFile = keyFile
	s.enableTLS = true
}

func (s *HTTP2Server) SetHTTP2Enabled(enabled bool) {
	s.enableHTTP2 = enabled
}

func (s *HTTP2Server) SetTimeouts(read, write, idle, keepAlive, shutdown time.Duration) {
	s.readTimeout = read
	s.writeTimeout = write
	s.idleTimeout = idle
	s.keepAliveTimeout = keepAlive
	s.shutdownTimeout = shutdown
}

func (s *HTTP2Server) SetMaxHeaderBytes(max int) {
	s.maxHeaderBytes = max
}

func (s *HTTP2Server) Start(handler http.Handler) error {
	serverMetrics.StartTime = time.Now()

	s.server = &http.Server{
		Addr:           ":8080",
		Handler:        s.wrapHandler(handler),
		ReadTimeout:    s.readTimeout,
		WriteTimeout:   s.writeTimeout,
		IdleTimeout:    s.idleTimeout,
		MaxHeaderBytes: s.maxHeaderBytes,
	}

	if s.enableHTTP2 {
		s.server.Handler = s.addHTTP2Support(s.server.Handler)
	}

	go s.processMetrics()
	go s.collectSystemMetrics()

	var err error
	if s.enableTLS {
		tlsConfig, err := s.createTLSConfig()
		if err != nil {
			return fmt.Errorf("failed to create TLS config: %w", err)
		}
		s.server.TLSConfig = tlsConfig

		log.Println("Starting HTTP/2 server with TLS...")
		err = s.server.ListenAndServeTLS(s.tlsCertFile, s.tlsKeyFile)
	} else {
		log.Println("Starting HTTP/2 server (via ALPN)...")
		err = s.server.ListenAndServe()
	}

	if err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}

func (s *HTTP2Server) createTLSConfig() (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(s.tlsCertFile, s.tlsKeyFile)
	if err != nil {
		return nil, err
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion: tls.VersionTLS12,
		CurvePreferences: []tls.CurveID{
			tls.CurveP256,
			tls.X25519,
		},
		PreferServerCipherSuites: true,
		NextProtos: []string{
			"h2",
			"http/1.1",
		},
	}

	if s.enableHTTP2 {
		tlsConfig.NextProtos = []string{"h2"}
	}

	return tlsConfig, nil
}

func (s *HTTP2Server) addHTTP2Support(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&serverMetrics.TotalConnections, 1)
		atomic.AddInt64(&serverMetrics.ActiveConnections, 1)
		defer atomic.AddInt64(&serverMetrics.ActiveConnections, -1)

		start := time.Now()
		rw := &responseWriter{
			ResponseWriter: w,
			statusCode:    http.StatusOK,
		}

		handler.ServeHTTP(rw, r)

		duration := time.Since(start)

		metrics := &RequestMetrics{
			Path:       r.URL.Path,
			Method:     r.Method,
			Duration:   duration,
			StatusCode: rw.statusCode,
			BytesSent:  int64(rw.bytesWritten),
			Timestamp:  start,
		}

		select {
		case requestBuffer <- metrics:
		default:
		}

		atomic.AddInt64(&serverMetrics.RequestsTotal, 1)
		if rw.statusCode >= 200 && rw.statusCode < 400 {
			atomic.AddInt64(&serverMetrics.RequestsSuccess, 1)
		} else {
			atomic.AddInt64(&serverMetrics.RequestsFailed, 1)
		}
		atomic.AddInt64(&serverMetrics.BytesSent, int64(rw.bytesWritten))
	})
}

func (s *HTTP2Server) wrapHandler(handler http.Handler) http.Handler {
	return s.addHTTP2Support(handler)
}

type responseWriter struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int64
	wroteHeader  bool
}

func (rw *responseWriter) WriteHeader(code int) {
	if !rw.wroteHeader {
		rw.statusCode = code
		rw.wroteHeader = true
	}
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.wroteHeader {
		rw.WriteHeader(http.StatusOK)
	}
	n, err := rw.ResponseWriter.Write(b)
	rw.bytesWritten += int64(n)
	return n, err
}

func (s *HTTP2Server) processMetrics() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.cleanupOldMetrics()
		case metrics := <-requestBuffer:
			if metrics == nil {
				return
			}
		}
	}
}

func (s *HTTP2Server) cleanupOldMetrics() {
	metricsMu.Lock()
	defer metricsMu.Unlock()

	cutoff := time.Now().Add(-1 * time.Hour)
	serverMetrics.Uptime = time.Since(serverMetrics.StartTime)
	_ = cutoff
}

func (s *HTTP2Server) collectSystemMetrics() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	var lastCPUTime time.Duration
	var lastWallTime time.Time

	for {
		select {
		case <-ticker.C:
			var m runtime.MemStats
			runtime.ReadMemStats(&m)

			now := time.Now()
			cpuTime := time.Duration(runtime.NumGoroutine() * 1000000)
			wallElapsed := now.Sub(lastWallTime)

			if wallElapsed > 0 && lastCPUTime > 0 {
				cpuUsage := float64(cpuTime-lastCPUTime) / float64(wallElapsed) * 100
				_ = cpuUsage
			}

			lastCPUTime = cpuTime
			lastWallTime = now
			_ = m
		}
	}
}

func (s *HTTP2Server) Shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
	defer cancel()

	if s.server == nil {
		return nil
	}

	log.Println("Shutting down server...")
	return s.server.Shutdown(ctx)
}

func GetServerMetrics() *ServerMetrics {
	metricsMu.RLock()
	defer metricsMu.RUnlock()

	return &ServerMetrics{
		RequestsTotal:     atomic.LoadInt64(&serverMetrics.RequestsTotal),
		RequestsSuccess:   atomic.LoadInt64(&serverMetrics.RequestsSuccess),
		RequestsFailed:    atomic.LoadInt64(&serverMetrics.RequestsFailed),
		BytesSent:        atomic.LoadInt64(&serverMetrics.BytesSent),
		BytesReceived:    atomic.LoadInt64(&serverMetrics.BytesReceived),
		ActiveConnections: atomic.LoadInt64(&serverMetrics.ActiveConnections),
		TotalConnections:  atomic.LoadInt64(&serverMetrics.TotalConnections),
		Uptime:            time.Since(serverMetrics.StartTime),
		StartTime:         serverMetrics.StartTime,
	}
}

func GetRequestMetricsBuffer() chan *RequestMetrics {
	return requestBuffer
}

type GracefulShutdown struct {
	shutdownCh chan struct{}
	doneCh     chan struct{}
	wg         sync.WaitGroup
}

func NewGracefulShutdown() *GracefulShutdown {
	return &GracefulShutdown{
		shutdownCh: make(chan struct{}),
		doneCh:     make(chan struct{}),
	}
}

func (gs *GracefulShutdown) WaitForShutdown() {
	<-gs.shutdownCh
	gs.doneCh <- struct{}{}
}

func (gs *GracefulShutdown) NotifyShutdown() {
	close(gs.shutdownCh)
}

func (gs *GracefulShutdown) Done() {
	<-gs.doneCh
}

func (gs *GracefulShutdown) Add(delta int) {
	gs.wg.Add(delta)
}

func (gs *GracefulShutdown) Donewg() {
	gs.wg.Done()
}

func (gs *GracefulShutdown) Wait() {
	gs.wg.Wait()
}

func SetupSignalHandler(server *HTTP2Server) *GracefulShutdown {
	gs := NewGracefulShutdown()

	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, os.Interrupt, os.Kill)
		<-quit

		gs.NotifyShutdown()

		if err := server.Shutdown(); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}
	}()

	return gs
}

type ConnectionPoolConfig struct {
	MaxIdleConns        int
	MaxOpenConns       int
	ConnMaxLifetime    time.Duration
	ConnMaxIdleTime    time.Duration
	ConnRevalidate     bool
}

type OptimizedTransport struct {
	http.RoundTripper
	poolConfig *ConnectionPoolConfig
}

func NewOptimizedTransport() *OptimizedTransport {
	return &OptimizedTransport{
		RoundTripper: &http.Transport{
			MaxIdleConns:        1000,
			MaxIdleConnsPerHost: 100,
			MaxConnsPerHost:     1000,
			IdleConnTimeout:     90 * time.Second,
			TLSHandshakeTimeout: 10 * time.Second,
			DisableKeepAlives:   false,
			DisableCompression:  false,
			WriteBufferSize:    32 * 1024,
			ReadBufferSize:     32 * 1024,
		},
		poolConfig: &ConnectionPoolConfig{
			MaxIdleConns:     1000,
			MaxOpenConns:    1000,
			ConnMaxLifetime:  5 * time.Minute,
			ConnMaxIdleTime:  2 * time.Minute,
		},
	}
}

func (t *OptimizedTransport) SetPoolConfig(config *ConnectionPoolConfig) {
	t.poolConfig = config

	if transport, ok := t.RoundTripper.(*http.Transport); ok {
		transport.MaxIdleConns = config.MaxIdleConns
		transport.MaxIdleConnsPerHost = config.MaxIdleConns / 10
		if transport.MaxIdleConnsPerHost < 10 {
			transport.MaxIdleConnsPerHost = 10
		}
		transport.IdleConnTimeout = config.ConnMaxIdleTime
	}
}

func (t *OptimizedTransport) GetPoolStats() map[string]interface{} {
	if transport, ok := t.RoundTripper.(*http.Transport); ok {
		return map[string]interface{}{
			"max_idle_conns":          transport.MaxIdleConns,
			"max_idle_conns_per_host": transport.MaxIdleConnsPerHost,
			"idle_conn_timeout":       transport.IdleConnTimeout.String(),
		}
	}
	return nil
}

func CreateSelfSignedCert(certFile, keyFile string) error {
	return fmt.Errorf("use proper certificates in production")
}
