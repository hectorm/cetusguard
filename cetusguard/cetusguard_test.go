package cetusguard

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/hectorm/cetusguard/cetusguard/testdata"
	"github.com/hectorm/cetusguard/internal/logger"
)

func TestMain(m *testing.M) {
	logger.SetLevel(logger.LvlDebug)

	code := m.Run()
	os.Exit(code)
}

func TestCetusGuardStartAndStop(t *testing.T) {
	tc := &testCase{
		daemonFunc:         plainDaemon,
		daemonListenerFunc: tcpDaemonListener,
		clientFunc:         plainClient,
		backendFunc:        plainBackend,
		frontendFunc:       plainFrontend,
	}

	defer tc.setup(t)()
	tc.daemon.Handler = http.HandlerFunc(httpDaemonHandler)

	if tc.server.IsRunning() {
		t.Fatalf("server started, want stopped")
	}

	addrs, err := tc.server.Addrs()
	if err == nil || addrs != nil {
		t.Fatalf("addr = %v, want an error", addrs)
	}

	ready := make(chan any, 1)
	go func() {
		err := tc.server.Start(ready)
		if err != nil {
			t.Error(err)
		}
	}()
	<-ready

	if !tc.server.IsRunning() {
		t.Fatalf("server stopped, want started")
	}

	ready = make(chan any, 1)
	go func() {
		err := tc.server.Start(ready)
		if err == nil {
			t.Errorf("server started, want an error")
		}
	}()
	<-ready

	_, err = tc.server.Addrs()
	if err != nil {
		t.Fatal(err)
	}

	err = tc.server.Stop()
	if err != nil {
		t.Fatal(err)
	}

	err = tc.server.Stop()
	if err == nil {
		t.Errorf("server stopped, want an error")
	}

	ready = make(chan any, 1)
	go func() {
		err := tc.server.Start(ready)
		if err != nil {
			t.Error(err)
		}
	}()
	<-ready

	if !tc.server.IsRunning() {
		t.Fatalf("server stopped, want started")
	}

	err = tc.server.Stop()
	if err != nil {
		t.Fatal(err)
	}
}

func TestCetusGuardPlainAllowedReq(t *testing.T) {
	tc := &testCase{
		daemonListenerFunc: tcpDaemonListener,
		daemonFunc:         plainDaemon,
		backendFunc:        plainBackend,
		frontendFunc:       plainFrontend,
		clientFunc:         plainClient,
	}

	defer tc.setup(t)()
	tc.daemon.Handler = http.HandlerFunc(httpDaemonHandler)

	ready := make(chan any, 1)
	go func() {
		err := tc.server.Start(ready)
		if err != nil {
			t.Error(err)
		}
	}()
	<-ready

	addrs, err := tc.server.Addrs()
	if err != nil {
		t.Fatal(err)
	}

	req, err := httpClientAllowedReq("http", addrs[0].String())
	if err != nil {
		t.Fatal(err)
	}

	res, err := tc.client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = res.Body.Close()
	}()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("res.StatusCode = %d, want %d", res.StatusCode, http.StatusOK)
	}

	msg, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}

	if string(msg) != "PONG" {
		t.Fatalf(`msg = "%s", want "%s"`, msg, "PONG")
	}

	err = tc.server.Stop()
	if err != nil {
		t.Fatal(err)
	}
}

func TestCetusGuardPlainAllowedStreamReq(t *testing.T) {
	tc := &testCase{
		daemonListenerFunc: tcpDaemonListener,
		daemonFunc:         plainDaemon,
		backendFunc:        plainBackend,
		frontendFunc:       plainFrontend,
		clientFunc:         plainClient,
	}

	defer tc.setup(t)()
	tc.daemon.Handler = http.HandlerFunc(httpDaemonHandler)

	ready := make(chan any, 1)
	go func() {
		err := tc.server.Start(ready)
		if err != nil {
			t.Error(err)
		}
	}()
	<-ready

	addrs, err := tc.server.Addrs()
	if err != nil {
		t.Fatal(err)
	}

	req, err := httpClientAllowedReq("http", addrs[0].String())
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Upgrade", "tcp")
	req.Header.Set("Connection", "Upgrade")

	res, err := tc.client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = res.Body.Close()
	}()

	if res.StatusCode != http.StatusSwitchingProtocols {
		t.Fatalf("res.StatusCode = %d, want %d", res.StatusCode, http.StatusSwitchingProtocols)
	}

	msg, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}

	if string(msg) != "PONG" {
		t.Fatalf(`msg = "%s", want "%s"`, msg, "PONG")
	}

	err = tc.server.Stop()
	if err != nil {
		t.Fatal(err)
	}
}

func TestCetusGuardPlainDeniedMethodReq(t *testing.T) {
	tc := &testCase{
		daemonListenerFunc: tcpDaemonListener,
		daemonFunc:         plainDaemon,
		backendFunc:        plainBackend,
		frontendFunc:       plainFrontend,
		clientFunc:         plainClient,
	}

	defer tc.setup(t)()
	tc.daemon.Handler = http.HandlerFunc(httpDaemonHandler)

	ready := make(chan any, 1)
	go func() {
		err := tc.server.Start(ready)
		if err != nil {
			t.Error(err)
		}
	}()
	<-ready

	addrs, err := tc.server.Addrs()
	if err != nil {
		t.Fatal(err)
	}

	req, err := httpClientDeniedMethodReq("http", addrs[0].String())
	if err != nil {
		t.Fatal(err)
	}

	res, err := tc.client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = res.Body.Close()
	}()

	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("res.StatusCode = %d, want %d", res.StatusCode, http.StatusForbidden)
	}

	err = tc.server.Stop()
	if err != nil {
		t.Fatal(err)
	}
}

