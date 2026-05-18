package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/ghchinoy/drivectl/internal/drive"
	"github.com/ghchinoy/drivectl/internal/ui"
	"github.com/spf13/cobra"
)

var parentID string

var uploadCmd = &cobra.Command{
	Use:     "upload [file]",
	GroupID: GroupCore,
	Short:   "Uploads a local file to Google Drive.",
	Long: `Uploads a file from your local filesystem to Google Drive.
Optionally, specify a parent folder ID to upload the file to a specific folder.`,
	Example: `  drivectl upload path/to/my/file.txt
  drivectl upload path/to/my/file.txt --parent <folder-id>`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath := args[0]

		res, err := drive.UploadFile(driveSvc, filePath, parentID)
		if err != nil {
			return ui.ErrorWithHint(err, "Ensure the file path is correct and you have permission to upload.")
		}

		if OutputFormat == "json" {
			b, _ := json.MarshalIndent(res, "", "  ")
			fmt.Println(string(b))
		} else {
			ui.PrintSuccess("Successfully uploaded file: %s (ID: %s)", res.Name, res.Id)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(uploadCmd)
	uploadCmd.Flags().StringVarP(&parentID, "parent", "p", "", "Parent folder ID to upload to")
}