package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/ghchinoy/drivectl/internal/drive"
	"github.com/ghchinoy/drivectl/internal/ui"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var authCmd = &cobra.Command{
	Use:     "auth",
	GroupID: GroupAuth,
	Short:   "Authentication commands",
	Long:    `Manage authentication state for drivectl.`,
}

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login to Google Drive",
	Long: `Login to Google Drive using OAuth2.
This command will open your browser to authenticate with Google.
It will save the credentials to a local configuration directory for future use.`,
	Example: `  drivectl auth login --secret-file ./client_secret.json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		secretFile := viper.GetString("secret-file")
		
		configDir, err := drive.ConfigDir()
		if err != nil {
			return err
		}
		targetSecretFile := filepath.Join(configDir, "client_secret.json")

		// If the user provided a secret file explicitly, copy it to the cache directory
		if secretFile != "" && secretFile != targetSecretFile {
			fmt.Printf("Copying client secrets from %s to %s...\n", secretFile, targetSecretFile)
			source, err := os.Open(secretFile)
			if err != nil {
				return fmt.Errorf("failed to open source secret file: %w", err)
			}
			defer source.Close()

			destination, err := os.Create(targetSecretFile)
			if err != nil {
				return fmt.Errorf("failed to create target secret file: %w", err)
			}
			defer destination.Close()

			if _, err := io.Copy(destination, source); err != nil {
				return fmt.Errorf("failed to copy secret file: %w", err)
			}
		}

		// Ensure the secret file exists in the target location
		if _, err := os.Stat(targetSecretFile); os.IsNotExist(err) {
			return fmt.Errorf("client secret file not found. Please provide one via --secret-file on your first login")
		}

		// Clear existing token to force a re-login
		tokenFile, err := drive.TokenCacheFile()
		if err == nil {
			os.Remove(tokenFile)
		}

		fmt.Println("Starting OAuth login flow...")
		ctx := context.Background()
		// We pass targetSecretFile to explicitly use the cached one
		_, err = drive.NewOAuthClient(ctx, targetSecretFile, noBrowserAuth)
		if err != nil {
			return ui.ErrorWithHint(fmt.Errorf("login failed: %w", err), "Ensure your client_secret.json is valid and the browser flow completed.")
		}

		ui.PrintSuccess("Login successful! You can now run drivectl commands.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(authCmd)
	authCmd.AddCommand(authLoginCmd)
}
