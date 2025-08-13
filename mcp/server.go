package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/ghchinoy/drivectl/internal/drive"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/api/docs/v1"
	googledrive "google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

const driveQueryCheatSheet = `
- "mimeType='application/vnd.google-apps.folder'"
- "name contains 'meeting notes'"
- "modifiedTime > '2025-01-01T00:00:00Z'"
- "trashed = false"
`

const a1NotationCheatSheet = `
A1 notation is a way to specify a cell or a range of cells in a spreadsheet. It consists of the column letter(s) followed by the row number.

Examples:
- A1 refers to the cell at the intersection of column A and row 1.
- A1:B2 refers to the range of cells from A1 to B2.
- Sheet1!A1:B2 refers to the range A1:B2 on the sheet named "Sheet1".
`

// getDriveSvc creates a new Google Drive service client.
func getDriveSvc(ctx context.Context) (*googledrive.Service, error) {
	viper.AutomaticEnv()
	secretFile := viper.GetString("secret-file")
	if secretFile == "" {
		return nil, fmt.Errorf("client secret file not set. Please use the --secret-file flag or set the DRIVE_SECRETS environment variable")
	}
	noBrowserAuth := viper.GetBool("no-browser-auth")
	client, err := drive.NewOAuthClient(ctx, secretFile, noBrowserAuth)
	if err != nil {
		return nil, fmt.Errorf("could not create oauth client: %w", err)
	}
	driveSvc, err := googledrive.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("could not create drive service: %w", err)
	}
	return driveSvc, nil
}

// getDocsSvc creates a new Google Docs service client.
func getDocsSvc(ctx context.Context) (*docs.Service, error) {
	viper.AutomaticEnv()
	secretFile := viper.GetString("secret-file")
	if secretFile == "" {
		return nil, fmt.Errorf("client secret file not set. Please use the --secret-file flag or set the DRIVE_SECRETS environment variable")
	}
	noBrowserAuth := viper.GetBool("no-browser-auth")
	client, err := drive.NewOAuthClient(ctx, secretFile, noBrowserAuth)
	if err != nil {
		return nil, fmt.Errorf("could not create oauth client: %w", err)
	}
	docsSvc, err := docs.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("could not create docs service: %w", err)
	}
	return docsSvc, nil
}

// getSheetsSvc creates a new Google Sheets service client.
func getSheetsSvc(ctx context.Context) (*sheets.Service, error) {
	viper.AutomaticEnv()
	secretFile := viper.GetString("secret-file")
	if secretFile == "" {
		return nil, fmt.Errorf("client secret file not set. Please use the --secret-file flag or set the DRIVE_SECRETS environment variable")
	}
	noBrowserAuth := viper.GetBool("no-browser-auth")
	client, err := drive.NewOAuthClient(ctx, secretFile, noBrowserAuth)
	if err != nil {
		return nil, fmt.Errorf("could not create oauth client: %w", err)
	}
	sheetsSvc, err := sheets.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("could not create sheets service: %w", err)
	}
	return sheetsSvc, nil
}

// ListArgs defines the arguments for the list tool.
type ListArgs struct {
	Limit int64  `json:"limit"`
	Query string `json:"query"`
}

// GetArgs defines the arguments for the get tool.
type GetArgs struct {
	FileID string `json:"file-id"`
	Format string `json:"format"`
	TabID  string `json:"tab-id"`
}

// DescribeArgs defines the arguments for the describe tool.
type DescribeArgs struct {
	FileID string `json:"file-id"`
}

// TabsArgs defines the arguments for the tabs tool.
type TabsArgs struct {
	DocumentID string `json:"document-id"`
}

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

// driveQueryCheatSheetHandler is a resource handler that returns a cheat sheet of Drive query examples.
func driveQueryCheatSheetHandler(ctx context.Context, ss *mcp.ServerSession, params *mcp.ReadResourceParams) (*mcp.ReadResourceResult, error) {
	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{
			{
				URI:      params.URI,
				MIMEType: "text/plain",
				Text:     driveQueryCheatSheet,
			},
		},
	}, nil
}

