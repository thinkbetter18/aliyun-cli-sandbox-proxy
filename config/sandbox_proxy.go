// Copyright (c) 2009-present, Alibaba Cloud All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.

package config

import (
	"os"
	"strings"
)

// §19 D20：Fork 仅识别 ALIBABA_CLOUD_SANDBOX_PROXY_* 前缀（不实现 ALIBABACLOUD_* 别名）。
const (
	EnvSandboxProxyURL          = "ALIBABA_CLOUD_SANDBOX_PROXY_URL"
	EnvSandboxProxyToken        = "ALIBABA_CLOUD_SANDBOX_PROXY_TOKEN"
	EnvSandboxProxyOpenAPIPath  = "ALIBABA_CLOUD_SANDBOX_PROXY_OPENAPI_PATH"
	EnvSandboxProxyIdentityData = "ALIBABA_CLOUD_X_IDENTITY_DATA"
	EnvSandboxProxyToolCallID   = "ALIBABA_CLOUD_TOOL_CALL_ID"
	EnvSandboxProxyThreadID     = "ALIBABA_CLOUD_THREAD_ID"
	EnvSandboxProxyClientCert   = "ALIBABA_CLOUD_SANDBOX_PROXY_CLIENT_CERT_FILE"
	EnvSandboxProxyClientKey    = "ALIBABA_CLOUD_SANDBOX_PROXY_CLIENT_KEY_FILE"
	EnvSandboxProxyCAFile       = "ALIBABA_CLOUD_SANDBOX_PROXY_CA_FILE"
)

// AllowEmptySandboxProxyToken 为 mock/单测：设置 ALIYUN_CLI_SANDBOX_PROXY_ALLOW_EMPTY_TOKEN=1 时允许不配置 exec 令牌。
func AllowEmptySandboxProxyToken() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("ALIYUN_CLI_SANDBOX_PROXY_ALLOW_EMPTY_TOKEN")))
	return v == "1" || v == "true" || v == "yes"
}
