// Copyright (c) 2009-present, Alibaba Cloud All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.

package openapi

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/responses"
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/aliyun/aliyun-cli/v3/meta"
)

func toSandboxInvoker(inv Invoker) Invoker {
	switch x := inv.(type) {
	case *RpcInvoker:
		return &SandboxRpcInvoker{BasicInvoker: x.BasicInvoker, api: x.api}
	case *ForceRpcInvoker:
		return &SandboxForceRpcInvoker{BasicInvoker: x.BasicInvoker, method: x.method}
	case *RestfulInvoker:
		return &SandboxRestfulInvoker{rf: x}
	default:
		return sandboxUnsupportedInvoker{typeName: fmt.Sprintf("%T", inv)}
	}
}

// sandboxUnsupportedInvoker 避免未知 Invoker 在 SandboxProxy 下回落为本地直连签名。
type sandboxUnsupportedInvoker struct {
	typeName string
}

func (sandboxUnsupportedInvoker) getClient() *sdk.Client { return nil }

func (sandboxUnsupportedInvoker) getRequest() *requests.CommonRequest { return &requests.CommonRequest{} }

func (u sandboxUnsupportedInvoker) Prepare(*cli.Context) error {
	return fmt.Errorf("sandbox proxy: unsupported invoker type %s (only RPC, force-RPC, and ROA are supported)", u.typeName)
}

func (u sandboxUnsupportedInvoker) Call() (*responses.CommonResponse, error) {
	return nil, fmt.Errorf("sandbox proxy: unsupported invoker type %s", u.typeName)
}

// SandboxRpcInvoker RPC 经代签网关；Prepare 与 RpcInvoker 一致。
type SandboxRpcInvoker struct {
	*BasicInvoker
	api *meta.Api
}

func (a *SandboxRpcInvoker) Prepare(ctx *cli.Context) error {
	inner := RpcInvoker{BasicInvoker: a.BasicInvoker, api: a.api}
	return inner.Prepare(ctx)
}

func (a *SandboxRpcInvoker) Call() (*responses.CommonResponse, error) {
	return callSandboxRPC(a.BasicInvoker)
}

// SandboxForceRpcInvoker 与 ForceRpcInvoker 对应。
type SandboxForceRpcInvoker struct {
	*BasicInvoker
	method string
}

func (a *SandboxForceRpcInvoker) Prepare(ctx *cli.Context) error {
	inner := ForceRpcInvoker{BasicInvoker: a.BasicInvoker, method: a.method}
	return inner.Prepare(ctx)
}

func (a *SandboxForceRpcInvoker) Call() (*responses.CommonResponse, error) {
	return callSandboxRPC(a.BasicInvoker)
}

// SandboxRestfulInvoker ROA 经代签网关。
type SandboxRestfulInvoker struct {
	rf *RestfulInvoker
}

func (a *SandboxRestfulInvoker) getClient() *sdk.Client {
	return nil
}

func (a *SandboxRestfulInvoker) getRequest() *requests.CommonRequest {
	return a.rf.getRequest()
}

func (a *SandboxRestfulInvoker) Prepare(ctx *cli.Context) error {
	return a.rf.Prepare(ctx)
}

func (a *SandboxRestfulInvoker) Call() (*responses.CommonResponse, error) {
	return callSandboxROA(a.rf.BasicInvoker, a.rf.method, a.rf.path)
}

func callSandboxRPC(b *BasicInvoker) (*responses.CommonResponse, error) {
	method := strings.ToUpper(strings.TrimSpace(b.request.Method))
	if method == "" {
		method = http.MethodPost
	}
	form := buildSandboxRPCForm(b.request)
	var body io.Reader
	urlSuffix := ""
	if method == http.MethodGet {
		if form != "" {
			urlSuffix = "?" + form
		}
	} else {
		body = strings.NewReader(form)
	}
	return doSandboxHTTP(b.profile, b.request, b.product, method, body, urlSuffix, true, "", pickRPCContentType(method, form))
}

func pickRPCContentType(method, form string) string {
	if method == http.MethodGet {
		return ""
	}
	if form != "" {
		return "application/x-www-form-urlencoded"
	}
	return ""
}

func callSandboxROA(b *BasicInvoker, method, roaPath string) (*responses.CommonResponse, error) {
	m := strings.ToUpper(strings.TrimSpace(method))
	if m == "" {
		m = http.MethodGet
	}
	ct, _ := b.request.GetContentType()
	if ct == "" && len(b.request.Content) > 0 {
		if bytes.HasPrefix(b.request.Content, []byte("{")) {
			ct = "application/json"
		}
	}
	var body io.Reader
	if len(b.request.Content) > 0 {
		body = bytes.NewReader(b.request.Content)
	}
	return doSandboxHTTP(b.profile, b.request, b.product, m, body, "", false, roaPath, ct)
}