func TestCetusGuardPlainDeniedPatternReq(t *testing.T) {
	tc := &testCase{
		daemonListenerFunc: tcpDaemonListener,
		daemonFunc:         plainDaemon,
		backendFunc:        plainBackend,
		frontendFunc:       plainFrontend,
		clientFunc:         plainClient,
	}

	defer tc.setup(t)()
	tc.daemon.Handler = http.HandlerFunc(httpDaemonHandler)

	ready := make(chan any, 1)
	go func() {
		err := tc.server.Start(ready)
		if err != nil {
			t.Error(err)
		}
	}()
	<-ready

	addrs, err := tc.server.Addrs()
	if err != nil {
		t.Fatal(err)
	}

	req, err := httpClientDeniedPatternReq("http", addrs[0].String())
	if err != nil {
		t.Fatal(err)
	}

	res, err := tc.client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = res.Body.Close()
	}()

	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("res.StatusCode = %d, want %d", res.StatusCode, http.StatusForbidden)
	}

	err = tc.server.Stop()
	if err != nil {
		t.Fatal(err)
	}
}

func TestCetusGuardPlainTlsAuthBackendReq(t *testing.T) {
	tc := &testCase{
		daemonListenerFunc: tcpDaemonListener,
		daemonFunc:         tlsAuthDaemon,
		backendFunc:        tlsAuthBackend,
		frontendFunc:       plainFrontend,
		clientFunc:         plainClient,
	}

	defer tc.setup(t)()
	tc.daemon.Handler = http.HandlerFunc(httpDaemonHandler)

	ready := make(chan any, 1)
	go func() {
		err := tc.server.Start(ready)
		if err != nil {
			t.Error(err)
		}
	}()
	<-ready

	addrs, err := tc.server.Addrs()
	if err != nil {
		t.Fatal(err)
	}

	req, err := httpClientAllowedReq("http", addrs[0].String())
	if err != nil {
		t.Fatal(err)
	}

	res, err := tc.client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = res.Body.Close()
	}()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("res.StatusCode = %d, want %d", res.StatusCode, http.StatusOK)
	}

	msg, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}

	if string(msg) != "PONG" {
		t.Fatalf(`msg = "%s", want "%s"`, msg, "PONG")
	}

	err = tc.server.Stop()
	if err != nil {
		t.Fatal(err)
	}
}

func TestCetusGuardTlsAllowedReq(t *testing.T) {
	tc := &testCase{
		daemonListenerFunc: tcpDaemonListener,
		daemonFunc:         tlsDaemon,
		backendFunc:        tlsBackend,
		frontendFunc:       tlsFrontend,
		clientFunc:         tlsClient,
	}

	defer tc.setup(t)()
	tc.daemon.Handler = http.HandlerFunc(httpDaemonHandler)

	ready := make(chan any, 1)
	go func() {
		err := tc.server.Start(ready)
		if err != nil {
			t.Error(err)
		}
	}()
	<-ready

	addrs, err := tc.server.Addrs()
	if err != nil {
		t.Fatal(err)
	}

	req, err := httpClientAllowedReq("https", addrs[0].String())
	if err != nil {
		t.Fatal(err)
	}

	res, err := tc.client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = res.Body.Close()
	}()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("res.StatusCode = %d, want %d", res.StatusCode, http.StatusOK)
	}

	msg, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}

	if string(msg) != "PONG" {
		t.Fatalf(`msg = "%s", want "%s"`, msg, "PONG")
	}

	err = tc.server.Stop()
	if err != nil {
		t.Fatal(err)
	}
}

func TestCetusGuardTlsDeniedMethodReq(t *testing.T) {
	tc := &testCase{
		daemonListenerFunc: tcpDaemonListener,
		daemonFunc:         tlsDaemon,
		backendFunc:        tlsBackend,
		frontendFunc:       tlsFrontend,
		clientFunc:         tlsClient,
	}

	defer tc.setup(t)()
	tc.daemon.Handler = http.HandlerFunc(httpDaemonHandler)

	ready := make(chan any, 1)
	go func() {
		err := tc.server.Start(ready)
		if err != nil {
			t.Error(err)
		}
	}()
	<-ready

	addrs, err := tc.server.Addrs()
	if err != nil {
		t.Fatal(err)
	}

	req, err := httpClientDeniedMethodReq("https", addrs[0].String())
	if err != nil {
		t.Fatal(err)
	}

	res, err := tc.client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = res.Body.Close()
	}()

	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("res.StatusCode = %d, want %d", res.StatusCode, http.StatusForbidden)
	}

	err = tc.server.Stop()
	if err != nil {
		t.Fatal(err)
	}
}

