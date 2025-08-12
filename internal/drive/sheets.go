package drive

import (
	"fmt"
	"strings"

	"google.golang.org/api/sheets/v4"
)

// ListSheets lists the sheets in a spreadsheet.
func ListSheets(sheetsSvc *sheets.Service, spreadsheetId string) ([]string, error) {
	spreadsheet, err := sheetsSvc.Spreadsheets.Get(spreadsheetId).Fields("sheets(properties(title))").Do()
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve spreadsheet: %w", err)
	}

	var sheetNames []string
	for _, sheet := range spreadsheet.Sheets {
		sheetNames = append(sheetNames, sheet.Properties.Title)
	}

	return sheetNames, nil
}

// GetSheetAsCSV gets a sheet as CSV.
func GetSheetAsCSV(sheetsSvc *sheets.Service, spreadsheetId string, sheetName string) (string, error) {
	readRange := sheetName
	resp, err := sheetsSvc.Spreadsheets.Values.Get(spreadsheetId, readRange).Do()
	if err != nil {
		return "", fmt.Errorf("unable to retrieve data from sheet: %w", err)
	}

	var csv string
	if len(resp.Values) == 0 {
		return "", fmt.Errorf("no data found")
	}

	for _, row := range resp.Values {
		var csvRow []string
		for _, cell := range row {
			csvRow = append(csvRow, fmt.Sprintf("%v", cell))
		}
		csv += strings.Join(csvRow, ",") + "\n"
	}

	return csv, nil
}

// GetSheetRange gets a specific range from a sheet.
func GetSheetRange(sheetsSvc *sheets.Service, spreadsheetId string, sheetName string, sheetRange string) ([][]interface{}, error) {
	readRange := sheetName + "!" + sheetRange
	resp, err := sheetsSvc.Spreadsheets.Values.Get(spreadsheetId, readRange).Do()
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve data from sheet: %w", err)
	}

	return resp.Values, nil
}

// UpdateSheetRange updates a specific range in a sheet.
func UpdateSheetRange(sheetsSvc *sheets.Service, spreadsheetId string, sheetName string, sheetRange string, values [][]interface{}) error {
	writeRange := sheetName + "!" + sheetRange
	valueRange := &sheets.ValueRange{
		Values: values,
	}
	_, err := sheetsSvc.Spreadsheets.Values.Update(spreadsheetId, writeRange, valueRange).ValueInputOption("USER_ENTERED").Do()
	if err != nil {
		return fmt.Errorf("unable to update sheet: %w", err)
	}
	return nil
}
