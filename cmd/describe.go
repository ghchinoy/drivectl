
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

	"github.com/spf13/cobra"
)

var describeCmd = &cobra.Command{
	Use:   "describe [fileId]",
	Short: "Describes a file in Google Drive",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fileId := args[0]

		file, err := driveSvc.Files.Get(fileId).Fields("*").Do()
		if err != nil {
			return fmt.Errorf("unable to retrieve file: %w", err)
		}

		jsonFile, err := json.MarshalIndent(file, "", "  ")
		if err != nil {
			return fmt.Errorf("unable to marshal file to json: %w", err)
		}

		fmt.Println(string(jsonFile))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(describeCmd)
}