func TestCetusGuardTlsDeniedPatternReq(t *testing.T) {
	tc := &testCase{
		daemonListenerFunc: tcpDaemonListener,
		daemonFunc:         tlsDaemon,
		backendFunc:        tlsBackend,
		frontendFunc:       tlsFrontend,
		clientFunc:         tlsClient,
	}

	defer tc.setup(t)()
	tc.daemon.Handler = http.HandlerFunc(httpDaemonHandler)

	ready := make(chan any, 1)
	go func() {
		err := tc.server.Start(ready)
		if err != nil {
			t.Error(err)
		}
	}()
	<-ready

	addrs, err := tc.server.Addrs()
	if err != nil {
		t.Fatal(err)
	}

	req, err := httpClientDeniedPatternReq("https", addrs[0].String())
	if err != nil {
		t.Fatal(err)
	}

	res, err := tc.client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = res.Body.Close()
	}()

	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("res.StatusCode = %d, want %d", res.StatusCode, http.StatusForbidden)
	}

	err = tc.server.Stop()
	if err != nil {
		t.Fatal(err)
	}
}

func TestCetusGuardTlsAuthAllowedReq(t *testing.T) {
	tc := &testCase{
		daemonListenerFunc: tcpDaemonListener,
		daemonFunc:         tlsAuthDaemon,
		backendFunc:        tlsAuthBackend,
		frontendFunc:       tlsAuthFrontend,
		clientFunc:         tlsAuthClient,
	}

	defer tc.setup(t)()
	tc.daemon.Handler = http.HandlerFunc(httpDaemonHandler)

	ready := make(chan any, 1)
	go func() {
		err := tc.server.Start(ready)
		if err != nil {
			t.Error(err)
		}
	}()
	<-ready

	addrs, err := tc.server.Addrs()
	if err != nil {
		t.Fatal(err)
	}

	req, err := httpClientAllowedReq("https", addrs[0].String())
	if err != nil {
		t.Fatal(err)
	}

	res, err := tc.client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = res.Body.Close()
	}()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("res.StatusCode = %d, want %d", res.StatusCode, http.StatusOK)
	}

	msg, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}

	if string(msg) != "PONG" {
		t.Fatalf(`msg = "%s", want "%s"`, msg, "PONG")
	}

	err = tc.server.Stop()
	if err != nil {
		t.Fatal(err)
	}
}

func TestCetusGuardTlsAuthAllowedStreamReq(t *testing.T) {
	tc := &testCase{
		daemonListenerFunc: tcpDaemonListener,
		daemonFunc:         tlsAuthDaemon,
		backendFunc:        tlsAuthBackend,
		frontendFunc:       tlsAuthFrontend,
		clientFunc:         tlsAuthClient,
	}

	defer tc.setup(t)()
	tc.daemon.Handler = http.HandlerFunc(httpDaemonHandler)

	ready := make(chan any, 1)
	go func() {
		err := tc.server.Start(ready)
		if err != nil {
			t.Error(err)
		}
	}()
	<-ready

	addrs, err := tc.server.Addrs()
	if err != nil {
		t.Fatal(err)
	}

	req, err := httpClientAllowedReq("https", addrs[0].String())
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Upgrade", "tcp")
	req.Header.Set("Connection", "Upgrade")

	res, err := tc.client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = res.Body.Close()
	}()

	if res.StatusCode != http.StatusSwitchingProtocols {
		t.Fatalf("res.StatusCode = %d, want %d", res.StatusCode, http.StatusSwitchingProtocols)
	}

	msg, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}

	if string(msg) != "PONG" {
		t.Fatalf(`msg = "%s", want "%s"`, msg, "PONG")
	}

	err = tc.server.Stop()
	if err != nil {
		t.Fatal(err)
	}
}

func TestCetusGuardTlsAuthDeniedMethodReq(t *testing.T) {
	tc := &testCase{
		daemonListenerFunc: tcpDaemonListener,
		daemonFunc:         tlsAuthDaemon,
		backendFunc:        tlsAuthBackend,
		frontendFunc:       tlsAuthFrontend,
		clientFunc:         tlsAuthClient,
	}

	defer tc.setup(t)()
	tc.daemon.Handler = http.HandlerFunc(httpDaemonHandler)

	ready := make(chan any, 1)
	go func() {
		err := tc.server.Start(ready)
		if err != nil {
			t.Error(err)
		}
	}()
	<-ready

	addrs, err := tc.server.Addrs()
	if err != nil {
		t.Fatal(err)
	}

	req, err := httpClientDeniedMethodReq("https", addrs[0].String())
	if err != nil {
		t.Fatal(err)
	}

	res, err := tc.client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = res.Body.Close()
	}()

	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("res.StatusCode = %d, want %d", res.StatusCode, http.StatusForbidden)
	}

	err = tc.server.Stop()
	if err != nil {
		t.Fatal(err)
	}
}

