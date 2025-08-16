package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ghchinoy/drivectl/internal/drive"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
)

// ListSheetsArgs defines the arguments for the sheets list tool.
type ListSheetsArgs struct {
	SpreadsheetID string `json:"spreadsheet-id"`
}

// GetSheetArgs defines the arguments for the sheets get tool.
type GetSheetArgs struct {
	SpreadsheetID string `json:"spreadsheet-id"`
	SheetName     string `json:"sheet-name"`
}

// GetSheetRangeArgs defines the arguments for the sheets get-range tool.
type GetSheetRangeArgs struct {
	SpreadsheetID string `json:"spreadsheet-id"`
	SheetName     string `json:"sheet-name"`
	Range         string `json:"range"`
}

// UpdateSheetRangeArgs defines the arguments for the sheets update-range tool.
type UpdateSheetRangeArgs struct {
	SpreadsheetID string `json:"spreadsheet-id"`
	SheetName     string `json:"sheet-name"`
	Range         string `json:"range"`
	Value         string `json:"value"`
}

// SheetsListHandler is the handler for the sheets list tool.
func SheetsListHandler(ctx context.Context, ss *mcp.ServerSession, params *mcp.CallToolParamsFor[ListSheetsArgs]) (*mcp.CallToolResultFor[any], error) {
	if params.Arguments.SpreadsheetID == "" {
		return nil, fmt.Errorf("spreadsheet-id is a required argument")
	}
	sheetsSvc, err := getSheetsSvc(ctx)
	if err != nil {
		return nil, err
	}

	sheets, err := drive.ListSheets(sheetsSvc, params.Arguments.SpreadsheetID)
	if err != nil {
		return nil, fmt.Errorf("unable to list sheets: %w", err)
	}

	return &mcp.CallToolResultFor[any]{
		Content: []mcp.Content{
			&mcp.TextContent{Text: strings.Join(sheets, "\n")},
		},
	}, nil
}

// SheetsGetHandler is the handler for the sheets get tool.
func SheetsGetHandler(ctx context.Context, ss *mcp.ServerSession, params *mcp.CallToolParamsFor[GetSheetArgs]) (*mcp.CallToolResultFor[any], error) {
	if params.Arguments.SpreadsheetID == "" {
		return nil, fmt.Errorf("spreadsheet-id is a required argument")
	}
	if params.Arguments.SheetName == "" {
		return nil, fmt.Errorf("sheet-name is a required argument")
	}
	sheetsSvc, err := getSheetsSvc(ctx)
	if err != nil {
		return nil, err
	}

	csv, err := drive.GetSheetAsCSV(sheetsSvc, params.Arguments.SpreadsheetID, params.Arguments.SheetName)
	if err != nil {
		return nil, fmt.Errorf("unable to get sheet as csv: %w", err)
	}

	return &mcp.CallToolResultFor[any]{
		Content: []mcp.Content{
			&mcp.TextContent{Text: csv},
		},
	}, nil
}

// SheetsGetRangeHandler is the handler for the sheets get-range tool.
func SheetsGetRangeHandler(ctx context.Context, ss *mcp.ServerSession, params *mcp.CallToolParamsFor[GetSheetRangeArgs]) (*mcp.CallToolResultFor[any], error) {
	if params.Arguments.SpreadsheetID == "" {
		return nil, fmt.Errorf("spreadsheet-id is a required argument")
	}
	if params.Arguments.SheetName == "" {
		return nil, fmt.Errorf("sheet-name is a required argument")
	}
	if params.Arguments.Range == "" {
		return nil, fmt.Errorf("range is a required argument")
	}
	sheetsSvc, err := getSheetsSvc(ctx)
	if err != nil {
		return nil, err
	}

	values, err := drive.GetSheetRange(sheetsSvc, params.Arguments.SpreadsheetID, params.Arguments.SheetName, params.Arguments.Range)
	if err != nil {
		return nil, fmt.Errorf("unable to get sheet range: %w", err)
	}

	jsonValues, err := json.Marshal(values)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal values to json: %w", err)
	}

	return &mcp.CallToolResultFor[any]{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(jsonValues)},
		},
	}, nil
}

// SheetsUpdateRangeHandler is the handler for the sheets update-range tool.
func SheetsUpdateRangeHandler(ctx context.Context, ss *mcp.ServerSession, params *mcp.CallToolParamsFor[UpdateSheetRangeArgs]) (*mcp.CallToolResultFor[any], error) {
	if params.Arguments.SpreadsheetID == "" {
		return nil, fmt.Errorf("spreadsheet-id is a required argument")
	}
	if params.Arguments.SheetName == "" {
		return nil, fmt.Errorf("sheet-name is a required argument")
	}
	if params.Arguments.Range == "" {
		return nil, fmt.Errorf("range is a required argument")
	}
	if params.Arguments.Value == "" {
		return nil, fmt.Errorf("value is a required argument")
	}
	sheetsSvc, err := getSheetsSvc(ctx)
	if err != nil {
		return nil, err
	}

	values := [][]interface{}{{params.Arguments.Value}}
	err = drive.UpdateSheetRange(sheetsSvc, params.Arguments.SpreadsheetID, params.Arguments.SheetName, params.Arguments.Range, values)
	if err != nil {
		return nil, fmt.Errorf("unable to update sheet range: %w", err)
	}

	return &mcp.CallToolResultFor[any]{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "Sheet updated successfully."},
		},
	}, nil
}

func RegisterSheetsTools(server *mcp.Server, rootCmd *cobra.Command) {
	for _, cmd := range rootCmd.Commands() {
		command := cmd

		if command.Name() == "sheets" {
			for _, subCmd := range command.Commands() {
				subCommand := subCmd
				switch subCommand.Name() {
				case "list":
					mcp.AddTool(server, &mcp.Tool{
						Name:        "sheets.list",
						Description: subCommand.Long,
					}, SheetsListHandler)
				case "get":
					mcp.AddTool(server, &mcp.Tool{
						Name:        "sheets.get",
						Description: subCommand.Long,
					}, SheetsGetHandler)
				case "get-range":
					mcp.AddTool(server, &mcp.Tool{
						Name:        "sheets.get-range",
						Description: subCommand.Long,
					}, SheetsGetRangeHandler)
				case "update-range":
					mcp.AddTool(server, &mcp.Tool{
						Name:        "sheets.update-range",
						Description: subCommand.Long,
					}, SheetsUpdateRangeHandler)
				}
			}
		}
	}
}