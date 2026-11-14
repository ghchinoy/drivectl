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
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"google.golang.org/api/docs/v1"
	"google.golang.org/api/drive/v3"
)

// TabInfo contains information about a tab in a Google Doc.
type TabInfo struct {
	Title    string
	TabID    string
	Level    int
	Children []*TabInfo
	Markdown string
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

// ListFiles lists the files and folders in Google Drive.
func ListFiles(driveSvc *drive.Service, limit int64, query string) ([]*drive.File, error) {
	fileList, err := driveSvc.Files.List().PageSize(limit).Q(query).
		Fields("nextPageToken, files(id, name)").Do()
	if err != nil {
		return nil, err
	}
	return fileList.Files, nil
}

// FetchFile downloads a file or exports a Google Doc.
func FetchFile(driveSvc *drive.Service, docsSvc *docs.Service, fileID, format, tabID string) ([]byte, error) {
	if tabID != "" {
		return fetchDocumentTabContent(docsSvc, fileID, tabID)
	}

	file, err := driveSvc.Files.Get(fileID).Fields("mimeType", "name").Do()
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve file metadata: %w", err)
	}

	if strings.HasPrefix(file.MimeType, "application/vnd.google-apps") {
		return exportGoogleAppsFile(driveSvc, fileID, file.MimeType, format)
	}

	return downloadStandardFile(driveSvc, fileID)
}

// DescribeFile retrieves detailed metadata for a specific file.
func DescribeFile(driveSvc *drive.Service, fileID string) (*drive.File, error) {
	file, err := driveSvc.Files.Get(fileID).Fields("*").Do()
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve file: %w", err)
	}
	return file, nil
}

// FetchTabs lists the tabs within a Google Doc.
func FetchTabs(docsSvc *docs.Service, documentID string) ([]*TabInfo, error) {
	doc, err := docsSvc.Documents.Get(documentID).IncludeTabsContent(true).Do()
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
func CreateDocFromMarkdown(docsSvc *docs.Service, title, markdownContent string) (*docs.Document, error) {
	requests, err := MarkdownToDocsRequests(markdownContent)
	if err != nil {
		return nil, fmt.Errorf("unable to convert markdown to requests: %w", err)
	}

	doc := &docs.Document{
		Title: title,
	}
	createdDoc, err := docsSvc.Documents.Create(doc).Do()
	if err != nil {
		return nil, fmt.Errorf("could not create file: %w", err)
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

// UploadFile uploads a local file to Google Drive.
func UploadFile(driveSvc *drive.Service, filePath, parentID string) (*drive.File, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("unable to open file %q: %w", filePath, err)
	}
	defer file.Close()

	fileName := filepath.Base(filePath)
	meta := &drive.File{
		Name: fileName,
	}
	if parentID != "" {
		meta.Parents = []string{parentID}
	}

	uploadedFile, err := driveSvc.Files.Create(meta).Media(file).Do()
	if err != nil {
		return nil, fmt.Errorf("unable to create file %q: %w", fileName, err)
	}
	return uploadedFile, nil
}

// renderBodyAsText converts a Google Docs Body object to a Markdown string.
func renderBodyAsText(body *docs.Body) string {
	if body == nil || len(body.Content) == 0 {
		return ""
	}
	var text strings.Builder
	for _, element := range body.Content {
		if element != nil && element.Paragraph != nil {
			renderParagraphAsMarkdown(&text, element.Paragraph)
		}
	}
	return text.String()
}

func renderParagraphAsMarkdown(text *strings.Builder, p *docs.Paragraph) {
	text.WriteString(headingPrefix(p.ParagraphStyle))
	text.WriteString(bulletPrefix(p.Bullet))
	for _, elem := range p.Elements {
		if elem != nil && elem.TextRun != nil {
			text.WriteString(renderTextRun(elem.TextRun))
		}
	}
}

func headingPrefix(style *docs.ParagraphStyle) string {
	if style == nil {
		return ""
	}
	switch style.NamedStyleType {
	case "HEADING_1":
		return "# "
	case "HEADING_2":
		return "## "
	case "HEADING_3":
		return "### "
	case "HEADING_4":
		return "#### "
	case "HEADING_5":
		return "##### "
	case "HEADING_6":
		return "###### "
	default:
		return ""
	}
}

func bulletPrefix(bullet *docs.Bullet) string {
	if bullet == nil {
		return ""
	}
	return strings.Repeat("  ", int(bullet.NestingLevel)) + "* "
}

func renderTextRun(tr *docs.TextRun) string {
	content := tr.Content
	style := tr.TextStyle
	if style == nil {
		return content
	}
	switch {
	case style.Bold:
		content = "**" + strings.TrimSpace(content) + "** "
	case style.Italic:
		content = "*" + strings.TrimSpace(content) + "* "
	}
	if style.Link != nil && style.Link.Url != "" {
		content = fmt.Sprintf("[%s](%s) ", strings.TrimSpace(content), style.Link.Url)
	}
	return content
}

func findTab(tabs []*docs.Tab, tabID string) *docs.Tab {
	for _, t := range tabs {
		if t.TabProperties != nil && t.TabProperties.TabId == tabID {
			return t
		}
		if len(t.ChildTabs) > 0 {
			if found := findTab(t.ChildTabs, tabID); found != nil {
				return found
			}
		}
	}
	return nil
}

func fetchDocumentTabContent(docsSvc *docs.Service, fileID, tabID string) ([]byte, error) {
	doc, err := docsSvc.Documents.Get(fileID).IncludeTabsContent(true).Do()
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve document with tabs: %w", err)
	}

	if tab := findTab(doc.Tabs, tabID); tab != nil && tab.DocumentTab != nil {
		textContent := renderBodyAsText(tab.DocumentTab.Body)
		return []byte(textContent), nil
	}

	return nil, fmt.Errorf("tab with id %q not found", tabID)
}

func exportGoogleAppsFile(driveSvc *drive.Service, fileID, mimeType, format string) ([]byte, error) {
	exportMimeType, err := resolveExportMimeType(mimeType, format)
	if err != nil {
		return nil, err
	}

	resp, err := driveSvc.Files.Export(fileID, exportMimeType).Download()
	if err != nil {
		return nil, fmt.Errorf("unable to export Google Doc: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("unable to read exported content: %w", err)
	}
	return body, nil
}

func resolveExportMimeType(mimeType, format string) (string, error) {
	exportMimeType, ok := formatMap[strings.ToLower(format)]
	if !ok && format != "" {
		return "", fmt.Errorf("invalid format %q (valid formats: pdf, docx, html, zip, epub, txt, md, csv, tsv, xlsx, ods, pptx, odp)", format)
	}
	if mimeType == "application/vnd.google-apps.spreadsheet" && (format == "" || format == "txt") {
		return "text/csv", nil
	}
	if format == "" {
		return "text/plain", nil
	}
	return exportMimeType, nil
}

func downloadStandardFile(driveSvc *drive.Service, fileID string) ([]byte, error) {
	resp, err := driveSvc.Files.Get(fileID).Download()
	if err != nil {
		return nil, fmt.Errorf("unable to download file: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("unable to read file content: %w", err)
	}
	return body, nil
}
