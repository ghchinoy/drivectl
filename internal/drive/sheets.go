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

package drive

import (
	"errors"
	"fmt"
	"strings"

	"google.golang.org/api/sheets/v4"
)

// ListSheets lists the sheets in a spreadsheet.
func ListSheets(sheetsSvc *sheets.Service, spreadsheetID string) ([]string, error) {
	spreadsheet, err := sheetsSvc.Spreadsheets.Get(spreadsheetID).Fields("sheets(properties(title))").Do()
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve spreadsheet: %w", err)
	}

	var sheetNames []string
	for _, sheet := range spreadsheet.Sheets {
		if sheet != nil && sheet.Properties != nil {
			sheetNames = append(sheetNames, sheet.Properties.Title)
		}
	}

	return sheetNames, nil
}

// FetchSheetAsCSV retrieves a sheet and formats its cells as CSV.
func FetchSheetAsCSV(sheetsSvc *sheets.Service, spreadsheetID, sheetName string) (string, error) {
	resp, err := sheetsSvc.Spreadsheets.Values.Get(spreadsheetID, sheetName).Do()
	if err != nil {
		return "", fmt.Errorf("unable to retrieve data from sheet: %w", err)
	}

	if len(resp.Values) == 0 {
		return "", errors.New("no data found")
	}

	var csvBuilder strings.Builder
	for _, row := range resp.Values {
		var csvRow []string
		for _, cell := range row {
			csvRow = append(csvRow, fmt.Sprint(cell))
		}
		csvBuilder.WriteString(strings.Join(csvRow, ","))
		csvBuilder.WriteString("\n")
	}

	return csvBuilder.String(), nil
}

// FetchSheetRange retrieves a specific A1 notation range from a sheet.
func FetchSheetRange(sheetsSvc *sheets.Service, spreadsheetID, sheetName, sheetRange string) ([][]any, error) {
	readRange := sheetName + "!" + sheetRange
	resp, err := sheetsSvc.Spreadsheets.Values.Get(spreadsheetID, readRange).Do()
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve data from sheet: %w", err)
	}

	return resp.Values, nil
}

// UpdateSheetRange updates a specific A1 notation range in a sheet.
func UpdateSheetRange(sheetsSvc *sheets.Service, spreadsheetID, sheetName, sheetRange string, values [][]any) error {
	writeRange := sheetName + "!" + sheetRange
	valueRange := &sheets.ValueRange{
		Values: values,
	}
	_, err := sheetsSvc.Spreadsheets.Values.Update(spreadsheetID, writeRange, valueRange).ValueInputOption("USER_ENTERED").Do()
	if err != nil {
		return fmt.Errorf("unable to update sheet: %w", err)
	}
	return nil
}