func TestCetusGuardTlsAuthDeniedPatternReq(t *testing.T) {
	tc := &testCase{
		daemonListenerFunc: tcpDaemonListener,
		daemonFunc:         tlsAuthDaemon,
		backendFunc:        tlsAuthBackend,
		frontendFunc:       tlsAuthFrontend,
		clientFunc:         tlsAuthClient,
	}

	defer tc.setup(t)()
	tc.daemon.Handler = http.HandlerFunc(httpDaemonHandler)

	ready := make(chan any, 1)
	go func() {
		err := tc.server.Start(ready)
		if err != nil {
			t.Error(err)
		}
	}()
	<-ready

	addrs, err := tc.server.Addrs()
	if err != nil {
		t.Fatal(err)
	}

	req, err := httpClientDeniedPatternReq("https", addrs[0].String())
	if err != nil {
		t.Fatal(err)
	}

	res, err := tc.client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = res.Body.Close()
	}()

	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("res.StatusCode = %d, want %d", res.StatusCode, http.StatusForbidden)
	}

	err = tc.server.Stop()
	if err != nil {
		t.Fatal(err)
	}
}

func TestCetusGuardTlsAuthPlainBackendReq(t *testing.T) {
	tc := &testCase{
		daemonListenerFunc: tcpDaemonListener,
		daemonFunc:         plainDaemon,
		backendFunc:        plainBackend,
		frontendFunc:       tlsAuthFrontend,
		clientFunc:         tlsAuthClient,
	}

	defer tc.setup(t)()
	tc.daemon.Handler = http.HandlerFunc(httpDaemonHandler)

	ready := make(chan any, 1)
	go func() {
		err := tc.server.Start(ready)
		if err != nil {
			t.Error(err)
		}
	}()
	<-ready

	addrs, err := tc.server.Addrs()
	if err != nil {
		t.Fatal(err)
	}

	req, err := httpClientAllowedReq("https", addrs[0].String())
	if err != nil {
		t.Fatal(err)
	}

	res, err := tc.client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = res.Body.Close()
	}()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("res.StatusCode = %d, want %d", res.StatusCode, http.StatusOK)
	}

	msg, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}

	if string(msg) != "PONG" {
		t.Fatalf(`msg = "%s", want "%s"`, msg, "PONG")
	}

	err = tc.server.Stop()
	if err != nil {
		t.Fatal(err)
	}
}

func TestCetusGuardExpiredDaemonCertReq(t *testing.T) {
	tc := &testCase{
		daemonListenerFunc: tcpDaemonListener,
		daemonFunc:         expiredTlsDaemon,
		backendFunc:        tlsBackend,
		frontendFunc:       tlsFrontend,
		clientFunc:         tlsClient,
	}

	defer tc.setup(t)()
	tc.daemon.Handler = http.HandlerFunc(httpDaemonHandler)

	ready := make(chan any, 1)
	go func() {
		err := tc.server.Start(ready)
		if err != nil {
			t.Error(err)
		}
	}()
	<-ready

	addrs, err := tc.server.Addrs()
	if err != nil {
		t.Fatal(err)
	}

	req, err := httpClientAllowedReq("https", addrs[0].String())
	if err != nil {
		t.Fatal(err)
	}

	res, err := tc.client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = res.Body.Close()
	}()

	if res.StatusCode != http.StatusBadGateway {
		t.Fatalf("res.StatusCode = %d, want %d", res.StatusCode, http.StatusBadGateway)
	}

	err = tc.server.Stop()
	if err != nil {
		t.Fatal(err)
	}
}

func TestCetusGuardUntrustedDaemonCertReq(t *testing.T) {
	tc := &testCase{
		daemonListenerFunc: tcpDaemonListener,
		daemonFunc:         altTlsDaemon,
		backendFunc:        tlsBackend,
		frontendFunc:       tlsFrontend,
		clientFunc:         tlsClient,
	}

	defer tc.setup(t)()
	tc.daemon.Handler = http.HandlerFunc(httpDaemonHandler)

	ready := make(chan any, 1)
	go func() {
		err := tc.server.Start(ready)
		if err != nil {
			t.Error(err)
		}
	}()
	<-ready

	addrs, err := tc.server.Addrs()
	if err != nil {
		t.Fatal(err)
	}

	req, err := httpClientAllowedReq("https", addrs[0].String())
	if err != nil {
		t.Fatal(err)
	}

	res, err := tc.client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = res.Body.Close()
	}()

	if res.StatusCode != http.StatusBadGateway {
		t.Fatalf("res.StatusCode = %d, want %d", res.StatusCode, http.StatusBadGateway)
	}

	err = tc.server.Stop()
	if err != nil {
		t.Fatal(err)
	}
}

func TestCetusGuardUntrustedClientCertReq(t *testing.T) {
	tc := &testCase{
		daemonListenerFunc: tcpDaemonListener,
		daemonFunc:         tlsDaemon,
		backendFunc:        tlsBackend,
		frontendFunc:       tlsAuthFrontend,
		clientFunc:         altTlsAuthClient,
	}

	defer tc.setup(t)()
	tc.daemon.Handler = http.HandlerFunc(httpDaemonHandler)

	ready := make(chan any, 1)
	go func() {
		err := tc.server.Start(ready)
		if err != nil {
			t.Error(err)
		}
	}()
	<-ready

	addrs, err := tc.server.Addrs()
	if err != nil {
		t.Fatal(err)
	}

	req, err := httpClientAllowedReq("https", addrs[0].String())
	if err != nil {
		t.Fatal(err)
	}

	res, err := tc.client.Do(req)
	if err == nil || res != nil {
		t.Fatalf("response returned, want an error")
	}

	err = tc.server.Stop()
	if err != nil {
		t.Fatal(err)
	}
}

