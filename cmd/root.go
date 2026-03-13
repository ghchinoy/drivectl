package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/ghchinoy/drivectl/internal/drive"
	"github.com/ghchinoy/drivectl/internal/ui"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/api/docs/v1"
	googledrive "google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

var (
	// Version is the current version of the application, injected at build time.
	Version = "dev"
	// noBrowserAuth is a flag to disable opening the browser for authentication.
	noBrowserAuth bool
	// client is the HTTP client used for all API calls.
	client *http.Client
	// driveSvc is the Google Drive service client.
	driveSvc *googledrive.Service
	// docsSvc is the Google Docs service client.
	docsSvc *docs.Service
	// sheetsSvc is the Google Sheets service client.
	sheetsSvc *sheets.Service
)

// Command Groups
const (
	GroupAuth        = "auth"
	GroupCore        = "core"
	GroupIntegration = "integration"
	GroupAdvanced    = "advanced"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:     "drivectl",
	Version: Version,
	Short:   "A CLI for Google Drive and Docs.",
	Long: `drivectl is a powerful command-line tool for interacting with your Google Drive files.
It allows you to list, describe, and download files, with advanced support for
Google Docs, including exporting to multiple formats and accessing individual tabs.`, 
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Name() == "login" || cmd.Name() == "auth" || cmd.Name() == "help" {
			return nil
		}
		secretFile := viper.GetString("secret-file")

		ctx := context.Background()
		var err error
		client, err = drive.NewOAuthClient(ctx, secretFile, noBrowserAuth)
		if err != nil {
			return ui.ErrorWithHint(fmt.Errorf("could not create oauth client: %w", err), "run 'drivectl auth login' to authenticate")
		}

		driveSvc, err = googledrive.NewService(ctx, option.WithHTTPClient(client))
		if err != nil {
			return fmt.Errorf("could not create drive service: %w", err)
		}

		docsSvc, err = docs.NewService(ctx, option.WithHTTPClient(client))
		if err != nil {
			return fmt.Errorf("could not create docs service: %w", err)
		}

		sheetsSvc, err = sheets.NewService(ctx, option.WithHTTPClient(client))
		if err != nil {
			return fmt.Errorf("could not create sheets service: %w", err)
		}
		return nil
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// ExecuteError executes the root command and returns any errors to the caller.
func ExecuteError() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddGroup(&cobra.Group{ID: GroupAuth, Title: "Authentication Commands:"})
	rootCmd.AddGroup(&cobra.Group{ID: GroupCore, Title: "Core Commands:"})
	rootCmd.AddGroup(&cobra.Group{ID: GroupIntegration, Title: "Integration Commands:"})
	rootCmd.AddGroup(&cobra.Group{ID: GroupAdvanced, Title: "Advanced Commands:"})

	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().String("secret-file", "", "path to your client secrets file")
	rootCmd.PersistentFlags().BoolVar(&noBrowserAuth, "no-browser-auth", false, "do not open a browser for authentication")
	rootCmd.PersistentFlags().Bool("mcp", false, "enable MCP server mode over stdio")
	rootCmd.PersistentFlags().String("mcp-http", "", "enable MCP server mode over HTTP at the given address")
	_ = viper.BindPFlag("secret-file", rootCmd.PersistentFlags().Lookup("secret-file"))
	_ = viper.BindEnv("secret-file", "DRIVE_SECRETS")
	_ = viper.BindPFlag("mcp", rootCmd.PersistentFlags().Lookup("mcp"))
	_ = viper.BindPFlag("mcp-http", rootCmd.PersistentFlags().Lookup("mcp-http"))
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	viper.AutomaticEnv()
}