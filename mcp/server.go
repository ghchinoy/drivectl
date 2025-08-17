package mcp

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/ghchinoy/drivectl/internal/drive"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/api/docs/v1"
	googledrive "google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
	"google.golang.org/api/slides/v1"
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

// getSlidesSvc creates a new Google Slides service client.
func getSlidesSvc(ctx context.Context) (*slides.Service, error) {
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
	slidesSvc, err := slides.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("could not create slides service: %w", err)
	}
	return slidesSvc, nil
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

	RegisterDriveTools(server, rootCmd)
	RegisterDocsTools(server, rootCmd)
	RegisterSheetsTools(server, rootCmd)
	RegisterSlidesTools(server, rootCmd)

	// drive-query-cheat-sheet resource
	server.AddResource(&mcp.Resource{
		Name:        "drive-query-cheat-sheet",
		Description: "A cheat sheet of example Google Drive query examples.",
		MIMEType:    "text/plain",
		URI:         "embedded:drive-query-cheat-sheet",
	}, driveQueryCheatSheetHandler)

	// a1-notation-cheat-sheet
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

	logFile := viper.GetString("log-file")
	if logFile != "" {
		f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return fmt.Errorf("failed to open log file: %w", err)
		}
		defer f.Close()
		log.SetOutput(f)
		fmt.Printf("MCP server logging to %s\n", logFile)
		t := mcp.NewLoggingTransport(mcp.NewStdioTransport(), f)
		if err := server.Run(context.Background(), t); err != nil {
			log.Printf("Server failed: %v", err)
			return err
		}
	} else {
		t := mcp.NewStdioTransport()
		if err := server.Run(context.Background(), t); err != nil {
			log.Printf("Server failed: %v", err)
			return err
		}
	}

	return nil
}