func TestCetusGuardInvalidBackendCacert(t *testing.T) {
	tc := &testCase{
		daemonListenerFunc: tcpDaemonListener,
		daemonFunc:         tlsDaemon,
		backendFunc:        invalidCacertTlsBackend,
		frontendFunc:       tlsFrontend,
		clientFunc:         tlsClient,
	}

	defer tc.setup(t)()
	tc.daemon.Handler = http.HandlerFunc(httpDaemonHandler)

	ready := make(chan any, 1)
	go func() {
		err := tc.server.Start(ready)
		if err == nil {
			t.Errorf("server started, want an error")
		}
	}()
	<-ready

	_ = tc.server.Stop()
}

func TestCetusGuardInvalidBackendCert(t *testing.T) {
	tc := &testCase{
		daemonListenerFunc: tcpDaemonListener,
		daemonFunc:         tlsAuthDaemon,
		backendFunc:        invalidCertTlsAuthBackend,
		frontendFunc:       tlsFrontend,
		clientFunc:         tlsClient,
	}

	defer tc.setup(t)()
	tc.daemon.Handler = http.HandlerFunc(httpDaemonHandler)

	ready := make(chan any, 1)
	go func() {
		err := tc.server.Start(ready)
		if err == nil {
			t.Errorf("server started, want an error")
		}
	}()
	<-ready

	_ = tc.server.Stop()
}

func TestCetusGuardInvalidFrontendCacert(t *testing.T) {
	tc := &testCase{
		daemonListenerFunc: tcpDaemonListener,
		daemonFunc:         tlsDaemon,
		backendFunc:        tlsBackend,
		frontendFunc:       invalidCacertTlsAuthFrontend,
		clientFunc:         tlsAuthClient,
	}

	defer tc.setup(t)()
	tc.daemon.Handler = http.HandlerFunc(httpDaemonHandler)

	ready := make(chan any, 1)
	go func() {
		err := tc.server.Start(ready)
		if err == nil {
			t.Errorf("server started, want an error")
		}
	}()
	<-ready

	_ = tc.server.Stop()
}

func TestCetusGuardInvalidFrontendCert(t *testing.T) {
	tc := &testCase{
		daemonListenerFunc: tcpDaemonListener,
		daemonFunc:         tlsDaemon,
		backendFunc:        tlsBackend,
		frontendFunc:       invalidCertTlsFrontend,
		clientFunc:         tlsClient,
	}

	defer tc.setup(t)()
	tc.daemon.Handler = http.HandlerFunc(httpDaemonHandler)

	ready := make(chan any, 1)
	go func() {
		err := tc.server.Start(ready)
		if err == nil {
			t.Errorf("server started, want an error")
		}
	}()
	<-ready

	_ = tc.server.Stop()
}

func TestCetusGuardInvalidBackendAddr(t *testing.T) {
	tc := &testCase{
		daemonListenerFunc: tcpDaemonListener,
		daemonFunc:         plainDaemon,
		backendFunc:        plainBackend,
		frontendFunc:       plainFrontend,
		clientFunc:         plainClient,
	}

	defer tc.setup(t)()
	tc.daemon.Handler = http.HandlerFunc(httpDaemonHandler)
	tc.server.Backend.Addr = "invalid"

	ready := make(chan any, 1)
	go func() {
		err := tc.server.Start(ready)
		if err == nil {
			t.Errorf("server started, want an error")
		}
	}()
	<-ready

	tc.server.Backend.Addr = "invalid://127.0.0.1:0"

	ready = make(chan any, 1)
	go func() {
		err := tc.server.Start(ready)
		if err == nil {
			t.Errorf("server started, want an error")
		}
	}()
	<-ready

	_ = tc.server.Stop()
}

func TestCetusGuardInvalidFrontendAddr(t *testing.T) {
	tc := &testCase{
		daemonListenerFunc: tcpDaemonListener,
		daemonFunc:         plainDaemon,
		backendFunc:        plainBackend,
		frontendFunc:       plainFrontend,
		clientFunc:         plainClient,
	}

	defer tc.setup(t)()
	tc.daemon.Handler = http.HandlerFunc(httpDaemonHandler)
	tc.server.Frontend.Addr = []string{"tcp://127.0.0.1:0", "invalid"}

	ready := make(chan any, 1)
	go func() {
		err := tc.server.Start(ready)
		if err == nil {
			t.Errorf("server started, want an error")
		}
	}()
	<-ready

	tc.server.Frontend.Addr = []string{"tcp://127.0.0.1:0", "invalid://127.0.0.1:0"}

	ready = make(chan any, 1)
	go func() {
		err := tc.server.Start(ready)
		if err == nil {
			t.Errorf("server started, want an error")
		}
	}()
	<-ready

	_ = tc.server.Stop()
}

