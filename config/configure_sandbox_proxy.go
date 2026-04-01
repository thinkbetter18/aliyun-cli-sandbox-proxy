// Copyright (c) 2009-present, Alibaba Cloud All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.

package config

import (
	"io"

	"github.com/aliyun/aliyun-cli/v3/cli"
)

func configureSandboxProxy(w io.Writer, cp *Profile) error {
	cli.Printf(w, "Sandbox Proxy Url [%s]: ", cp.SandboxProxyUrl)
	cp.SandboxProxyUrl = ReadInput(cp.SandboxProxyUrl)
	cli.Printf(w, "Sandbox Proxy Token (exec session) [%s]: ", MosaicString(cp.SandboxProxyToken, 3))
	cp.SandboxProxyToken = ReadInput(cp.SandboxProxyToken)
	return nil
}