// a1NotationCheatSheetHandler is a resource handler that returns a cheat sheet of A1 notation examples.
func a1NotationCheatSheetHandler(ctx context.Context, ss *mcp.ServerSession, params *mcp.ReadResourceParams) (*mcp.ReadResourceResult, error) {
	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{
			{
				URI:      params.URI,
				MIMEType: "text/plain",
				Text:     a1NotationCheatSheet,
			},
		},
	}, nil
}

// Start starts the MCP server.
func Start(rootCmd *cobra.Command, httpAddr string) error {
	server := mcp.NewServer(&mcp.Implementation{Name: "drivectl"}, nil)

	for _, cmd := range rootCmd.Commands() {
		command := cmd

		switch command.Name() {
		case "list":
			mcp.AddTool(server, &mcp.Tool{
				Name:        command.Name(),
				Description: command.Long,
			}, func(ctx context.Context, ss *mcp.ServerSession, params *mcp.CallToolParamsFor[ListArgs]) (*mcp.CallToolResultFor[any], error) {
				driveSvc, err := getDriveSvc(ctx)
				if err != nil {
					return nil, err
				}

				limit := params.Arguments.Limit
				if limit == 0 {
					limit = 100
				}
				files, err := drive.ListFiles(driveSvc, limit, params.Arguments.Query)
				if err != nil {
					return nil, fmt.Errorf("unable to retrieve files: %w", err)
				}

				var output string
				if len(files) == 0 {
					output = "No files found."
				} else {
					for _, i := range files {
						output += fmt.Sprintf("%s (%s)\n", i.Name, i.Id)
					}
				}

				return &mcp.CallToolResultFor[any]{
					Content: []mcp.Content{
						&mcp.TextContent{Text: output},
					},
				}, nil
			})
		case "get":
			mcp.AddTool(server, &mcp.Tool{
				Name:        command.Name(),
				Description: command.Long,
			}, func(ctx context.Context, ss *mcp.ServerSession, params *mcp.CallToolParamsFor[GetArgs]) (*mcp.CallToolResultFor[any], error) {
				driveSvc, err := getDriveSvc(ctx)
				if err != nil {
					return nil, err
				}
				docsSvc, err := getDocsSvc(ctx)
				if err != nil {
					return nil, err
				}

				content, err := drive.GetFile(driveSvc, docsSvc, params.Arguments.FileID, params.Arguments.Format, params.Arguments.TabID)
				if err != nil {
					return nil, fmt.Errorf("unable to get file: %w", err)
				}

				return &mcp.CallToolResultFor[any]{
					Content: []mcp.Content{
						&mcp.TextContent{Text: string(content)},
					},
				}, nil
			})
		case "describe":
			mcp.AddTool(server, &mcp.Tool{
				Name:        command.Name(),
				Description: command.Long,
			}, func(ctx context.Context, ss *mcp.ServerSession, params *mcp.CallToolParamsFor[DescribeArgs]) (*mcp.CallToolResultFor[any], error) {
				if params.Arguments.FileID == "" {
					return nil, fmt.Errorf("file-id is a required argument")
				}

				driveSvc, err := getDriveSvc(ctx)
				if err != nil {
					return nil, err
				}

				file, err := drive.DescribeFile(driveSvc, params.Arguments.FileID)
				if err != nil {
					return nil, fmt.Errorf("unable to describe file: %w", err)
				}

				jsonFile, err := json.MarshalIndent(file, "", "  ")
				if err != nil {
					return nil, fmt.Errorf("unable to marshal file to json: %w", err)
				}

				return &mcp.CallToolResultFor[any]{
					Content: []mcp.Content{
						&mcp.TextContent{Text: string(jsonFile)},
					},
				}, nil
			})
		case "docs":
			for _, subCmd := range command.Commands() {
				subCommand := subCmd
				switch subCommand.Name() {
				case "tabs":
					mcp.AddTool(server, &mcp.Tool{
						Name:        "docs.tabs",
						Description: subCommand.Long,
					}, func(ctx context.Context, ss *mcp.ServerSession, params *mcp.CallToolParamsFor[TabsArgs]) (*mcp.CallToolResultFor[any], error) {
						if params.Arguments.DocumentID == "" {
							return nil, fmt.Errorf("document-id is a required argument")
						}

						docsSvc, err := getDocsSvc(ctx)
						if err != nil {
							return nil, err
						}

						tabs, err := drive.GetTabs(docsSvc, params.Arguments.DocumentID)
						if err != nil {
							return nil, fmt.Errorf("unable to get tabs: %w", err)
						}

						var tabStrings []string
						var traverse func(tabs []*drive.TabInfo, level int)
						traverse = func(tabs []*drive.TabInfo, level int) {
							for _, tab := range tabs {
								tabStrings = append(tabStrings, fmt.Sprintf("%s%s (%s)", strings.Repeat("\t", level), tab.Title, tab.TabID))
								if len(tab.Children) > 0 {
									traverse(tab.Children, level+1)
								}
							}
						}
						traverse(tabs, 0)

						return &mcp.CallToolResultFor[any]{
							Content: []mcp.Content{
								&mcp.TextContent{Text: strings.Join(tabStrings, "\n")},
							},
						}, nil
					})
				}
			}
		case "sheets":
			for _, subCmd := range command.Commands() {
				subCommand := subCmd
				switch subCommand.Name() {
				case "list":
					mcp.AddTool(server, &mcp.Tool{
						Name:        "sheets.list",
						Description: subCommand.Long,
					}, func(ctx context.Context, ss *mcp.ServerSession, params *mcp.CallToolParamsFor[ListSheetsArgs]) (*mcp.CallToolResultFor[any], error) {
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
					})
				case "get":
					mcp.AddTool(server, &mcp.Tool{
						Name:        "sheets.get",
						Description: subCommand.Long,
					}, func(ctx context.Context, ss *mcp.ServerSession, params *mcp.CallToolParamsFor[GetSheetArgs]) (*mcp.CallToolResultFor[any], error) {
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
					})
				case "get-range":
					mcp.AddTool(server, &mcp.Tool{
						Name:        "sheets.get-range",
						Description: subCommand.Long,
					}, func(ctx context.Context, ss *mcp.ServerSession, params *mcp.CallToolParamsFor[GetSheetRangeArgs]) (*mcp.CallToolResultFor[any], error) {
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
					})
				case "update-range":
					mcp.AddTool(server, &mcp.Tool{
						Name:        "sheets.update-range",
						Description: subCommand.Long,
					}, updateRangeHandler)
				}
			}
		}
	}

	server.AddResource(&mcp.Resource{
		Name:        "drive-query-cheat-sheet",
		Description: "A cheat sheet of example Google Drive query examples.",
		MIMEType:    "text/plain",
		URI:         "embedded:drive-query-cheat-sheet",
	}, driveQueryCheatSheetHandler)

	server.AddResource(&mcp.Resource{
		Name:        "a1-notation-cheat-sheet",
		Description: "A cheat sheet of example A1 notation for Google Sheets.",
		MIMEType:    "text/plain",
		URI:         "embedded:a1-notation-cheat-sheet",
	}, a1NotationCheatSheetHandler)

	if httpAddr != "" {
		handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
			return server
		}, nil)
		log.Printf("MCP handler listening at %s", httpAddr)
		return http.ListenAndServe(httpAddr, handler)
	}

	logFile, err := os.OpenFile("drivectl-mcp.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer logFile.Close()

	log.SetOutput(logFile)

	t := mcp.NewLoggingTransport(mcp.NewStdioTransport(), logFile)
	if err := server.Run(context.Background(), t); err != nil {
		log.Printf("Server failed: %v", err)
		return err
	}

	return nil
}

func updateRangeHandler(ctx context.Context, ss *mcp.ServerSession, params *mcp.CallToolParamsFor[UpdateSheetRangeArgs]) (*mcp.CallToolResultFor[any], error) {
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