func TestCetusGuardFrontendListenMultiple(t *testing.T) {
	tc := &testCase{
		daemonListenerFunc: tcpDaemonListener,
		daemonFunc:         plainDaemon,
		backendFunc:        plainBackend,
		frontendFunc:       plainFrontend,
		clientFunc:         plainClient,
	}

	defer tc.setup(t)()
	tc.daemon.Handler = http.HandlerFunc(httpDaemonHandler)
	tc.server.Frontend.Addr = []string{"tcp://127.0.0.1:0", "tcp://127.0.0.1:0", "tcp://127.0.0.1:0", "tcp://127.0.0.1:0"}

	ready := make(chan any, 1)
	go func() {
		err := tc.server.Start(ready)
		if err != nil {
			t.Error(err)
		}
	}()
	<-ready

	addrs, err := tc.server.Addrs()
	if err != nil {
		t.Fatal(err)
	}
	if len(addrs) != len(tc.server.Frontend.Addr) {
		t.Fatalf("len(addrs) = %d, want %d", len(addrs), len(tc.server.Frontend.Addr))
	}

	for _, addr := range addrs {
		req, err := httpClientAllowedReq("http", addr.String())
		if err != nil {
			t.Fatal(err)
		}

		res, err := tc.client.Do(req)
		if err != nil {
			t.Fatal(err)
		}

		if res.StatusCode != http.StatusOK {
			t.Fatalf("res.StatusCode = %d, want %d", res.StatusCode, http.StatusOK)
		}

		msg, err := io.ReadAll(res.Body)
		if err != nil {
			t.Fatal(err)
		}
		_ = res.Body.Close()

		if string(msg) != "PONG" {
			t.Fatalf(`msg = "%s", want "%s"`, msg, "PONG")
		}
	}

	err = tc.server.Stop()
	if err != nil {
		t.Fatal(err)
	}
}

func httpClientAllowedReq(scheme string, addr string) (*http.Request, error) {
	body := strings.NewReader("PING")
	req, err := http.NewRequest("POST", "/~foo+bar+%F0%9F%90%B3?foo=bar", body)
	if err != nil {
		return nil, err
	}
	req.URL.Scheme = scheme
	req.URL.Host = addr

	return req, nil
}

func httpClientDeniedMethodReq(scheme string, addr string) (*http.Request, error) {
	body := strings.NewReader("PING")
	req, err := http.NewRequest("PATCH", "/~foo+bar+%F0%9F%90%B3?foo=bar", body)
	if err != nil {
		return nil, err
	}
	req.URL.Scheme = scheme
	req.URL.Host = addr

	return req, nil
}

func httpClientDeniedPatternReq(scheme string, addr string) (*http.Request, error) {
	body := strings.NewReader("PING")
	req, err := http.NewRequest("PUT", "/~foo+bar?foo=bar", body)
	if err != nil {
		return nil, err
	}
	req.URL.Scheme = scheme
	req.URL.Host = addr

	return req, nil
}

func httpDaemonHandler(wri http.ResponseWriter, req *http.Request) {
	b, err := ioutil.ReadAll(req.Body)
	if err != nil {
		wri.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer func() {
		_ = req.Body.Close()
	}()

	if string(b) != "PING" {
		wri.WriteHeader(http.StatusBadRequest)
		return
	}

	if req.Header.Get("Upgrade") == "tcp" {
		hj, ok := wri.(http.Hijacker)
		if !ok {
			wri.WriteHeader(http.StatusInternalServerError)
			return
		}

		conn, brw, err := hj.Hijack()
		if err != nil {
			return
		}
		defer func() {
			_ = conn.Close()
		}()

		_, err = brw.Write([]byte("HTTP/1.1 101 UPGRADED\r\nConnection: Upgrade\r\nUpgrade: tcp\r\nContent-Type: " + contentTypeRawStream + "\r\n\r\nPONG"))
		if err != nil {
			return
		}

		err = brw.Flush()
		if err != nil {
			return
		}
	} else {
		wri.WriteHeader(http.StatusOK)
		_, err = fmt.Fprintf(wri, "PONG")
		if err != nil {
			return
		}
	}
}

func tcpDaemonListener(_ string) (net.Listener, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}

	return listener, nil
}

func plainDaemon() (*http.Server, error) {
	server := &http.Server{
		ReadTimeout:       120 * time.Minute,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      120 * time.Minute,
		IdleTimeout:       90 * time.Second,
	}

	return server, nil
}

func tlsDaemon() (*http.Server, error) {
	server, err := plainDaemon()
	if err != nil {
		return nil, err
	}

	cert, err := tls.X509KeyPair(testdata.TestTlsServerCert, testdata.TestTlsServerKey)
	if err != nil {
		return nil, err
	}
	server.TLSConfig = &tls.Config{
		MinVersion:   minTlsVersion,
		Certificates: []tls.Certificate{cert},
	}

	return server, nil
}

func expiredTlsDaemon() (*http.Server, error) {
	server, err := plainDaemon()
	if err != nil {
		return nil, err
	}

	cert, err := tls.X509KeyPair(testdata.TestTlsServerExpiredCert, testdata.TestTlsServerKey)
	if err != nil {
		return nil, err
	}
	server.TLSConfig = &tls.Config{
		MinVersion:   minTlsVersion,
		Certificates: []tls.Certificate{cert},
	}

	return server, nil
}

