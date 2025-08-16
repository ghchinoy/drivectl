package mcp

import (
	"context"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/ghchinoy/drivectl/internal/drive"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
)

// TabsArgs defines the arguments for the tabs tool.
type TabsArgs struct {
	DocumentID string `json:"document-id"`
}

// DocsCreateArgs defines the arguments for the docs create tool.
type DocsCreateArgs struct {
	Title        string `json:"title"`
	MarkdownFile string `json:"markdown_file,omitempty"`
	MarkdownText string `json:"markdown_text,omitempty"`
}

// DocsTabsHandler is the handler for the docs tabs tool.
func DocsTabsHandler(ctx context.Context, ss *mcp.ServerSession, params *mcp.CallToolParamsFor[TabsArgs]) (*mcp.CallToolResultFor[any], error) {
	if params.Arguments.DocumentID == "" {
		return nil, fmt.Errorf("document-id is a required argument")
	}

	docsSvc, err := getDocsSvc(ctx)
	if err != nil {
		return nil, err
	}

	tabs, err := drive.GetTabs(docsSvc, params.Arguments.DocumentID)
	if err != nil {
		return nil, fmt.Errorf("unable to get tabs: %w", err)
	}

	var tabStrings []string
	var traverse func(tabs []*drive.TabInfo, level int)
	traverse = func(tabs []*drive.TabInfo, level int) {
		for _, tab := range tabs {
			tabStrings = append(tabStrings, fmt.Sprintf("%s%s (%s)", strings.Repeat("\t", level), tab.Title, tab.TabID))
			if len(tab.Children) > 0 {
				traverse(tab.Children, level+1)
			}
		}
	}
	traverse(tabs, 0)

	return &mcp.CallToolResultFor[any]{
		Content: []mcp.Content{
			&mcp.TextContent{Text: strings.Join(tabStrings, "\n")},
		},
	}, nil
}

// DocsCreateHandler is the handler for the docs create tool.
func DocsCreateHandler(ctx context.Context, ss *mcp.ServerSession, params *mcp.CallToolParamsFor[DocsCreateArgs]) (*mcp.CallToolResultFor[any], error) {
	if params.Arguments.Title == "" {
		return nil, fmt.Errorf("title is a required argument")
	}
	if params.Arguments.MarkdownFile == "" && params.Arguments.MarkdownText == "" {
		return nil, fmt.Errorf("either markdown_file or markdown_text is required")
	}
	if params.Arguments.MarkdownFile != "" && params.Arguments.MarkdownText != "" {
		return nil, fmt.Errorf("only one of markdown_file or markdown_text can be provided")
	}

	var markdownContent string
	if params.Arguments.MarkdownFile != "" {
		content, err := ioutil.ReadFile(params.Arguments.MarkdownFile)
		if err != nil {
			return nil, fmt.Errorf("unable to read markdown file: %w", err)
		}
		markdownContent = string(content)
	} else {
		markdownContent = params.Arguments.MarkdownText
	}

	docsSvc, err := getDocsSvc(ctx)
	if err != nil {
		return nil, err
	}

	doc, err := drive.CreateDocFromMarkdown(docsSvc, params.Arguments.Title, markdownContent)
	if err != nil {
		return nil, err
	}

	return &mcp.CallToolResultFor[any]{
		Content: []mcp.Content{
			&mcp.TextContent{Text: fmt.Sprintf("Successfully created document %s (%s)", doc.Title, doc.DocumentId)},
		},
	}, nil
}

func RegisterDocsTools(server *mcp.Server, rootCmd *cobra.Command) {
	for _, cmd := range rootCmd.Commands() {
		command := cmd

		if command.Name() == "docs" {
			for _, subCmd := range command.Commands() {
				subCommand := subCmd
				switch subCommand.Name() {
				case "tabs":
					mcp.AddTool(server, &mcp.Tool{
						Name:        "docs.tabs",
						Description: subCommand.Long,
					}, DocsTabsHandler)
				case "create":
					mcp.AddTool(server, &mcp.Tool{
						Name:        "docs.create",
						Description: subCommand.Long,
					}, DocsCreateHandler)
				}
			}
		}
	}
}