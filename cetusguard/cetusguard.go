package cetusguard

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/hectorm/cetusguard/internal/logger"
	"github.com/hectorm/cetusguard/internal/utils/middleware"
)

const (
	minTlsVersion = tls.VersionTLS12
)

const (
	mediaTypeRawStream         = "application/vnd.docker.raw-stream"
	mediaTypeMultiplexedStream = "application/vnd.docker.multiplexed-stream"
)

type Backend struct {
	Addr      string
	TlsCacert string
	TlsCert   string
	TlsKey    string
}

type Frontend struct {
	Addr      []string
	TlsCacert string
	TlsCert   string
	TlsKey    string
}

type Server struct {
	Backend  *Backend
	Frontend *Frontend
	Rules    []Rule

	backendProto      string
	backendHost       string
	backendTlsConfig  *tls.Config
	backendHttpClient *http.Client

	frontendNetListeners []net.Listener
	frontendTlsConfig    *tls.Config
	frontendHttpServer   *http.Server

	runningState int32
	mu           sync.Mutex
}

func (cg *Server) Start(ready chan<- any) error {
	cg.mu.Lock()
	var unlockOnce sync.Once
	defer unlockOnce.Do(cg.mu.Unlock)

	var closeOnce sync.Once
	defer closeOnce.Do(func() { close(ready) })

	if cg.IsRunning() {
		return errors.New("server is already running")
	}
	defer cg.setIsRunning(false)

	var err error
	cg.backendProto, cg.backendHost, err = parseAddr(cg.Backend.Addr)
	if err != nil {
		return err
	}

	cg.backendTlsConfig, err = clientTlsConfig(cg.Backend.TlsCacert, cg.Backend.TlsCert, cg.Backend.TlsKey)
	if err != nil {
		return err
	}

	backendDialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 90 * time.Second,
	}

	cg.backendHttpClient = &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig:       cg.backendTlsConfig,
			MaxIdleConns:          10,
			MaxIdleConnsPerHost:   10,
			TLSHandshakeTimeout:   10 * time.Second,
			IdleConnTimeout:       90 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			DialContext: func(ctx context.Context, _ string, _ string) (net.Conn, error) {
				return backendDialer.DialContext(ctx, cg.backendProto, cg.backendHost)
			},
		},
	}

	cg.frontendNetListeners = nil
	for _, addr := range cg.Frontend.Addr {
		proto, host, err := parseAddr(addr)
		if err != nil {
			return err
		}
		l, err := net.Listen(proto, host)
		if err != nil {
			return err
		}
		cg.frontendNetListeners = append(cg.frontendNetListeners, l)
	}
	defer func() {
		for _, l := range cg.frontendNetListeners {
			_ = l.Close()
		}
	}()

	cg.frontendTlsConfig, err = serverTlsConfig(cg.Frontend.TlsCacert, cg.Frontend.TlsCert, cg.Frontend.TlsKey)
	if err != nil {
		return err
	}

	cg.frontendHttpServer = &http.Server{
		TLSConfig:         cg.frontendTlsConfig,
		ReadTimeout:       120 * time.Minute,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      120 * time.Minute,
		IdleTimeout:       90 * time.Second,
		ErrorLog:          logger.LgrError(),
		Handler: http.HandlerFunc(func(wri http.ResponseWriter, req *http.Request) {
			if cg.validateRequest(req) {
				err := cg.handleValidRequest(wri, req)
				if err != nil {
					logger.Error(err)
				}
			} else {
				cg.handleInvalidRequest(wri, req)
			}
		}),
	}

	chErr := make(chan error, 1)

	go func() {
		chSig := make(chan os.Signal, 1)
		signal.Notify(chSig, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

		sig := <-chSig
		logger.Infof("%v signal received\n", sig)

		chErr <- cg.Stop()
	}()

	for _, l := range cg.frontendNetListeners {
		logger.Infof("serve on %s\n", l.Addr())
		go func(l net.Listener, srv *http.Server, tls *tls.Config) {
			var err error
			if tls != nil && l.Addr().Network() != "unix" {
				err = srv.ServeTLS(l, "", "")
			} else {
				err = srv.Serve(l)
			}
			if err != http.ErrServerClosed {
				chErr <- err
			}
		}(l, cg.frontendHttpServer, cg.frontendTlsConfig)
	}

	cg.setIsRunning(true)
	unlockOnce.Do(cg.mu.Unlock)
	closeOnce.Do(func() { close(ready) })

	return <-chErr
}

