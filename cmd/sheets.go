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
	"strings"
	"text/tabwriter"

	"github.com/ghchinoy/drivectl/internal/drive"
	"github.com/ghchinoy/drivectl/internal/ui"
	"github.com/spf13/cobra"
)

var (
	sheetName        string
	sheetRange       string
	sheetsOutputFile string
)

var sheetsCmd = &cobra.Command{
	Use:     "sheets",
	GroupID: GroupIntegration,
	Short:   "Interact with Google Sheets",
	Long:    `A set of commands to interact with Google Sheets.`,
}

var sheetsListCmd = &cobra.Command{
	Use:     "list [spreadsheetId]",
	Short:   "Lists the sheets in a spreadsheet.",
	Long:    `Lists all the individual sheets (tabs) within a given spreadsheet.`,
	Example: `  drivectl sheets list <spreadsheet-id>`,
	Args:    cobra.ExactArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		spreadsheetID := args[0]
		sheets, err := drive.ListSheets(sheetsSvc, spreadsheetID)
		if err != nil {
			return ui.ErrorWithHint(err, "Ensure the spreadsheet ID is correct.")
		}

		if OutputFormat == "json" {
			b, err := json.MarshalIndent(sheets, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(b))
			return nil
		}

		fmt.Println(ui.Accent("Sheets:"))
		for _, sheet := range sheets {
			fmt.Println(sheet)
		}
		return nil
	},
}

var sheetsGetCmd = &cobra.Command{
	Use:     "get [spreadsheetId]",
	Short:   "Gets a sheet as CSV.",
	Long:    `Retrieves the entire content of a specified sheet and outputs it as CSV. An optional output file can be specified.`,
	Example: `  drivectl sheets get <spreadsheet-id> --sheet Sheet1`,
	Args:    cobra.ExactArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		spreadsheetID := args[0]
		csv, err := drive.FetchSheetAsCSV(sheetsSvc, spreadsheetID, sheetName)
		if err != nil {
			return ui.ErrorWithHint(err, "Check if the sheet name exists in the given spreadsheet ID.")
		}

		if OutputFormat == "json" {
			res := map[string]any{
				"spreadsheetId": spreadsheetID,
				"sheetName":     sheetName,
				"csv":           csv,
			}
			b, err := json.MarshalIndent(res, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(b))
			return nil
		}

		if sheetsOutputFile != "" {
			if err := os.WriteFile(sheetsOutputFile, []byte(csv), 0o600); err != nil {
				wrapped := fmt.Errorf("failed to write to output file %q: %w", sheetsOutputFile, err)
				return ui.ErrorWithHint(wrapped, "Check file path permissions.")
			}
			ui.PrintSuccess("Saved sheet to %s", sheetsOutputFile)
			return nil
		}

		fmt.Println(csv)
		return nil
	},
}

var sheetsGetRangeCmd = &cobra.Command{
	Use:     "get-range [spreadsheetId]",
	Short:   "Gets a specific range from a sheet.",
	Long:    `Retrieves a specific range of cells from a sheet, specified using A1 notation.`,
	Example: `  drivectl sheets get-range <spreadsheet-id> --sheet Sheet1 --range A1:C10`,
	Args:    cobra.ExactArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		spreadsheetID := args[0]
		values, err := drive.FetchSheetRange(sheetsSvc, spreadsheetID, sheetName, sheetRange)
		if err != nil {
			return ui.ErrorWithHint(err, "Verify the A1 notation of the range (e.g. A1:B2).")
		}

		if OutputFormat == "json" {
			b, err := json.MarshalIndent(values, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(b))
			return nil
		}

		w := new(tabwriter.Writer)
		w.Init(os.Stdout, 0, 8, 0, '\t', 0)
		for _, row := range values {
			var rowStr []string
			for _, cell := range row {
				rowStr = append(rowStr, fmt.Sprint(cell))
			}
			fmt.Fprintln(w, strings.Join(rowStr, "\t"))
		}
		w.Flush()

		return nil
	},
}

var sheetsUpdateRangeCmd = &cobra.Command{
	Use:     "update-range [spreadsheetId] [value]",
	Short:   "Updates a specific range in a sheet.",
	Long:    `Updates a specific cell or range of cells within a sheet, specified using A1 notation.`,
	Example: `  drivectl sheets update-range <spreadsheet-id> "Done" --sheet Sheet1 --range B2`,
	Args:    cobra.ExactArgs(2),
	RunE: func(_ *cobra.Command, args []string) error {
		spreadsheetID := args[0]
		value := args[1]
		values := [][]any{{value}}
		err := drive.UpdateSheetRange(sheetsSvc, spreadsheetID, sheetName, sheetRange, values)
		if err != nil {
			return ui.ErrorWithHint(err, "Verify the A1 notation of the range and that you have write permissions.")
		}

		if OutputFormat == "json" {
			res := map[string]any{
				"status":        "success",
				"spreadsheetId": spreadsheetID,
				"sheetName":     sheetName,
				"range":         sheetRange,
				"updatedValue":  value,
			}
			b, err := json.MarshalIndent(res, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(b))
			return nil
		}

		ui.PrintSuccess("Sheet updated successfully.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(sheetsCmd)
	sheetsCmd.AddCommand(sheetsListCmd)
	sheetsCmd.AddCommand(sheetsGetCmd)
	sheetsCmd.AddCommand(sheetsGetRangeCmd)
	sheetsCmd.AddCommand(sheetsUpdateRangeCmd)

	sheetsGetCmd.Flags().StringVar(&sheetName, "sheet", "", "Name of the sheet to get")
	_ = sheetsGetCmd.MarkFlagRequired("sheet")
	sheetsGetCmd.Flags().StringVarP(&sheetsOutputFile, "output", "o", "", "Path to save the output file")

	sheetsGetRangeCmd.Flags().StringVar(&sheetName, "sheet", "", "Name of the sheet")
	_ = sheetsGetRangeCmd.MarkFlagRequired("sheet")
	sheetsGetRangeCmd.Flags().StringVar(&sheetRange, "range", "", "The A1 notation of the range to retrieve")
	_ = sheetsGetRangeCmd.MarkFlagRequired("range")

	sheetsUpdateRangeCmd.Flags().StringVar(&sheetName, "sheet", "", "Name of the sheet")
	_ = sheetsUpdateRangeCmd.MarkFlagRequired("sheet")
	sheetsUpdateRangeCmd.Flags().StringVar(&sheetRange, "range", "", "The A1 notation of the range to update")
	_ = sheetsUpdateRangeCmd.MarkFlagRequired("range")
}
