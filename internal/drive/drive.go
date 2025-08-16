package drive

import (
	"fmt"
	"io"
	"strings"

	"google.golang.org/api/docs/v1"
	googledrive "google.golang.org/api/drive/v3"
)

// ListFiles lists the files and folders in Google Drive.
func ListFiles(srv *googledrive.Service, limit int64, query string) ([]*googledrive.File, error) {
	r, err := srv.Files.List().PageSize(limit).Q(query).
		Fields("nextPageToken, files(id, name)").Do()
	if err != nil {
		return nil, err
	}
	return r.Files, nil
}

var formatMap = map[string]string{
	"pdf":      "application/pdf",
	"docx":     "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	"html":     "text/html",
	"zip":      "application/zip",
	"epub":     "application/epub+zip",
	"txt":      "text/plain",
	"md":       "text/markdown",
	"markdown": "text/markdown",
	"csv":      "text/csv",
	"tsv":      "text/tab-separated-values",
	"xlsx":     "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	"ods":      "application/vnd.oasis.opendocument.spreadsheet",
	"pptx":     "application/vnd.openxmlformats-officedocument.presentationml.presentation",
	"odp":      "application/vnd.oasis.opendocument.presentation",
	"png":      "image/png",
	"jpg":      "image/jpeg",
}

// GetFile downloads a file or exports a Google Doc.
func GetFile(driveSvc *googledrive.Service, docsSvc *docs.Service, fileId string, format string, tabId string) ([]byte, error) {
	if tabId != "" {
		doc, err := docsSvc.Documents.Get(fileId).IncludeTabsContent(true).Do()
		if err != nil {
			return nil, fmt.Errorf("unable to retrieve document with tabs: %w", err)
		}

		var findTab func(tabs []*docs.Tab) *docs.Tab
		findTab = func(tabs []*docs.Tab) *docs.Tab {
			for _, t := range tabs {
				if t.TabProperties != nil && t.TabProperties.TabId == tabId {
					return t
				}
				if len(t.ChildTabs) > 0 {
					if found := findTab(t.ChildTabs); found != nil {
						return found
					}
				}
			}
			return nil
		}

		if tab := findTab(doc.Tabs); tab != nil {
			textContent := renderBodyAsText(tab.DocumentTab.Body)
			return []byte(textContent), nil
		}

		return nil, fmt.Errorf("tab with id %s not found", tabId)
	}

	file, err := driveSvc.Files.Get(fileId).Fields("mimeType", "name").Do()
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve file metadata: %w", err)
	}

	var content []byte

	if strings.HasPrefix(file.MimeType, "application/vnd.google-apps") {
		exportMimeType, ok := formatMap[strings.ToLower(format)]
		if !ok && format != "" {
			return nil, fmt.Errorf("invalid format: %s. Valid formats are: pdf, docx, html, zip, epub, txt, md, csv, tsv, xlsx, ods, pptx, odp", format)
		}

		if file.MimeType == "application/vnd.google-apps.spreadsheet" {
			if format == "" || format == "txt" {
				exportMimeType = "text/csv"
			}
		} else if file.MimeType == "application/vnd.google-apps.presentation" {
			if format == "" || format == "txt" {
				exportMimeType = "text/plain"
			}
		} else if format == "" {
			exportMimeType = "text/plain"
		}

		resp, err := driveSvc.Files.Export(fileId, exportMimeType).Download()
		if err != nil {
			return nil, fmt.Errorf("unable to export Google Doc: %w", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("unable to read exported content: %w", err)
		}
		content = body
	} else {
		resp, err := driveSvc.Files.Get(fileId).Download()
		if err != nil {
			return nil, fmt.Errorf("unable to download file: %w", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("unable to read file content: %w", err)
		}
		content = body
	}
	return content, nil
}

// DescribeFile shows detailed metadata for a specific file.
func DescribeFile(driveSvc *googledrive.Service, fileId string) (*googledrive.File, error) {
	file, err := driveSvc.Files.Get(fileId).Fields("*").Do()
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve file: %w", err)
	}
	return file, nil
}

// UploadFile uploads a file to Google Drive.
func UploadFile(driveSvc *googledrive.Service, name string, mimeType string, content io.Reader) (*googledrive.File, error) {
	file := &googledrive.File{
		Name:     name,
		MimeType: mimeType,
	}
	file, err := driveSvc.Files.Create(file).Media(content).Do()
	if err != nil {
		return nil, fmt.Errorf("could not create file: %w", err)
	}
	return file, nil
}