func altTlsDaemon() (*http.Server, error) {
	server, err := plainDaemon()
	if err != nil {
		return nil, err
	}

	cert, err := tls.X509KeyPair(testdata.TestAltTlsServerCert, testdata.TestAltTlsServerKey)
	if err != nil {
		return nil, err
	}
	server.TLSConfig = &tls.Config{
		MinVersion:   minTlsVersion,
		Certificates: []tls.Certificate{cert},
	}

	return server, nil
}

func tlsAuthDaemon() (*http.Server, error) {
	server, err := tlsDaemon()
	if err != nil {
		return nil, err
	}

	cacertPool := x509.NewCertPool()
	if ok := cacertPool.AppendCertsFromPEM(testdata.TestTlsCacert); !ok {
		return nil, errors.New("error loading CA certificate")
	}
	server.TLSConfig.ClientAuth = tls.RequireAndVerifyClientCert
	server.TLSConfig.ClientCAs = cacertPool

	return server, nil
}

func plainBackend(listener net.Listener, _ string) (*Backend, error) {
	backend := &Backend{
		Addr: fmt.Sprintf(
			"%s://%s",
			listener.Addr().Network(),
			listener.Addr().String(),
		),
	}

	return backend, nil
}

func tlsBackend(listener net.Listener, tmpdir string) (*Backend, error) {
	backend, err := plainBackend(listener, tmpdir)
	if err != nil {
		return nil, err
	}

	clientCacertPath := filepath.Join(tmpdir, "client-ca.pem")
	if err := os.WriteFile(clientCacertPath, testdata.TestTlsCacert, 0600); err != nil {
		return nil, err
	}
	backend.TlsCacert = clientCacertPath

	return backend, nil
}

func invalidCacertTlsBackend(listener net.Listener, tmpdir string) (*Backend, error) {
	backend, err := plainBackend(listener, tmpdir)
	if err != nil {
		return nil, err
	}

	clientCacertPath := filepath.Join(tmpdir, "client-ca.pem")
	if err := os.WriteFile(clientCacertPath, testdata.TestInvalidTlsCacert, 0600); err != nil {
		return nil, err
	}
	backend.TlsCacert = clientCacertPath

	return backend, nil
}

func tlsAuthBackend(listener net.Listener, tmpdir string) (*Backend, error) {
	backend, err := tlsBackend(listener, tmpdir)
	if err != nil {
		return nil, err
	}

	clientCertPath := filepath.Join(tmpdir, "client-cert.pem")
	if err := os.WriteFile(clientCertPath, testdata.TestTlsClientCert, 0600); err != nil {
		return nil, err
	}
	backend.TlsCert = clientCertPath

	clientKeyPath := filepath.Join(tmpdir, "client-key.pem")
	if err := os.WriteFile(clientKeyPath, testdata.TestTlsClientKey, 0600); err != nil {
		return nil, err
	}
	backend.TlsKey = clientKeyPath

	return backend, nil
}

func invalidCertTlsAuthBackend(listener net.Listener, tmpdir string) (*Backend, error) {
	backend, err := tlsBackend(listener, tmpdir)
	if err != nil {
		return nil, err
	}

	clientCertPath := filepath.Join(tmpdir, "client-cert.pem")
	if err := os.WriteFile(clientCertPath, testdata.TestInvalidTlsClientCert, 0600); err != nil {
		return nil, err
	}
	backend.TlsCert = clientCertPath

	clientKeyPath := filepath.Join(tmpdir, "client-key.pem")
	if err := os.WriteFile(clientKeyPath, testdata.TestInvalidTlsClientKey, 0600); err != nil {
		return nil, err
	}
	backend.TlsKey = clientKeyPath

	return backend, nil
}

func plainFrontend(_ string) (*Frontend, error) {
	frontend := &Frontend{
		Addr: []string{"tcp://127.0.0.1:0"},
	}

	return frontend, nil
}

func tlsFrontend(tmpdir string) (*Frontend, error) {
	frontend, err := plainFrontend(tmpdir)
	if err != nil {
		return nil, err
	}

	serverCertPath := filepath.Join(tmpdir, "server-cert.pem")
	if err := os.WriteFile(serverCertPath, testdata.TestTlsServerCert, 0600); err != nil {
		return nil, err
	}
	frontend.TlsCert = serverCertPath

	serverKeyPath := filepath.Join(tmpdir, "server-key.pem")
	if err := os.WriteFile(serverKeyPath, testdata.TestTlsServerKey, 0600); err != nil {
		return nil, err
	}
	frontend.TlsKey = serverKeyPath

	return frontend, nil
}

func invalidCertTlsFrontend(tmpdir string) (*Frontend, error) {
	frontend, err := plainFrontend(tmpdir)
	if err != nil {
		return nil, err
	}

	serverCertPath := filepath.Join(tmpdir, "server-cert.pem")
	if err := os.WriteFile(serverCertPath, testdata.TestInvalidTlsServerCert, 0600); err != nil {
		return nil, err
	}
	frontend.TlsCert = serverCertPath

	serverKeyPath := filepath.Join(tmpdir, "server-key.pem")
	if err := os.WriteFile(serverKeyPath, testdata.TestInvalidTlsServerKey, 0600); err != nil {
		return nil, err
	}
	frontend.TlsKey = serverKeyPath

	return frontend, nil
}