func (cg *Server) Stop() error {
	cg.mu.Lock()
	defer cg.mu.Unlock()

	if !cg.IsRunning() {
		return errors.New("server is not running")
	}
	defer cg.setIsRunning(false)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cg.backendHttpClient.CloseIdleConnections()
	err := cg.frontendHttpServer.Shutdown(ctx)

	logger.Infof("exit\n")
	return err
}

func (cg *Server) Addrs() ([]net.Addr, error) {
	if !cg.IsRunning() {
		return nil, errors.New("server is not running")
	}

	var addr []net.Addr
	for _, l := range cg.frontendNetListeners {
		addr = append(addr, l.Addr())
	}

	return addr, nil
}

func (cg *Server) IsRunning() bool {
	return atomic.LoadInt32(&cg.runningState) != 0
}

func (cg *Server) setIsRunning(running bool) {
	if running {
		atomic.StoreInt32(&cg.runningState, 1)
	} else {
		atomic.StoreInt32(&cg.runningState, 0)
	}
}

func (cg *Server) validateRequest(req *http.Request) bool {
	p := cleanPath(req.URL.Path)
	for _, rule := range cg.Rules {
		_, mOk := rule.Methods[req.Method]
		if mOk && rule.Pattern.MatchString(p) {
			return true
		}
	}
	return false
}

func (cg *Server) handleValidRequest(wri http.ResponseWriter, req *http.Request) error {
	logger.Debugf("allowed request: %s %s\n", req.Method, req.URL.Path)

	mWri := &middleware.ResponseWriter{ResponseWriter: wri}
	if f, ok := wri.(http.Flusher); ok {
		mWri.Flusher = f
	}

	newReq := req.Clone(req.Context())
	if cg.backendTlsConfig != nil {
		newReq.URL.Scheme = "https"
	} else {
		newReq.URL.Scheme = "http"
	}
	if cg.backendProto == "unix" {
		newReq.URL.Host = "localhost"
	} else {
		newReq.URL.Host = cg.backendHost
	}

	res, err := cg.backendHttpClient.Transport.RoundTrip(newReq)
	if errors.Is(err, context.Canceled) || errors.Is(err, syscall.ECONNREFUSED) {
		mWri.WriteHeader(http.StatusBadGateway)
		return nil
	} else if err != nil {
		mWri.WriteHeader(http.StatusBadGateway)
		return err
	}
	defer func() {
		_ = res.Body.Close()
	}()

	resMediaType := res.Header.Get("Content-Type")

	if resMediaType == mediaTypeRawStream || resMediaType == mediaTypeMultiplexedStream {
		logger.Debugf("stream response\n")

		// If the response is a stream, we need to disable the write deadline to prevent the connection from being closed
		rc := http.NewResponseController(wri)
		err = rc.SetWriteDeadline(time.Time{})
		if err != nil {
			return err
		}
	}

	if res.StatusCode == 101 {
		logger.Debugf("connection hijack\n")

		var upCloseOnce sync.Once
		var downCloseOnce sync.Once

		up, ok := res.Body.(io.ReadWriteCloser)
		if !ok {
			mWri.WriteHeader(http.StatusInternalServerError)
			return errors.New("body is not writable")
		}
		defer func() {
			upCloseOnce.Do(func() { _ = up.Close() })
		}()

		hj, ok := wri.(http.Hijacker)
		if !ok {
			mWri.WriteHeader(http.StatusInternalServerError)
			return errors.New("unable to hijack connection")
		}

		down, downRw, err := hj.Hijack()
		if err != nil {
			return err
		}
		defer func() {
			downCloseOnce.Do(func() { _ = down.Close() })
		}()

		_, err = downRw.Write([]byte(res.Proto + " " + res.Status + "\r\n"))
		if err != nil {
			return err
		}

		err = res.Header.Write(downRw)
		if err != nil {
			return err
		}

		_, err = downRw.Write([]byte("\r\n"))
		if err != nil {
			return err
		}

		err = downRw.Flush()
		if err != nil {
			return err
		}

		var wg sync.WaitGroup
		wg.Add(2)

		go func() {
			defer wg.Done()
			_, _ = io.Copy(up, down)
		}()

		go func() {
			defer wg.Done()
			_, _ = io.Copy(down, up)
			downCloseOnce.Do(func() { _ = down.Close() })
		}()

		wg.Wait()
	} else {
		for k, vv := range res.Header {
			for _, v := range vv {
				mWri.Header().Add(k, v)
			}
		}
		mWri.WriteHeader(res.StatusCode)

		if res.StatusCode >= 200 && res.StatusCode != 204 && res.StatusCode != 304 {
			_, err = io.Copy(mWri, res.Body)
			if errors.Is(err, context.Canceled) || errors.Is(err, syscall.EPIPE) || errors.Is(err, syscall.ECONNRESET) {
				return nil
			} else if err != nil {
				return err
			}
		}
	}

	return nil
}

