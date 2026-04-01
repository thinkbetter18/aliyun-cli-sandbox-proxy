// Copyright (c) 2009-present, Alibaba Cloud All rights reserved.

package config

import (
	"os"
	"testing"
)

func TestValidateSandboxProxy_tokenRequired(t *testing.T) {
	t.Setenv("ALIYUN_CLI_SANDBOX_PROXY_ALLOW_EMPTY_TOKEN", "")
	p := Profile{
		Name:              "t",
		Mode:              SandboxProxy,
		RegionId:          "cn-hangzhou",
		SandboxProxyUrl:   "https://proxy.example:8445",
		SandboxProxyToken: "",
	}
	if err := p.Validate(); err == nil {
		t.Fatal("expected error when token empty")
	}
}

func TestValidateSandboxProxy_allowEmptyTokenDev(t *testing.T) {
	t.Setenv("ALIYUN_CLI_SANDBOX_PROXY_ALLOW_EMPTY_TOKEN", "1")
	t.Cleanup(func() { _ = os.Unsetenv("ALIYUN_CLI_SANDBOX_PROXY_ALLOW_EMPTY_TOKEN") })
	p := Profile{
		Name:            "t",
		Mode:            SandboxProxy,
		RegionId:        "cn-hangzhou",
		SandboxProxyUrl: "https://proxy.example:8445",
	}
	if err := p.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestValidateSandboxProxy_urlRequired(t *testing.T) {
	t.Setenv("ALIYUN_CLI_SANDBOX_PROXY_ALLOW_EMPTY_TOKEN", "1")
	t.Cleanup(func() { _ = os.Unsetenv("ALIYUN_CLI_SANDBOX_PROXY_ALLOW_EMPTY_TOKEN") })
	p := Profile{
		Name:            "t",
		Mode:            SandboxProxy,
		RegionId:        "cn-hangzhou",
		SandboxProxyUrl: "",
	}
	if err := p.Validate(); err == nil {
		t.Fatal("expected error when url empty")
	}
}

func TestValidateSandboxProxy_rejectsAutoPluginInstall(t *testing.T) {
	t.Setenv("ALIYUN_CLI_SANDBOX_PROXY_ALLOW_EMPTY_TOKEN", "1")
	t.Cleanup(func() { _ = os.Unsetenv("ALIYUN_CLI_SANDBOX_PROXY_ALLOW_EMPTY_TOKEN") })
	p := Profile{
		Name:              "t",
		Mode:              SandboxProxy,
		RegionId:          "cn-hangzhou",
		SandboxProxyUrl:   "https://proxy.example:8445",
		AutoPluginInstall: true,
	}
	if err := p.Validate(); err == nil {
		t.Fatal("expected error when auto_plugin_install in SandboxProxy mode")
	}
}

func TestValidateSandboxProxy_insecureRequiresExplicitEnv(t *testing.T) {
	t.Setenv("ALIYUN_CLI_SANDBOX_PROXY_ALLOW_EMPTY_TOKEN", "1")
	t.Setenv(EnvAllowInsecureTLS, "")
	t.Cleanup(func() {
		_ = os.Unsetenv("ALIYUN_CLI_SANDBOX_PROXY_ALLOW_EMPTY_TOKEN")
		_ = os.Unsetenv(EnvAllowInsecureTLS)
	})
	p := Profile{
		Name:                 "t",
		Mode:                 SandboxProxy,
		RegionId:             "cn-hangzhou",
		SandboxProxyUrl:      "https://proxy.example:8445",
		SandboxProxyInsecure: true,
	}
	if err := p.Validate(); err == nil {
		t.Fatal("expected error when insecure without opt-in env")
	}
}

func TestValidateSandboxProxy_insecureAllowedWithEnv(t *testing.T) {
	t.Setenv("ALIYUN_CLI_SANDBOX_PROXY_ALLOW_EMPTY_TOKEN", "1")
	t.Setenv(EnvAllowInsecureTLS, "1")
	t.Cleanup(func() {
		_ = os.Unsetenv("ALIYUN_CLI_SANDBOX_PROXY_ALLOW_EMPTY_TOKEN")
		_ = os.Unsetenv(EnvAllowInsecureTLS)
	})
	p := Profile{
		Name:                 "t",
		Mode:                 SandboxProxy,
		RegionId:             "cn-hangzhou",
		SandboxProxyUrl:      "https://proxy.example:8445",
		SandboxProxyInsecure: true,
	}
	if err := p.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestGetCredential_SandboxProxy(t *testing.T) {
	p := Profile{
		Name:     "t",
		Mode:     SandboxProxy,
		RegionId: "cn-hangzhou",
	}
	_, err := p.GetCredential(nil, nil)
	if err == nil {
		t.Fatal("expected error")
	}
}