func buildSandboxRPCForm(req *requests.CommonRequest) string {
	v := url.Values{}
	if req.ApiName != "" {
		v.Set("Action", req.ApiName)
	}
	if req.Version != "" {
		v.Set("Version", req.Version)
	}
	if req.RegionId != "" {
		v.Set("RegionId", req.RegionId)
	}
	for k, val := range req.QueryParams {
		v.Set(k, val)
	}
	for k, val := range req.FormParams {
		v.Set(k, val)
	}
	return v.Encode()
}

func sandboxOpenAPIPath() string {
	if p := strings.TrimSpace(os.Getenv(config.EnvSandboxProxyOpenAPIPath)); p != "" {
		if !strings.HasPrefix(p, "/") {
			return "/" + p
		}
		return p
	}
	return "/api/v1/openapi"
}

func doSandboxHTTP(profile *config.Profile, creq *requests.CommonRequest, product *meta.Product,
	method string, body io.Reader, urlSuffix string, isRPC bool, roaPath, contentType string,
) (*responses.CommonResponse, error) {
	base := strings.TrimSuffix(strings.TrimSpace(profile.SandboxProxyUrl), "/")
	target := base + sandboxOpenAPIPath() + urlSuffix

	httpReq, err := http.NewRequest(method, target, body)
	if err != nil {
		return nil, err
	}
	if contentType != "" {
		httpReq.Header.Set("Content-Type", contentType)
	}
	if tok := strings.TrimSpace(profile.SandboxProxyToken); tok != "" {
		httpReq.Header.Set("Authorization", "Bearer "+tok)
	}
	lc := strings.ToLower(strings.TrimSpace(product.Code))
	httpReq.Header.Set("X-Acs-Product", lc)
	httpReq.Header.Set("X-Acs-Version", strings.TrimSpace(creq.Version))
	httpReq.Header.Set("X-Acs-Region", strings.TrimSpace(creq.RegionId))
	if isRPC {
		httpReq.Header.Set("X-Acs-Action", strings.TrimSpace(creq.ApiName))
	} else {
		httpReq.Header.Set("X-Acs-Path", strings.TrimSpace(roaPath))
		httpReq.Header.Set("X-Acs-Method", strings.TrimSpace(method))
		if s := strings.TrimSpace(creq.ApiName); s != "" {
			httpReq.Header.Set("X-Acs-Action", s)
		}
	}
	if s := strings.TrimSpace(os.Getenv(config.EnvSandboxProxyIdentityData)); s != "" {
		httpReq.Header.Set("X-Identity-Data", s)
	}
	if s := strings.TrimSpace(os.Getenv(config.EnvSandboxProxyToolCallID)); s != "" {
		httpReq.Header.Set("X-Tool-Call-ID", s)
	}
	if s := strings.TrimSpace(os.Getenv(config.EnvSandboxProxyThreadID)); s != "" {
		httpReq.Header.Set("X-Thread-Id", s)
	}
	httpReq.Header.Set("User-Agent", "Aliyun-CLI/"+cli.GetVersion())

	connectTO := 30 * time.Second
	if profile.ConnectTimeout > 0 {
		connectTO = time.Duration(profile.ConnectTimeout) * time.Second
	}
	tr, err := newSandboxTransport(profile, connectTO)
	if err != nil {
		return nil, err
	}
	readTO := 120 * time.Second
	if profile.ReadTimeout > 0 {
		readTO = time.Duration(profile.ReadTimeout) * time.Second
	}
	client := &http.Client{Transport: tr, Timeout: readTO}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return httpResponseToCommon(resp)
}

func newSandboxTransport(profile *config.Profile, connectTimeout time.Duration) (*http.Transport, error) {
	if connectTimeout <= 0 {
		connectTimeout = 30 * time.Second
	}
	tlsCfg := &tls.Config{MinVersion: tls.VersionTLS12}
	if profile.SandboxProxyInsecure {
		tlsCfg.InsecureSkipVerify = true
	}
	certFile := strings.TrimSpace(os.Getenv(config.EnvSandboxProxyClientCert))
	keyFile := strings.TrimSpace(os.Getenv(config.EnvSandboxProxyClientKey))
	if certFile != "" && keyFile != "" {
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			return nil, fmt.Errorf("load sandbox proxy client cert: %w", err)
		}
		tlsCfg.Certificates = []tls.Certificate{cert}
	}
	if caFile := strings.TrimSpace(os.Getenv(config.EnvSandboxProxyCAFile)); caFile != "" {
		b, err := os.ReadFile(caFile)
		if err != nil {
			return nil, fmt.Errorf("read CA bundle: %w", err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(b) {
			return nil, fmt.Errorf("parse CA bundle %s", caFile)
		}
		tlsCfg.RootCAs = pool
	}
	dialer := &net.Dialer{Timeout: connectTimeout}
	return &http.Transport{
		DialContext:         dialer.DialContext,
		TLSClientConfig:     tlsCfg,
		TLSHandshakeTimeout: connectTimeout,
		ForceAttemptHTTP2:   true,
	}, nil
}

func httpResponseToCommon(resp *http.Response) (*responses.CommonResponse, error) {
	out := responses.NewCommonResponse()
	err := responses.Unmarshal(out, resp, "JSON")
	return out, err
}
