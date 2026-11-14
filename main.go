// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Drivectl is a command-line interface and MCP server for Google Drive, Docs, and Sheets.
package main

import (
	"context"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/ghchinoy/drivectl/cmd"
	"github.com/ghchinoy/drivectl/internal/ui"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	err := run(ctx, os.Args[1:])
	ui.HandleError(err)
}

func run(ctx context.Context, args []string) error {
	mcpMode, mcpHTTP := parseMCPFlags(args)
	if mcpMode || mcpHTTP != "" {
		return cmd.ExecuteMCP(ctx, mcpHTTP)
	}
	return cmd.ExecuteContext(ctx)
}

func parseMCPFlags(args []string) (bool, string) {
	for i, arg := range args {
		if arg == "--mcp" {
			return true, ""
		}
		if strings.HasPrefix(arg, "--mcp-http") {
			if _, addr, ok := strings.Cut(arg, "="); ok {
				return false, addr
			}
			if i+1 < len(args) {
				return false, args[i+1]
			}
			return false, ""
		}
	}
	return false, ""
}
