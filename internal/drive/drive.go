package drive

import (
	"fmt"
	"io"
	"strings"

	"google.golang.org/api/docs/v1"
	"google.golang.org/api/drive/v3"
)

// ListFiles lists the files and folders in Google Drive.
func ListFiles(srv *drive.Service, limit int64, query string) ([]*drive.File, error) {
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
}

// renderBodyAsText converts a Google Docs Body object to a plain text string.
func renderBodyAsText(body *docs.Body) string {
	var text strings.Builder
	if body == nil || body.Content == nil {
		return ""
	}
	for _, element := range body.Content {
		if element.Paragraph != nil {
			for _, pElem := range element.Paragraph.Elements {
				if pElem.TextRun != nil {
					text.WriteString(pElem.TextRun.Content)
				}
			}
		}
	}
	return text.String()
}

// GetFile downloads a file or exports a Google Doc.
func GetFile(driveSvc *drive.Service, docsSvc *docs.Service, fileId string, format string, tabId string) ([]byte, error) {
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
func DescribeFile(driveSvc *drive.Service, fileId string) (*drive.File, error) {
	file, err := driveSvc.Files.Get(fileId).Fields("*").Do()
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve file: %w", err)
	}
	return file, nil
}

// TabInfo contains information about a tab in a Google Doc.
type TabInfo struct {
	Title    string
	TabID    string
	Level    int
	Children []*TabInfo
}

// GetTabs lists the tabs within a Google Doc.
func GetTabs(docsSvc *docs.Service, documentId string) ([]*TabInfo, error) {
	doc, err := docsSvc.Documents.Get(documentId).IncludeTabsContent(true).Do()
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve document with tabs: %w", err)
	}

	var buildTabs func(tabs []*docs.Tab, level int) []*TabInfo
	buildTabs = func(tabs []*docs.Tab, level int) []*TabInfo {
		var result []*TabInfo
		for _, t := range tabs {
			if t.TabProperties != nil {
				tabInfo := &TabInfo{
					Title: t.TabProperties.Title,
					TabID: t.TabProperties.TabId,
					Level: level,
				}
				if len(t.ChildTabs) > 0 {
					tabInfo.Children = buildTabs(t.ChildTabs, level+1)
				}
				result = append(result, tabInfo)
			}
		}
		return result
	}

	return buildTabs(doc.Tabs, 0), nil
}

// CreateDocFromMarkdown creates a new Google Doc from a Markdown string.
func CreateDocFromMarkdown(docsSvc *docs.Service, title string, markdownContent string) (*docs.Document, error) {
	doc := &docs.Document{
		Title: title,
	}
	createdDoc, err := docsSvc.Documents.Create(doc).Do()
	if err != nil {
		return nil, fmt.Errorf("could not create file: %w", err)
	}

	requests, err := MarkdownToDocsRequests(markdownContent)
	if err != nil {
		return nil, fmt.Errorf("unable to convert markdown to requests: %w", err)
	}

	if len(requests) > 0 {
		_, err = docsSvc.Documents.BatchUpdate(createdDoc.DocumentId, &docs.BatchUpdateDocumentRequest{
			Requests: requests,
		}).Do()
		if err != nil {
			return nil, fmt.Errorf("could not update document: %w", err)
		}
	}

	return createdDoc, nil
}
