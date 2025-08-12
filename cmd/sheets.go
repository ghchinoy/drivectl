package cmd

import (
	"fmt"
	"os"

	"github.com/ghchinoy/drivectl/internal/drive"
	"github.com/spf13/cobra"
)

var (
	sheetName        string
	sheetRange       string
	sheetsOutputFile string
)

var sheetsCmd = &cobra.Command{
	Use:   "sheets",
	Short: "Interact with Google Sheets",
	Long:  `A set of commands to interact with Google Sheets.`,
}

var sheetsListCmd = &cobra.Command{
	Use:   "list [spreadsheetId]",
	Short: "Lists the sheets in a spreadsheet.",
	Long:  `Lists the sheets in a spreadsheet.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		spreadsheetId := args[0]
		sheets, err := drive.ListSheets(sheetsSvc, spreadsheetId)
		if err != nil {
			return err
		}
		fmt.Println("Sheets:")
		for _, sheet := range sheets {
			fmt.Println(sheet)
		}
		return nil
	},
}

var sheetsGetCmd = &cobra.Command{
	Use:   "get [spreadsheetId]",
	Short: "Gets a sheet as CSV.",
	Long:  `Gets a sheet as CSV.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		spreadsheetId := args[0]
		csv, err := drive.GetSheetAsCSV(sheetsSvc, spreadsheetId, sheetName)
		if err != nil {
			return err
		}

		if sheetsOutputFile != "" {
			err := os.WriteFile(sheetsOutputFile, []byte(csv), 0644)
			if err != nil {
				return fmt.Errorf("failed to write to output file %s: %w", sheetsOutputFile, err)
			}
			fmt.Printf("Successfully saved sheet to %s\n", sheetsOutputFile)
		} else {
			fmt.Println(csv)
		}
		return nil
	},
}

var sheetsGetRangeCmd = &cobra.Command{
	Use:   "get-range [spreadsheetId]",
	Short: "Gets a specific range from a sheet.",
	Long:  `Gets a specific range from a sheet.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		spreadsheetId := args[0]
		values, err := drive.GetSheetRange(sheetsSvc, spreadsheetId, sheetName, sheetRange)
		if err != nil {
			return err
		}
		// TODO: Print the values in a nice format.
		fmt.Println(values)
		return nil
	},
}

var sheetsUpdateRangeCmd = &cobra.Command{
	Use:   "update-range [spreadsheetId] [value]",
	Short: "Updates a specific range in a sheet.",
	Long:  `Updates a specific range in a sheet.`,
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		spreadsheetId := args[0]
		value := args[1]
		values := [][]interface{}{{value}}
		err := drive.UpdateSheetRange(sheetsSvc, spreadsheetId, sheetName, sheetRange, values)
		if err != nil {
			return err
		}
		fmt.Println("Sheet updated successfully.")
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
	sheetsGetCmd.MarkFlagRequired("sheet")
	sheetsGetCmd.Flags().StringVarP(&sheetsOutputFile, "output", "o", "", "Path to save the output file")

	sheetsGetRangeCmd.Flags().StringVar(&sheetName, "sheet", "", "Name of the sheet")
	sheetsGetRangeCmd.MarkFlagRequired("sheet")
	sheetsGetRangeCmd.Flags().StringVar(&sheetRange, "range", "", "The A1 notation of the range to retrieve")
	sheetsGetRangeCmd.MarkFlagRequired("range")

	sheetsUpdateRangeCmd.Flags().StringVar(&sheetName, "sheet", "", "Name of the sheet")
	sheetsUpdateRangeCmd.MarkFlagRequired("sheet")
	sheetsUpdateRangeCmd.Flags().StringVar(&sheetRange, "range", "", "The A1 notation of the range to update")
	sheetsUpdateRangeCmd.MarkFlagRequired("range")
}