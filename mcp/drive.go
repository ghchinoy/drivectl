package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ghchinoy/drivectl/internal/drive"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
)

// ListArgs defines the arguments for the list tool.
type ListArgs struct {
	Limit int64  `json:"limit"`
	Query string `json:"query"`
}

// GetArgs defines the arguments for the get tool.
type GetArgs struct {
	FileID string `json:"file-id"`
	Format string `json:"format"`
	TabID  string `json:"tab-id"`
}

// DescribeArgs defines the arguments for the describe tool.
type DescribeArgs struct {
	FileID string `json:"file-id"`
}

// ListHandler is the handler for the list tool.
func ListHandler(ctx context.Context, ss *mcp.ServerSession, params *mcp.CallToolParamsFor[ListArgs]) (*mcp.CallToolResultFor[any], error) {
	driveSvc, err := getDriveSvc(ctx)
	if err != nil {
		return nil, err
	}

	limit := params.Arguments.Limit
	if limit == 0 {
		limit = 100
	}
	files, err := drive.ListFiles(driveSvc, limit, params.Arguments.Query)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve files: %w", err)
	}

	var output string
	if len(files) == 0 {
		output = "No files found."
	} else {
		for _, i := range files {
			output += fmt.Sprintf("%s (%s)\n", i.Name, i.Id)
		}
	}

	return &mcp.CallToolResultFor[any]{
		Content: []mcp.Content{
			&mcp.TextContent{Text: output},
		},
	}, nil
}

// GetHandler is the handler for the get tool.
func GetHandler(ctx context.Context, ss *mcp.ServerSession, params *mcp.CallToolParamsFor[GetArgs]) (*mcp.CallToolResultFor[any], error) {
	driveSvc, err := getDriveSvc(ctx)
	if err != nil {
		return nil, err
	}
	docsSvc, err := getDocsSvc(ctx)
	if err != nil {
		return nil, err
	}

	content, err := drive.GetFile(driveSvc, docsSvc, params.Arguments.FileID, params.Arguments.Format, params.Arguments.TabID)
	if err != nil {
		return nil, fmt.Errorf("unable to get file: %w", err)
	}

	return &mcp.CallToolResultFor[any]{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(content)},
		},
	}, nil
}

// DescribeHandler is the handler for the describe tool.
func DescribeHandler(ctx context.Context, ss *mcp.ServerSession, params *mcp.CallToolParamsFor[DescribeArgs]) (*mcp.CallToolResultFor[any], error) {
	if params.Arguments.FileID == "" {
		return nil, fmt.Errorf("file-id is a required argument")
	}

	driveSvc, err := getDriveSvc(ctx)
	if err != nil {
		return nil, err
	}

	file, err := drive.DescribeFile(driveSvc, params.Arguments.FileID)
	if err != nil {
		return nil, fmt.Errorf("unable to describe file: %w", err)
	}

	jsonFile, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("unable to marshal file to json: %w", err)
	}

	return &mcp.CallToolResultFor[any]{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(jsonFile)},
		},
	}, nil
}

func RegisterDriveTools(server *mcp.Server, rootCmd *cobra.Command) {
	for _, cmd := range rootCmd.Commands() {
		command := cmd

		switch command.Name() {
		case "list":
			mcp.AddTool(server, &mcp.Tool{
				Name:        command.Name(),
				Description: command.Long,
			}, ListHandler)
		case "get":
			mcp.AddTool(server, &mcp.Tool{
				Name:        command.Name(),
				Description: command.Long,
			}, GetHandler)
		case "describe":
			mcp.AddTool(server, &mcp.Tool{
				Name:        command.Name(),
				Description: command.Long,
			}, DescribeHandler)
		}
	}
}