func tlsAuthFrontend(tmpdir string) (*Frontend, error) {
	frontend, err := tlsFrontend(tmpdir)
	if err != nil {
		return nil, err
	}

	serverCacertPath := filepath.Join(tmpdir, "server-ca.pem")
	if err := os.WriteFile(serverCacertPath, testdata.TestTlsCacert, 0600); err != nil {
		return nil, err
	}
	frontend.TlsCacert = serverCacertPath

	return frontend, nil
}

func invalidCacertTlsAuthFrontend(tmpdir string) (*Frontend, error) {
	frontend, err := tlsFrontend(tmpdir)
	if err != nil {
		return nil, err
	}

	serverCacertPath := filepath.Join(tmpdir, "server-ca.pem")
	if err := os.WriteFile(serverCacertPath, testdata.TestInvalidTlsCacert, 0600); err != nil {
		return nil, err
	}
	frontend.TlsCacert = serverCacertPath

	return frontend, nil
}

func plainClient() (*http.Client, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:          10,
			MaxIdleConnsPerHost:   10,
			TLSHandshakeTimeout:   10 * time.Second,
			IdleConnTimeout:       90 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}

	return client, nil
}

func tlsClient() (*http.Client, error) {
	client, err := plainClient()
	if err != nil {
		return nil, err
	}

	cacertPool := x509.NewCertPool()
	if ok := cacertPool.AppendCertsFromPEM(testdata.TestTlsCacert); !ok {
		return nil, errors.New("error loading CA certificate")
	}
	transport := client.Transport.(*http.Transport)
	/* #nosec G402 */
	transport.TLSClientConfig = &tls.Config{
		MinVersion:         minTlsVersion,
		RootCAs:            cacertPool,
		InsecureSkipVerify: true,
	}

	return client, nil
}

func altTlsClient() (*http.Client, error) {
	client, err := plainClient()
	if err != nil {
		return nil, err
	}

	cacertPool := x509.NewCertPool()
	if ok := cacertPool.AppendCertsFromPEM(testdata.TestAltTlsCacert); !ok {
		return nil, errors.New("error loading CA certificate")
	}
	transport := client.Transport.(*http.Transport)
	/* #nosec G402 */
	transport.TLSClientConfig = &tls.Config{
		MinVersion:         minTlsVersion,
		RootCAs:            cacertPool,
		InsecureSkipVerify: true,
	}

	return client, nil
}

func tlsAuthClient() (*http.Client, error) {
	client, err := tlsClient()
	if err != nil {
		return nil, err
	}

	cert, err := tls.X509KeyPair(testdata.TestTlsClientCert, testdata.TestTlsClientKey)
	if err != nil {
		return nil, err
	}
	transport := client.Transport.(*http.Transport)
	transport.TLSClientConfig.Certificates = []tls.Certificate{cert}

	return client, nil
}

func altTlsAuthClient() (*http.Client, error) {
	client, err := altTlsClient()
	if err != nil {
		return nil, err
	}

	cert, err := tls.X509KeyPair(testdata.TestAltTlsClientCert, testdata.TestAltTlsClientKey)
	if err != nil {
		return nil, err
	}
	transport := client.Transport.(*http.Transport)
	transport.TLSClientConfig.Certificates = []tls.Certificate{cert}

	return client, nil
}

type testCase struct {
	daemonListener     net.Listener
	daemonListenerFunc func(tmpdir string) (net.Listener, error)

	daemon     *http.Server
	daemonFunc func() (*http.Server, error)

	backend     *Backend
	backendFunc func(listener net.Listener, tmpdir string) (*Backend, error)

	frontend     *Frontend
	frontendFunc func(tmpdir string) (*Frontend, error)

	client     *http.Client
	clientFunc func() (*http.Client, error)

	server *Server
}

func (tc *testCase) setup(t *testing.T) func() {
	tmpdir := t.TempDir()
	var err error

	tc.daemonListener, err = tc.daemonListenerFunc(tmpdir)
	if err != nil {
		t.Fatal(err)
	}

	tc.daemon, err = tc.daemonFunc()
	if err != nil {
		t.Fatal(err)
	}

	tc.backend, err = tc.backendFunc(tc.daemonListener, tmpdir)
	if err != nil {
		t.Fatal(err)
	}

	tc.frontend, err = tc.frontendFunc(tmpdir)
	if err != nil {
		t.Fatal(err)
	}

	tc.client, err = tc.clientFunc()
	if err != nil {
		t.Fatal(err)
	}

	tc.server = &Server{
		Backend:  tc.backend,
		Frontend: tc.frontend,
		Rules: []Rule{{
			Methods: map[string]struct{}{"HEAD": {}},
			Pattern: regexp.MustCompile(`^.*$`),
		}, {
			Methods: map[string]struct{}{"GET": {}, "POST": {}, "PUT": {}},
			Pattern: regexp.MustCompile(`^/~foo\+bar\+\x{1F433}$`),
		}},
	}

	go func() {
		var err error
		if tc.backend.TlsCacert != "" {
			err = tc.daemon.ServeTLS(tc.daemonListener, "", "")
		} else {
			err = tc.daemon.Serve(tc.daemonListener)
		}
		if err != http.ErrServerClosed {
			t.Error(err)
		}
	}()

	return func() {
		err := tc.daemon.Close()
		if err != nil {
			t.Error(err)
		}
	}
}
