package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/ghchinoy/drivectl/internal/ui"
	"github.com/spf13/cobra"
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
			return ui.ErrorWithHint(fmt.Errorf("failed to read recipe: %w", err), "Ensure the recipe file exists and you have read permissions.")
		}

		var recipe Recipe
		if err := json.Unmarshal(b, &recipe); err != nil {
			return ui.ErrorWithHint(fmt.Errorf("invalid recipe format: %w", err), "Ensure the file is valid JSON matching the Recipe schema.")
		}

		ui.PrintSuccess("Starting recipe: %s (%s)", recipe.Name, recipe.Description)
		
		for i, stepArgs := range recipe.Steps {
			fmt.Printf("\n%s Step %d: %v\n", ui.Accent("=>"), i+1, stepArgs)
			
			// Dynamically set args and execute the root command for this step
			rootCmd.SetArgs(stepArgs)
			if err := rootCmd.Execute(); err != nil {
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
