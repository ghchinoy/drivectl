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

package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/ghchinoy/drivectl/internal/ui"
)

// Recipe defines a sequence of command-line string arguments to pass to the CLI.
type Recipe struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Steps       [][]string `json:"steps"`
}

var runCmd = &cobra.Command{
	Use:     "run [recipe-file]",
	GroupID: GroupAdvanced,
	Short:   "Run a predefined recipe of commands",
	Long:    "Execute a sequence of drivectl commands defined in a JSON recipe file.",
	Example: "  drivectl run ./recipes/backup.json",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		file := args[0]
		b, err := os.ReadFile(file)
		if err != nil {
			wrapped := fmt.Errorf("failed to read recipe %q: %w", file, err)
			return ui.ErrorWithHint(wrapped, "Ensure the recipe file exists and you have read permissions.")
		}

		var recipe Recipe
		if err := json.Unmarshal(b, &recipe); err != nil {
			wrapped := fmt.Errorf("invalid recipe format: %w", err)
			return ui.ErrorWithHint(wrapped, "Ensure the file is valid JSON matching the Recipe schema.")
		}

		ui.PrintSuccess("Starting recipe: %s (%s)", recipe.Name, recipe.Description)

		for i, stepArgs := range recipe.Steps {
			fmt.Printf("\n%s Step %d: %v\n", ui.Accent("=>"), i+1, stepArgs)

			rootCmd.SetArgs(stepArgs)
			if err := rootCmd.ExecuteContext(cmd.Context()); err != nil {
				return fmt.Errorf("recipe failed at step %d: %w", i+1, err)
			}
		}

		fmt.Println()
		ui.PrintSuccess("Recipe completed.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
}