func (cg *Server) handleInvalidRequest(wri http.ResponseWriter, req *http.Request) {
	logger.Warningf("denied request: %s %s\n", req.Method, req.URL.Path)

	wri.WriteHeader(http.StatusForbidden)
}

func clientTlsConfig(cacertPath string, certPath string, keyPath string) (*tls.Config, error) {
	var tlsConfig *tls.Config

	var cacertPool *x509.CertPool
	if cacertPath != "" {
		cacert, err := os.ReadFile(filepath.Clean(cacertPath))
		if err != nil {
			return nil, err
		}
		cacertPool = x509.NewCertPool()
		if ok := cacertPool.AppendCertsFromPEM(cacert); !ok {
			return nil, errors.New("error loading CA certificate")
		}
	}

	var certificates []tls.Certificate
	if certPath != "" || keyPath != "" {
		cert, err := tls.LoadX509KeyPair(certPath, keyPath)
		if err != nil {
			return nil, err
		}
		certificates = []tls.Certificate{cert}
	}

	if cacertPool != nil || len(certificates) > 0 {
		tlsConfig = &tls.Config{
			MinVersion:   minTlsVersion,
			RootCAs:      cacertPool,
			Certificates: certificates,
		}
	}

	return tlsConfig, nil
}

func serverTlsConfig(cacertPath string, certPath string, keyPath string) (*tls.Config, error) {
	var tlsConfig *tls.Config

	var clientAuth tls.ClientAuthType
	var cacertPool *x509.CertPool
	if cacertPath != "" {
		cacert, err := os.ReadFile(filepath.Clean(cacertPath))
		if err != nil {
			return nil, err
		}
		cacertPool = x509.NewCertPool()
		if ok := cacertPool.AppendCertsFromPEM(cacert); !ok {
			return nil, errors.New("error loading CA certificate")
		}
		clientAuth = tls.RequireAndVerifyClientCert
	} else {
		clientAuth = tls.NoClientCert
	}

	var certificates []tls.Certificate
	if certPath != "" || keyPath != "" {
		cert, err := tls.LoadX509KeyPair(certPath, keyPath)
		if err != nil {
			return nil, err
		}
		certificates = []tls.Certificate{cert}
	}

	if cacertPool != nil || len(certificates) > 0 {
		tlsConfig = &tls.Config{
			MinVersion:   minTlsVersion,
			Certificates: certificates,
			ClientAuth:   clientAuth,
			ClientCAs:    cacertPool,
		}
	}

	return tlsConfig, nil
}

func parseAddr(addr string) (string, string, error) {
	parts := strings.SplitN(addr, "://", 2)
	if len(parts) != 2 || len(parts[0]) == 0 || len(parts[1]) == 0 {
		return "", "", fmt.Errorf("invalid address format: %s", addr)
	}

	switch parts[0] {
	case "unix":
		return parts[0], parts[1], nil
	case "tcp":
		u, err := url.Parse(addr)
		if err != nil {
			return "", "", err
		}
		return u.Scheme, u.Host, nil
	default:
		return "", "", fmt.Errorf("unsupported address protocol: %s", addr)
	}
}

// Borrowed from net/http/server.go
func cleanPath(p string) string {
	if p == "" {
		return "/"
	}
	if p[0] != '/' {
		p = "/" + p
	}
	np := path.Clean(p)
	if p[len(p)-1] == '/' && np != "/" {
		np += "/"
	}
	return np
}
