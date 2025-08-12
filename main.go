
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

package main

import (
	"os"
	"strings"

	"github.com/ghchinoy/drivectl/cmd"
)

// main is the entry point of the application.
// It checks for the --mcp and --mcp-http flags to determine whether to run in MCP mode.
func main() {
	mcp := false
	mcpHTTP := ""

	for i, arg := range os.Args[1:] {
		if arg == "--mcp" {
			mcp = true
			break
		}
		if strings.HasPrefix(arg, "--mcp-http") {
			parts := strings.SplitN(arg, "=", 2)
			if len(parts) == 2 {
				mcpHTTP = parts[1]
			} else if i+1 < len(os.Args[1:]) {
				mcpHTTP = os.Args[i+2]
			}
			break
		}
	}

	if mcp || mcpHTTP != "" {
		cmd.ExecuteMCP(mcpHTTP)
	} else {
		cmd.Execute()
	}
}
