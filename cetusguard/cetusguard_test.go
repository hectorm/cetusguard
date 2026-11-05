package cetusguard

import (
	"bufio"
	"bytes"
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
	"path/filepath"
	"regexp"
	"slices"
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

func TestCetusGuardPlainAllowedNormalizedReq(t *testing.T) {
	tc := &testCase{
		daemonListenerFunc: tcpDaemonListener,
		daemonFunc:         plainDaemon,
		backendFunc:        plainBackend,
		frontendFunc:       plainFrontend,
		clientFunc:         plainClient,
	}

	defer tc.setup(t)()
	uris := make(chan string, 1)
	tc.daemon.Handler = http.HandlerFunc(func(wri http.ResponseWriter, req *http.Request) {
		uris <- req.RequestURI
		httpDaemonHandler(wri, req)
	})

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

	req, err := httpClientAllowedNormalizedReq("http", addrs[0].String())
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

	wantedURI := "/~foo+bar+%F0%9F%90%B3?a=1&b=2"
	if uri := <-uris; uri != wantedURI {
		t.Fatalf(`uri = "%s", want "%s"`, uri, wantedURI)
	}

	err = tc.server.Stop()
	if err != nil {
		t.Fatal(err)
	}
}

func TestCetusGuardPlainAllowedEncodedSlashReq(t *testing.T) {
	tc := &testCase{
		daemonListenerFunc: tcpDaemonListener,
		daemonFunc:         plainDaemon,
		backendFunc:        plainBackend,
		frontendFunc:       plainFrontend,
		clientFunc:         plainClient,
	}

	defer tc.setup(t)()
	uris := make(chan string, 1)
	tc.daemon.Handler = http.HandlerFunc(func(wri http.ResponseWriter, req *http.Request) {
		uris <- req.RequestURI
		httpDaemonHandler(wri, req)
	})

	rules, err := BuildRules(`GET,POST,PUT /~foo\+bar\+\x{1F433}/x`)
	if err != nil {
		t.Fatal(err)
	}
	tc.server.Rules = rules

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

	req, err := httpClientAllowedEncodedSlashReq("http", addrs[0].String())
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

	wantedURI := "/~foo+bar+%F0%9F%90%B3/x?foo=bar"
	if uri := <-uris; uri != wantedURI {
		t.Fatalf(`uri = "%s", want "%s"`, uri, wantedURI)
	}

	err = tc.server.Stop()
	if err != nil {
		t.Fatal(err)
	}
}

func TestCetusGuardPlainAllowedQueryReq(t *testing.T) {
	tc := &testCase{
		daemonListenerFunc: tcpDaemonListener,
		daemonFunc:         plainDaemon,
		backendFunc:        plainBackend,
		frontendFunc:       plainFrontend,
		clientFunc:         plainClient,
	}

	defer tc.setup(t)()
	tc.daemon.Handler = http.HandlerFunc(httpDaemonHandler)

	rules, err := BuildRules(`GET,POST,PUT /~foo\+bar\+\x{1F433}\?a=1&b=2`)
	if err != nil {
		t.Fatal(err)
	}
	tc.server.Rules = rules

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

	req, err := httpClientAllowedQueryReq("http", addrs[0].String())
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

	err = tc.server.Stop()
	if err != nil {
		t.Fatal(err)
	}
}

func TestCetusGuardPlainAllowedMalformedQueryReq(t *testing.T) {
	tc := &testCase{
		daemonListenerFunc: tcpDaemonListener,
		daemonFunc:         plainDaemon,
		backendFunc:        plainBackend,
		frontendFunc:       plainFrontend,
		clientFunc:         plainClient,
	}

	defer tc.setup(t)()
	uris := make(chan string, 1)
	tc.daemon.Handler = http.HandlerFunc(func(wri http.ResponseWriter, req *http.Request) {
		uris <- req.RequestURI
		httpDaemonHandler(wri, req)
	})

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

	req, err := httpClientAllowedMalformedQueryReq("http", addrs[0].String())
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

	wantedURI := "/~foo+bar+%F0%9F%90%B3?a=1;b=2"
	if uri := <-uris; uri != wantedURI {
		t.Fatalf(`uri = "%s", want "%s"`, uri, wantedURI)
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

func TestCetusGuardPlainDeniedTraversalReq(t *testing.T) {
	tc := &testCase{
		daemonListenerFunc: tcpDaemonListener,
		daemonFunc:         plainDaemon,
		backendFunc:        plainBackend,
		frontendFunc:       plainFrontend,
		clientFunc:         plainClient,
	}

	defer tc.setup(t)()
	tc.daemon.Handler = http.HandlerFunc(httpDaemonHandler)

	rules, err := BuildRules(`GET,POST,PUT /~foo\+bar\+\x{1F433}/.*`)
	if err != nil {
		t.Fatal(err)
	}
	tc.server.Rules = rules

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

	req, err := httpClientDeniedTraversalReq("http", addrs[0].String())
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

func TestCetusGuardPlainDeniedOpaqueReq(t *testing.T) {
	tc := &testCase{
		daemonListenerFunc: tcpDaemonListener,
		daemonFunc:         plainDaemon,
		backendFunc:        plainBackend,
		frontendFunc:       plainFrontend,
		clientFunc:         plainClient,
	}

	defer tc.setup(t)()
	tc.daemon.Handler = http.HandlerFunc(httpDaemonHandler)

	rules, err := BuildRules(`GET,POST,PUT /`)
	if err != nil {
		t.Fatal(err)
	}
	tc.server.Rules = rules

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

	req, err := httpClientDeniedOpaqueReq("http", addrs[0].String())
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

func TestCetusGuardPlainDeniedFormReq(t *testing.T) {
	tc := &testCase{
		daemonListenerFunc: tcpDaemonListener,
		daemonFunc:         plainDaemon,
		backendFunc:        plainBackend,
		frontendFunc:       plainFrontend,
		clientFunc:         plainClient,
	}

	defer tc.setup(t)()
	tc.daemon.Handler = http.HandlerFunc(httpDaemonHandler)

	rules, err := BuildRules(`GET,POST,PUT /~foo\+bar\+\x{1F433}\?a=1&b=2`)
	if err != nil {
		t.Fatal(err)
	}
	tc.server.Rules = rules

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

	req, err := httpClientDeniedFormReq("http", addrs[0].String())
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

func TestCetusGuardPlainTLSAuthBackendReq(t *testing.T) {
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

func TestCetusGuardTLSAllowedReq(t *testing.T) {
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

func TestCetusGuardTLSDeniedMethodReq(t *testing.T) {
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

func TestCetusGuardTLSDeniedPatternReq(t *testing.T) {
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

func TestCetusGuardTLSAuthAllowedReq(t *testing.T) {
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

func TestCetusGuardTLSAuthAllowedStreamReq(t *testing.T) {
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

func TestCetusGuardTLSAuthDeniedMethodReq(t *testing.T) {
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

func TestCetusGuardTLSAuthDeniedPatternReq(t *testing.T) {
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

func TestCetusGuardTLSAuthPlainBackendReq(t *testing.T) {
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
		daemonFunc:         expiredTLSDaemon,
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
		daemonFunc:         altTLSDaemon,
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
		clientFunc:         altTLSAuthClient,
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
		backendFunc:        invalidCacertTLSBackend,
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
		backendFunc:        invalidCertTLSAuthBackend,
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
		frontendFunc:       invalidCacertTLSAuthFrontend,
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
		frontendFunc:       invalidCertTLSFrontend,
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

func TestCanonicalizeValidQuery(t *testing.T) {
	testCases := map[string]string{
		"":            "",
		"a=1":         "a=1",
		"b=2&a=1":     "a=1&b=2",
		"b=1&B=1&a=1": "B=1&a=1&b=1",
		"a=2&a=1":     "a=2&a=1",
		"a=%31":       "a=1",
		"a=%2f":       "a=%2F",
		"a=x/y:z":     "a=x%2Fy%3Az",
		"a=x%20y":     "a=x+y",
		"a=x+y":       "a=x+y",
		"a=-._~*":     "a=-._~%2A",
		"a":           "a=",
	}

	for input, wanted := range testCases {
		if result, ok := canonicalizeQuery(input); !ok || result != wanted {
			t.Errorf(`"%s" = "%s", %t, want "%s", true`, input, result, ok, wanted)
		}
	}
}

func TestCanonicalizeInvalidQuery(t *testing.T) {
	testCases := []string{
		"a=1;b=2",
		"a=%2",
	}

	for _, input := range testCases {
		if result, ok := canonicalizeQuery(input); ok {
			t.Errorf(`"%s" = "%s", %t, want "", false`, input, result, ok)
		}
	}
}

func TestCanonicalizeEscapedQuery(t *testing.T) {
	for b := range 256 {
		c := byte(b)
		wanted := fmt.Sprintf("%%%02X", c)
		if c == ' ' {
			wanted = "+"
		} else if 'A' <= c && c <= 'Z' || 'a' <= c && c <= 'z' || '0' <= c && c <= '9' || strings.ContainsRune("-._~", rune(c)) {
			wanted = string(c)
		}
		wanted += "=" + wanted

		for _, escaped := range []string{fmt.Sprintf("%%%02x", c), fmt.Sprintf("%%%02X", c)} {
			input := escaped + "=" + escaped
			if result, ok := canonicalizeQuery(input); !ok || result != wanted {
				t.Errorf(`"%s" = "%s", %t, want "%s", true`, input, result, ok, wanted)
			}
		}
	}
}

func TestValidateRequest(t *testing.T) {
	type testReq struct {
		method, uri, contentType string
		contentLength            int64
		allowed                  bool
	}
	testCases := []struct {
		rule string
		reqs []testReq
	}{
		{`GET /x`, []testReq{
			{"GET", "/x", "", 0, true},
			{"GET", "/x?a=1", "", 0, true},
			{"GET", "/x?a=1;b=2", "", 0, true},
			{"GET", "/x?a=1", "application/x-www-form-urlencoded", -1, true},
			{"GET", "/x%3F", "", 0, false},
			{"GET", "/x%3F/y", "", 0, false},
			{"GET", "/x%3Fa=1", "", 0, false},
		}},
		{`GET /x/.*`, []testReq{
			{"GET", "/x/y", "", 0, true},
			{"GET", "//x/./y", "", 0, true},
			{"GET", "/x/../y", "", 0, false},
			{"GET", "/x/%2E%2E/y", "", 0, false},
		}},
		{`GET /`, []testReq{
			{"GET", "/", "", 0, true},
			{"GET", "x:http://h/x", "", 0, false},
		}},
		{`GET %API_PREFIX_NETWORKS%/%NETWORK_NAME%`, []testReq{
			{"GET", "/networks/foo?a=1", "", 0, true},
			{"GET", "/networks/?a=1", "", 0, false},
			{"GET", "/networks?a=1", "", 0, false},
		}},
		{`GET,POST /x\?a=1`, []testReq{
			{"GET", "/x?a=1", "", 0, true},
			{"GET", "/x?a=%31", "", 0, true},
			{"GET", "/x", "", 0, false},
			{"GET", "/x?a=2", "", 0, false},
			{"GET", "/x?a=1&b=2", "", 0, false},
			{"GET", "/x%3Fa=1", "", 0, false},
			{"POST", "/x?a=1", "application/json", 2, true},
			{"POST", "/x?a=1", "application/x-www-form-urlencoded", 0, true},
			{"POST", "/x?a=1", "application/x-www-form-urlencoded", 3, false},
			{"POST", "/x?a=1", "application/x-www-form-urlencoded", -1, false},
			{"POST", "/x?a=1", "Application/X-WWW-Form-Urlencoded; charset=utf-8", 3, false},
			{"POST", "/x?a=1", "application/x-www-form-urlencoded; =invalid", 3, false},
			{"POST", "/x?a=1", "multipart/form-data; boundary=x", 3, false},
			{"GET", "/x?a=1", "application/x-www-form-urlencoded", 3, false},
			{"GET", "/x?a=1", "multipart/form-data; boundary=x", 3, false},
		}},
		{`GET /n/%NETWORK_NAME%(?:\?a=1)?`, []testReq{
			{"GET", "/n/x", "", 0, true},
			{"GET", "/n/x?a=1", "", 0, true},
			{"GET", "/n/x?b=1", "", 0, false},
			{"GET", "/n/x?a=1;b=2", "", 0, false},
			{"GET", "/n/x?a=1&b=%zz", "", 0, false},
			{"GET", "/n/?b=1", "", 0, false},
		}},
		{`GET /a\?x=1|/b\?y=1`, []testReq{
			{"GET", "/a?x=1", "", 0, true},
			{"GET", "/b?y=1", "", 0, true},
			{"GET", "/a?x=1&z=1", "", 0, false},
			{"GET", "/c/b?y=1", "", 0, false},
		}},
	}

	for _, c := range testCases {
		rules, err := BuildRules(c.rule)
		if err != nil {
			t.Fatal(err)
		}
		cg := &Server{Rules: rules}

		for _, tr := range c.reqs {
			u, err := url.ParseRequestURI(tr.uri)
			if err != nil {
				t.Fatal(err)
			}
			req := &http.Request{Method: tr.method, URL: u, Header: http.Header{}, ContentLength: tr.contentLength}
			if tr.contentType != "" {
				req.Header.Set("Content-Type", tr.contentType)
			}
			if result := cg.validateRequest(req); result != tr.allowed {
				t.Errorf(`rule "%s", %s %s (Content-Type "%s", Content-Length %d) = %t, want %t`, c.rule, tr.method, tr.uri, tr.contentType, tr.contentLength, result, tr.allowed)
			}
		}
	}
}

func FuzzForwardedRequest(f *testing.F) {
	buildRules := func(rule string) []Rule {
		rules, err := BuildRules(rule)
		if err != nil {
			f.Fatal(err)
		}
		return rules
	}
	pathRules := buildRules(`X (?s)/.*`)
	queryRules := buildRules(`X (?s)/.*(?:\?.*)?`)
	scopedRules := buildRules(`X /n/%NETWORK_NAME%(?:\?a=1)?|/m\?b=2`)

	methodRegex := regexp.MustCompile(`^[A-Z]+$`)
	headerNameRegex := regexp.MustCompile("^[!#$%&'*+.^_`|~0-9A-Za-z-]+$")
	validHeaders := func(headers ...http.Header) bool {
		for _, header := range headers {
			for k := range header {
				if !headerNameRegex.MatchString(k) {
					return false
				}
			}
		}
		return true
	}
	allows := func(rules []Rule, req *http.Request) bool {
		methodRules := make([]Rule, len(rules))
		for i, rule := range rules {
			rule.Methods = map[string]struct{}{req.Method: {}}
			methodRules[i] = rule
		}
		return (&Server{Rules: methodRules}).validateRequest(req)
	}
	readRequest := func(b []byte) (*http.Request, error) {
		return http.ReadRequest(bufio.NewReader(bytes.NewReader(b)))
	}
	hasBodyParams := func(contentType string, body []byte) bool {
		req := &http.Request{Method: "POST", Header: http.Header{"Content-Type": {contentType}}, Body: io.NopCloser(bytes.NewReader(body))}
		_ = req.ParseMultipartForm(32 << 20)
		return len(req.PostForm) > 0 || req.MultipartForm != nil && len(req.MultipartForm.File) > 0
	}

	multipart := "--b\r\nContent-Disposition: form-data; name=\"b\"\r\n\r\n2\r\n--b--\r\n"
	seeds := []string{
		"GET /n/x?a=1 HTTP/1.1\r\nHost: h\r\n\r\n",
		"GET /n/x?b=2&a=%31 HTTP/1.1\r\nHost: h\r\n\r\n",
		"GET /n/x?a=1&b=2 HTTP/1.1\r\nHost: h\r\n\r\n",
		"GET /n/x/y?a=1 HTTP/1.1\r\nHost: h\r\n\r\n",
		"GET /z/m?b=2 HTTP/1.1\r\nHost: h\r\n\r\n",
		"DELETE //n/./y/../x/ HTTP/1.1\r\nHost: h\r\n\r\n",
		"GET http://h/n/x?a=1 HTTP/1.1\r\nHost: h\r\n\r\n",
		"GET x:http://h/n/x?a=1 HTTP/1.1\r\nHost: h\r\n\r\n",
		"GET /n/x%2Fy/%2E%2E/%2e%2e/z HTTP/1.1\r\nHost: h\r\n\r\n",
		"GET /n/x%2Fy?a=1 HTTP/1.1\r\nHost: h\r\n\r\n",
		"GET /n/x%3Fa=1 HTTP/1.1\r\nHost: h\r\n\r\n",
		"GET /n/x?a=1;b=2&c=3 HTTP/1.1\r\nHost: h\r\n\r\n",
		"GET /n/x?a=%zz&a=1 HTTP/1.1\r\nHost: h\r\n\r\n",
		"CONNECT h:80 HTTP/1.1\r\nHost: h\r\n\r\n",
		"POST /n/x?a=1 HTTP/1.1\r\nHost: h\r\nContent-Type: application/json\r\nContent-Length: 7\r\n\r\n{\"b\":2}",
		"POST /n/x?a=1 HTTP/1.1\r\nHost: h\r\nContent-Type: application/json\r\nContent-Length: 9\r\n\r\n{}",
		"POST /n/x?a=1 HTTP/1.1\r\nHost: h\r\nContent-Type: application/x-www-form-urlencoded\r\nContent-Length: 3\r\n\r\nb=2",
		"POST /n/x?a=1 HTTP/1.1\r\nHost: h\r\nContent-Type:\r\n application/x-www-form-urlencoded\r\nContent-Length: 3\r\n\r\nb=2",
		"POST /n/x?a=1 HTTP/1.1\r\nHost: h\r\nTrailer: X\r\nContent-Type: application/x-www-form-urlencoded\r\nTransfer-Encoding: chunked\r\n\r\n3;e=1\r\nb=2\r\n0\r\nX: 1\r\n\r\n",
		"POST /n/x?a=1 HTTP/1.1\r\nHost: h\r\nContent-Type: application/x-www-form-urlencoded\r\nContent-Length: 0\r\n\r\n",
		"POST /n/x?a=1 HTTP/1.1\r\nHost: h\r\nContent-Type: application/x-www-form-urlencoded\r\nTransfer-Encoding: chunked\r\n\r\n3\r\nb=2\r\n0\r\n\r\n",
		"POST /n/x?a=1 HTTP/1.1\r\nHost: h\r\nContent-Type: application/x-www-form-urlencoded\r\nContent-Length: 0\r\nTransfer-Encoding: chunked\r\n\r\n3\r\nb=2\r\n0\r\n\r\n",
		"POST /n/x?a=1 HTTP/1.0\r\nContent-Type: application/x-www-form-urlencoded\r\nTransfer-Encoding: chunked\r\nContent-Length: 3\r\n\r\nb=2",
		"PUT /n/x?a=1 HTTP/1.1\r\nHost: h\r\nContent-Type: text/plain\r\nContent-Type: application/x-www-form-urlencoded\r\nContent-Length: 3\r\n\r\nb=2",
		"PATCH /n/x?a=1 HTTP/1.1\r\nHost: h\r\ncontent-type: Application/X-WWW-Form-URLEncoded ; Charset=UTF-8\r\nContent-Length: 3\r\n\r\nb=2",
		fmt.Sprintf("DELETE /n/x?a=1 HTTP/1.1\r\nHost: h\r\nContent-Type: multipart/form-data; boundary=b\r\nContent-Length: %d\r\n\r\n%s", len(multipart), multipart),
		fmt.Sprintf("GET /n/x?a=1 HTTP/1.1\r\nHost: h\r\nContent-Type: multipart/form-data; boundary=b\r\nTransfer-Encoding: chunked\r\n\r\n%x\r\n%s\r\n0\r\n\r\n", len(multipart), multipart),
	}
	for _, seed := range seeds {
		f.Add([]byte(seed))
	}

	f.Fuzz(func(t *testing.T, raw []byte) {
		fe, err := readRequest(raw)
		if err != nil || fe.ProtoMajor != 1 || !methodRegex.MatchString(fe.Method) || !validHeaders(fe.Header, fe.Trailer) {
			return
		}

		req := normalizeRequest(fe)
		pathOk := allows(pathRules, req)
		queryOk := allows(queryRules, req)
		scopedOk := allows(scopedRules, req)
		if fe.URL.Opaque != "" {
			if pathOk || queryOk || scopedOk {
				t.Fatalf(`opaque request-target allowed: %q`, raw)
			}
			return
		}
		if !pathOk {
			t.Fatalf(`allow-all path rule denied the request: %q`, raw)
		}
		values, err := url.ParseQuery(fe.URL.RawQuery)
		parsable := err == nil
		if !queryOk && parsable && fe.ContentLength == 0 && !strings.Contains(fe.URL.Path, "?") {
			t.Fatalf(`allow-all query rule denied a bodiless request with a parsable query: %q`, raw)
		}

		out := req.Clone(context.Background())
		out.URL.Scheme = "http"
		out.URL.Host = "backend"
		var fwd bytes.Buffer
		writeErr := out.Write(&fwd)
		be, err := readRequest(fwd.Bytes())
		if err != nil {
			if writeErr == nil {
				t.Fatalf(`backend rejected the forwarded request (%v): %q`, err, raw)
			}
			return
		}
		beBody, _ := io.ReadAll(be.Body)

		if be.Method != fe.Method {
			t.Fatalf(`backend method "%s", want "%s": %q`, be.Method, fe.Method, raw)
		}
		if wantedPath := cleanPath(fe.URL.Path); be.URL.Path != wantedPath || be.URL.RawPath != "" {
			t.Fatalf(`backend path "%s" (raw "%s"), want "%s" (raw ""): %q`, be.URL.Path, be.URL.RawPath, wantedPath, raw)
		}
		wantedQuery := fe.URL.RawQuery
		if parsable {
			wantedQuery = values.Encode()
		}
		if be.URL.RawQuery != wantedQuery {
			t.Fatalf(`backend query "%s", want "%s": %q`, be.URL.RawQuery, wantedQuery, raw)
		}

		seg, isN := strings.CutPrefix(be.URL.Path, "/n/")
		inScope := be.URL.Path == "/m" && be.URL.RawQuery == "b=2" ||
			isN && seg != "" && !strings.ContainsAny(seg, "/?") && (be.URL.RawQuery == "" || be.URL.RawQuery == "a=1")
		if scopedOk != (queryOk && inScope) {
			t.Fatalf(`scoped query rule = %t, want %t: %q`, scopedOk, queryOk && inScope, raw)
		}

		if !queryOk {
			return
		}
		if !parsable || strings.Contains(be.URL.Path, "?") {
			t.Fatalf(`allow-all query rule allowed an ambiguous target: %q`, raw)
		}
		for _, ct := range slices.Compact(slices.Sorted(slices.Values(be.Header.Values("Content-Type")))) {
			if hasBodyParams(ct, beBody) {
				t.Fatalf(`backend honoring Content-Type "%s" sees body parameters: %q`, ct, raw)
			}
		}
	})
}

func httpClientAllowedReq(scheme string, addr string) (*http.Request, error) {
	body := strings.NewReader("PING")
	req, err := http.NewRequest("POST", "/~foo+bar+%F0%9F%90%B3?foo=bar", body)
	if err != nil {
		return nil, err
	}
	req.URL.Scheme = scheme
	req.URL.Host = addr

	if strings.HasPrefix(addr, "/") {
		req.Host = "test.cetusguard.localhost"
	}

	return req, nil
}

func httpClientAllowedNormalizedReq(scheme string, addr string) (*http.Request, error) {
	body := strings.NewReader("PING")
	req, err := http.NewRequest("POST", "/x/../~foo+bar+%F0%9F%90%B3?b=2&a=%31", body)
	if err != nil {
		return nil, err
	}
	req.URL.Scheme = scheme
	req.URL.Host = addr

	if strings.HasPrefix(addr, "/") {
		req.Host = "test.cetusguard.localhost"
	}

	return req, nil
}

func httpClientAllowedEncodedSlashReq(scheme string, addr string) (*http.Request, error) {
	body := strings.NewReader("PING")
	req, err := http.NewRequest("POST", "/~foo+bar+%F0%9F%90%B3%2Fx?foo=bar", body)
	if err != nil {
		return nil, err
	}
	req.URL.Scheme = scheme
	req.URL.Host = addr

	if strings.HasPrefix(addr, "/") {
		req.Host = "test.cetusguard.localhost"
	}

	return req, nil
}

func httpClientAllowedQueryReq(scheme string, addr string) (*http.Request, error) {
	body := strings.NewReader("PING")
	req, err := http.NewRequest("POST", "/~foo+bar+%F0%9F%90%B3?b=2&a=%31", body)
	if err != nil {
		return nil, err
	}
	req.URL.Scheme = scheme
	req.URL.Host = addr

	if strings.HasPrefix(addr, "/") {
		req.Host = "test.cetusguard.localhost"
	}

	return req, nil
}

func httpClientAllowedMalformedQueryReq(scheme string, addr string) (*http.Request, error) {
	body := strings.NewReader("PING")
	req, err := http.NewRequest("POST", "/~foo+bar+%F0%9F%90%B3?a=1;b=2", body)
	if err != nil {
		return nil, err
	}
	req.URL.Scheme = scheme
	req.URL.Host = addr

	if strings.HasPrefix(addr, "/") {
		req.Host = "test.cetusguard.localhost"
	}

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

	if strings.HasPrefix(addr, "/") {
		req.Host = "test.cetusguard.localhost"
	}

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

	if strings.HasPrefix(addr, "/") {
		req.Host = "test.cetusguard.localhost"
	}

	return req, nil
}

func httpClientDeniedTraversalReq(scheme string, addr string) (*http.Request, error) {
	body := strings.NewReader("PING")
	req, err := http.NewRequest("POST", "/~foo+bar+%F0%9F%90%B3/../x?foo=bar", body)
	if err != nil {
		return nil, err
	}
	req.URL.Scheme = scheme
	req.URL.Host = addr

	if strings.HasPrefix(addr, "/") {
		req.Host = "test.cetusguard.localhost"
	}

	return req, nil
}

func httpClientDeniedOpaqueReq(scheme string, addr string) (*http.Request, error) {
	body := strings.NewReader("PING")
	req, err := http.NewRequest("POST", "/", body)
	if err != nil {
		return nil, err
	}
	req.URL.Scheme = scheme
	req.URL.Host = addr
	req.URL.Opaque = "x:http://test.cetusguard.localhost/~foo+bar+%F0%9F%90%B3"

	if strings.HasPrefix(addr, "/") {
		req.Host = "test.cetusguard.localhost"
	}

	return req, nil
}

func httpClientDeniedFormReq(scheme string, addr string) (*http.Request, error) {
	body := strings.NewReader("PING")
	req, err := http.NewRequest("POST", "/~foo+bar+%F0%9F%90%B3?a=1&b=2", body)
	if err != nil {
		return nil, err
	}
	req.URL.Scheme = scheme
	req.URL.Host = addr
	req.Header.Add("Content-Type", "text/plain")
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	if strings.HasPrefix(addr, "/") {
		req.Host = "test.cetusguard.localhost"
	}

	return req, nil
}

func httpDaemonHandler(wri http.ResponseWriter, req *http.Request) {
	b, err := io.ReadAll(req.Body)
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

		_, err = brw.Write([]byte("HTTP/1.1 101 UPGRADED\r\n" +
			"Connection: Upgrade\r\n" +
			"Upgrade: tcp\r\n" +
			"Content-Type: application/vnd.docker.raw-stream\r\n" +
			"\r\n" +
			"PONG"))
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

	cert, err := tls.X509KeyPair(testdata.TestTLSServerCert, testdata.TestTLSServerKey)
	if err != nil {
		return nil, err
	}
	server.TLSConfig = &tls.Config{
		MinVersion:   minTLSVersion,
		Certificates: []tls.Certificate{cert},
	}

	return server, nil
}

func expiredTLSDaemon() (*http.Server, error) {
	server, err := plainDaemon()
	if err != nil {
		return nil, err
	}

	cert, err := tls.X509KeyPair(testdata.TestTLSServerExpiredCert, testdata.TestTLSServerKey)
	if err != nil {
		return nil, err
	}
	server.TLSConfig = &tls.Config{
		MinVersion:   minTLSVersion,
		Certificates: []tls.Certificate{cert},
	}

	return server, nil
}

func altTLSDaemon() (*http.Server, error) {
	server, err := plainDaemon()
	if err != nil {
		return nil, err
	}

	cert, err := tls.X509KeyPair(testdata.TestAltTLSServerCert, testdata.TestAltTLSServerKey)
	if err != nil {
		return nil, err
	}
	server.TLSConfig = &tls.Config{
		MinVersion:   minTLSVersion,
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
	if ok := cacertPool.AppendCertsFromPEM(testdata.TestTLSCacert); !ok {
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
	if err := os.WriteFile(clientCacertPath, testdata.TestTLSCacert, 0600); err != nil {
		return nil, err
	}
	backend.TLSCacert = clientCacertPath

	return backend, nil
}

func invalidCacertTLSBackend(listener net.Listener, tmpdir string) (*Backend, error) {
	backend, err := plainBackend(listener, tmpdir)
	if err != nil {
		return nil, err
	}

	clientCacertPath := filepath.Join(tmpdir, "client-ca.pem")
	if err := os.WriteFile(clientCacertPath, testdata.TestInvalidTLSCacert, 0600); err != nil {
		return nil, err
	}
	backend.TLSCacert = clientCacertPath

	return backend, nil
}

func tlsAuthBackend(listener net.Listener, tmpdir string) (*Backend, error) {
	backend, err := tlsBackend(listener, tmpdir)
	if err != nil {
		return nil, err
	}

	clientCertPath := filepath.Join(tmpdir, "client-cert.pem")
	if err := os.WriteFile(clientCertPath, testdata.TestTLSClientCert, 0600); err != nil {
		return nil, err
	}
	backend.TLSCert = clientCertPath

	clientKeyPath := filepath.Join(tmpdir, "client-key.pem")
	if err := os.WriteFile(clientKeyPath, testdata.TestTLSClientKey, 0600); err != nil {
		return nil, err
	}
	backend.TLSKey = clientKeyPath

	return backend, nil
}

func invalidCertTLSAuthBackend(listener net.Listener, tmpdir string) (*Backend, error) {
	backend, err := tlsBackend(listener, tmpdir)
	if err != nil {
		return nil, err
	}

	clientCertPath := filepath.Join(tmpdir, "client-cert.pem")
	if err := os.WriteFile(clientCertPath, testdata.TestInvalidTLSClientCert, 0600); err != nil {
		return nil, err
	}
	backend.TLSCert = clientCertPath

	clientKeyPath := filepath.Join(tmpdir, "client-key.pem")
	if err := os.WriteFile(clientKeyPath, testdata.TestInvalidTLSClientKey, 0600); err != nil {
		return nil, err
	}
	backend.TLSKey = clientKeyPath

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
	if err := os.WriteFile(serverCertPath, testdata.TestTLSServerCert, 0600); err != nil {
		return nil, err
	}
	frontend.TLSCert = serverCertPath

	serverKeyPath := filepath.Join(tmpdir, "server-key.pem")
	if err := os.WriteFile(serverKeyPath, testdata.TestTLSServerKey, 0600); err != nil {
		return nil, err
	}
	frontend.TLSKey = serverKeyPath

	return frontend, nil
}

func invalidCertTLSFrontend(tmpdir string) (*Frontend, error) {
	frontend, err := plainFrontend(tmpdir)
	if err != nil {
		return nil, err
	}

	serverCertPath := filepath.Join(tmpdir, "server-cert.pem")
	if err := os.WriteFile(serverCertPath, testdata.TestInvalidTLSServerCert, 0600); err != nil {
		return nil, err
	}
	frontend.TLSCert = serverCertPath

	serverKeyPath := filepath.Join(tmpdir, "server-key.pem")
	if err := os.WriteFile(serverKeyPath, testdata.TestInvalidTLSServerKey, 0600); err != nil {
		return nil, err
	}
	frontend.TLSKey = serverKeyPath

	return frontend, nil
}

func tlsAuthFrontend(tmpdir string) (*Frontend, error) {
	frontend, err := tlsFrontend(tmpdir)
	if err != nil {
		return nil, err
	}

	serverCacertPath := filepath.Join(tmpdir, "server-ca.pem")
	if err := os.WriteFile(serverCacertPath, testdata.TestTLSCacert, 0600); err != nil {
		return nil, err
	}
	frontend.TLSCacert = serverCacertPath

	return frontend, nil
}

func invalidCacertTLSAuthFrontend(tmpdir string) (*Frontend, error) {
	frontend, err := tlsFrontend(tmpdir)
	if err != nil {
		return nil, err
	}

	serverCacertPath := filepath.Join(tmpdir, "server-ca.pem")
	if err := os.WriteFile(serverCacertPath, testdata.TestInvalidTLSCacert, 0600); err != nil {
		return nil, err
	}
	frontend.TLSCacert = serverCacertPath

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
	if ok := cacertPool.AppendCertsFromPEM(testdata.TestTLSCacert); !ok {
		return nil, errors.New("error loading CA certificate")
	}
	transport := client.Transport.(*http.Transport)
	transport.TLSClientConfig = &tls.Config{
		MinVersion: minTLSVersion,
		RootCAs:    cacertPool,
		/* #nosec G402 */
		InsecureSkipVerify: true,
	}

	return client, nil
}

func altTLSClient() (*http.Client, error) {
	client, err := plainClient()
	if err != nil {
		return nil, err
	}

	cacertPool := x509.NewCertPool()
	if ok := cacertPool.AppendCertsFromPEM(testdata.TestAltTLSCacert); !ok {
		return nil, errors.New("error loading CA certificate")
	}
	transport := client.Transport.(*http.Transport)
	transport.TLSClientConfig = &tls.Config{
		MinVersion: minTLSVersion,
		RootCAs:    cacertPool,
		/* #nosec G402 */
		InsecureSkipVerify: true,
	}

	return client, nil
}

func tlsAuthClient() (*http.Client, error) {
	client, err := tlsClient()
	if err != nil {
		return nil, err
	}

	cert, err := tls.X509KeyPair(testdata.TestTLSClientCert, testdata.TestTLSClientKey)
	if err != nil {
		return nil, err
	}
	transport := client.Transport.(*http.Transport)
	transport.TLSClientConfig.Certificates = []tls.Certificate{cert}

	return client, nil
}

func altTLSAuthClient() (*http.Client, error) {
	client, err := altTLSClient()
	if err != nil {
		return nil, err
	}

	cert, err := tls.X509KeyPair(testdata.TestAltTLSClientCert, testdata.TestAltTLSClientKey)
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
		if tc.backend.TLSCacert != "" {
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
