package mcp

import (
	"context"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/ghchinoy/drivectl/internal/drive"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
)

// SlidesGetArgs defines the arguments for the slides get tool.
type SlidesGetArgs struct {
	PresentationID string `json:"presentation-id"`
	Format         string `json:"format"`
}

// SlidesNotesArgs defines the arguments for the slides notes tool.
type SlidesNotesArgs struct {
	PresentationID string `json:"presentation-id"`
	SlideNumber    int    `json:"slide-number"`
	Format         string `json:"format"`
}

// SlidesCreateArgs defines the arguments for the slides create tool.
type SlidesCreateArgs struct {
	Title string `json:"title"`
}

// SlidesCreateFromArgs defines the arguments for the slides create-from tool.
type SlidesCreateFromArgs struct {
	File  string `json:"file"`
	Type  string `json:"type"`
	Title string `json:"title"`
}

// SlidesAddArgs defines the arguments for the slides add tool.
type SlidesAddArgs struct {
	PresentationID string `json:"presentation-id"`
	Layout         string `json:"layout"`
	Title          string `json:"title"`
}

// SlidesAddImageArgs defines the arguments for the slides add-image tool.
type SlidesAddImageArgs struct {
	PresentationID string  `json:"presentation-id"`
	ImagePath      string  `json:"image-path"`
	SlideID        string  `json:"slide-id"`
	Left           float64 `json:"left"`
	Top            float64 `json:"top"`
	Width          float64 `json:"width"`
	Height         float64 `json:"height"`
}

// SlidesSetBackgroundArgs defines the arguments for the slides set-background tool.
type SlidesSetBackgroundArgs struct {
	PresentationID string `json:"presentation-id"`
	ImagePath      string `json:"image-path"`
	SlideID        string `json:"slide-id"`
}

// SlidesGetHandler is the handler for the slides get tool.
func SlidesGetHandler(ctx context.Context, ss *mcp.ServerSession, params *mcp.CallToolParamsFor[SlidesGetArgs]) (*mcp.CallToolResultFor[any], error) {
	if params.Arguments.PresentationID == "" {
		return nil, fmt.Errorf("presentation-id is a required argument")
	}

	slidesSvc, err := getSlidesSvc(ctx)
	if err != nil {
		return nil, err
	}

	if params.Arguments.Format == "png" || params.Arguments.Format == "jpg" {
		// Image export requires a directory, which is not supported in MCP yet.
		// We will return an error for now.
		return nil, fmt.Errorf("image export is not yet supported for the slides.get tool")
	} else if params.Arguments.Format != "" {
		driveSvc, err := getDriveSvc(ctx)
		if err != nil {
			return nil, err
		}
		docsSvc, err := getDocsSvc(ctx)
		if err != nil {
			return nil, err
		}
		content, err := drive.GetFile(driveSvc, docsSvc, params.Arguments.PresentationID, params.Arguments.Format, "", false)
		if err != nil {
			return nil, fmt.Errorf("unable to get presentation: %w", err)
		}
		return &mcp.CallToolResultFor[any]{
			Content: []mcp.Content{
				&mcp.TextContent{Text: string(content)},
			},
		}, nil
	} else {
		prez, err := drive.GetPresentation(slidesSvc, params.Arguments.PresentationID)
		if err != nil {
			return nil, fmt.Errorf("unable to get presentation: %w", err)
		}

		var text strings.Builder
		for _, slide := range prez.Slides {
			for _, element := range slide.PageElements {
				if element.Shape != nil && element.Shape.Text != nil {
					for _, textElement := range element.Shape.Text.TextElements {
						if textElement.TextRun != nil {
							text.WriteString(textElement.TextRun.Content)
						}
					}
				}
			}
		}
		return &mcp.CallToolResultFor[any]{
			Content: []mcp.Content{
				&mcp.TextContent{Text: text.String()},
			},
		}, nil
	}
}

// SlidesNotesHandler is the handler for the slides notes tool.
func SlidesNotesHandler(ctx context.Context, ss *mcp.ServerSession, params *mcp.CallToolParamsFor[SlidesNotesArgs]) (*mcp.CallToolResultFor[any], error) {
	if params.Arguments.PresentationID == "" {
		return nil, fmt.Errorf("presentation-id is a required argument")
	}

	slidesSvc, err := getSlidesSvc(ctx)
	if err != nil {
		return nil, err
	}

	notes, err := drive.GetPresentationNotes(slidesSvc, params.Arguments.PresentationID, params.Arguments.SlideNumber, params.Arguments.Format)
	if err != nil {
		return nil, fmt.Errorf("unable to get presentation notes: %w", err)
	}

	return &mcp.CallToolResultFor[any]{
		Content: []mcp.Content{
			&mcp.TextContent{Text: notes},
		},
	}, nil
}

// SlidesCreateHandler is the handler for the slides create tool.
func SlidesCreateHandler(ctx context.Context, ss *mcp.ServerSession, params *mcp.CallToolParamsFor[SlidesCreateArgs]) (*mcp.CallToolResultFor[any], error) {
	if params.Arguments.Title == "" {
		return nil, fmt.Errorf("title is a required argument")
	}

	slidesSvc, err := getSlidesSvc(ctx)
	if err != nil {
		return nil, err
	}

	prez, err := drive.CreatePresentation(slidesSvc, params.Arguments.Title)
	if err != nil {
		return nil, fmt.Errorf("unable to create presentation: %w", err)
	}

	return &mcp.CallToolResultFor[any]{
		Content: []mcp.Content{
			&mcp.TextContent{Text: fmt.Sprintf("Successfully created presentation with ID: %s", prez.PresentationId)},
		},
	}, nil
}

// SlidesCreateFromHandler is the handler for the slides create-from tool.
func SlidesCreateFromHandler(ctx context.Context, ss *mcp.ServerSession, params *mcp.CallToolParamsFor[SlidesCreateFromArgs]) (*mcp.CallToolResultFor[any], error) {
	if params.Arguments.File == "" {
		return nil, fmt.Errorf("file is a required argument")
	}

	content, err := ioutil.ReadFile(params.Arguments.File)
	if err != nil {
		return nil, fmt.Errorf("unable to read file: %w", err)
	}

	fileType := params.Arguments.Type
	if fileType == "" {
		ext := filepath.Ext(params.Arguments.File)
		if ext == ".md" || ext == ".markdown" {
			fileType = "markdown"
		} else if ext == ".slides" {
			fileType = "go-slides"
		} else {
			return nil, fmt.Errorf("unable to determine file type from extension: %s", ext)
		}
	}

	title := params.Arguments.Title
	if title == "" {
		// Get title from file content
		// For now, use the file name as the title
		title = strings.TrimSuffix(filepath.Base(params.Arguments.File), filepath.Ext(params.Arguments.File))
	}

	slidesSvc, err := getSlidesSvc(ctx)
	if err != nil {
		return nil, err
	}

	if fileType == "markdown" {
		prez, err := drive.CreatePresentationFromMarkdown(slidesSvc, title, string(content))
		if err != nil {
			return nil, err
		}
		return &mcp.CallToolResultFor[any]{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Successfully created presentation with ID: %s", prez.PresentationId)},
			},
		}, nil
	} else if fileType == "go-slides" {
		prez, err := drive.CreatePresentationFromGoSlides(slidesSvc, title, string(content))
		if err != nil {
			return nil, err
		}
		return &mcp.CallToolResultFor[any]{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Successfully created presentation with ID: %s", prez.PresentationId)},
			},
		}, nil
	} else {
		return nil, fmt.Errorf("unsupported file type: %s", fileType)
	}
}

// SlidesAddHandler is the handler for the slides add tool.
func SlidesAddHandler(ctx context.Context, ss *mcp.ServerSession, params *mcp.CallToolParamsFor[SlidesAddArgs]) (*mcp.CallToolResultFor[any], error) {
	if params.Arguments.PresentationID == "" {
		return nil, fmt.Errorf("presentation-id is a required argument")
	}

	slidesSvc, err := getSlidesSvc(ctx)
	if err != nil {
		return nil, err
	}

	_, err = drive.AddSlide(slidesSvc, params.Arguments.PresentationID, params.Arguments.Layout, params.Arguments.Title)
	if err != nil {
		return nil, fmt.Errorf("unable to add slide: %w", err)
	}

	return &mcp.CallToolResultFor[any]{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "Successfully added slide."},
		},
	}, nil
}

// SlidesAddImageHandler is the handler for the slides add-image tool.
func SlidesAddImageHandler(ctx context.Context, ss *mcp.ServerSession, params *mcp.CallToolParamsFor[SlidesAddImageArgs]) (*mcp.CallToolResultFor[any], error) {
	if params.Arguments.PresentationID == "" {
		return nil, fmt.Errorf("presentation-id is a required argument")
	}
	if params.Arguments.ImagePath == "" {
		return nil, fmt.Errorf("image-path is a required argument")
	}

	driveSvc, err := getDriveSvc(ctx)
	if err != nil {
		return nil, err
	}

	slidesSvc, err := getSlidesSvc(ctx)
	if err != nil {
		return nil, err
	}

	slideId := params.Arguments.SlideID
	if slideId == "" {
		prez, err := drive.GetPresentation(slidesSvc, params.Arguments.PresentationID)
		if err != nil {
			return nil, err
		}
		if len(prez.Slides) > 0 {
			slideId = prez.Slides[0].ObjectId
		} else {
			return nil, fmt.Errorf("presentation has no slides")
		}
	}

	err = drive.AddImage(driveSvc, slidesSvc, params.Arguments.PresentationID, slideId, params.Arguments.ImagePath, params.Arguments.Left, params.Arguments.Top, params.Arguments.Width, params.Arguments.Height)
	if err != nil {
		return nil, fmt.Errorf("unable to add image to slide: %w", err)
	}

	return &mcp.CallToolResultFor[any]{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "Successfully added image to slide."},
		},
	}, nil
}

// SlidesSetBackgroundHandler is the handler for the slides set-background tool.
func SlidesSetBackgroundHandler(ctx context.Context, ss *mcp.ServerSession, params *mcp.CallToolParamsFor[SlidesSetBackgroundArgs]) (*mcp.CallToolResultFor[any], error) {
	if params.Arguments.PresentationID == "" {
		return nil, fmt.Errorf("presentation-id is a required argument")
	}
	if params.Arguments.ImagePath == "" {
		return nil, fmt.Errorf("image-path is a required argument")
	}

	driveSvc, err := getDriveSvc(ctx)
	if err != nil {
		return nil, err
	}

	slidesSvc, err := getSlidesSvc(ctx)
	if err != nil {
		return nil, err
	}

	slideId := params.Arguments.SlideID
	if slideId == "" {
		prez, err := drive.GetPresentation(slidesSvc, params.Arguments.PresentationID)
		if err != nil {
			return nil, err
		}
		if len(prez.Slides) > 0 {
			slideId = prez.Slides[0].ObjectId
		} else {
			return nil, fmt.Errorf("presentation has no slides")
		}
	}

	err = drive.SetSlideBackgroundImage(driveSvc, slidesSvc, params.Arguments.PresentationID, slideId, params.Arguments.ImagePath)
	if err != nil {
		return nil, fmt.Errorf("unable to set background image: %w", err)
	}

	return &mcp.CallToolResultFor[any]{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "Successfully set background image."},
		},
	}, nil
}

func RegisterSlidesTools(server *mcp.Server, rootCmd *cobra.Command) {
	for _, cmd := range rootCmd.Commands() {
		command := cmd

		if command.Name() == "slides" {
			for _, subCmd := range command.Commands() {
				subCommand := subCmd
				switch subCommand.Name() {
				case "get":
					mcp.AddTool(server, &mcp.Tool{
						Name:        "slides.get",
						Description: subCommand.Long,
					}, SlidesGetHandler)
				case "notes":
					mcp.AddTool(server, &mcp.Tool{
						Name:        "slides.notes",
						Description: subCommand.Long,
					}, SlidesNotesHandler)
				case "create":
					mcp.AddTool(server, &mcp.Tool{
						Name:        "slides.create",
						Description: subCommand.Long,
					}, SlidesCreateHandler)
				case "create-from":
					mcp.AddTool(server, &mcp.Tool{
						Name:        "slides.create-from",
						Description: subCommand.Long,
					}, SlidesCreateFromHandler)
				case "add":
					mcp.AddTool(server, &mcp.Tool{
						Name:        "slides.add",
						Description: subCommand.Long,
					}, SlidesAddHandler)
				case "add-image":
					mcp.AddTool(server, &mcp.Tool{
						Name:        "slides.add-image",
						Description: subCommand.Long,
					}, SlidesAddImageHandler)
				case "set-background":
					mcp.AddTool(server, &mcp.Tool{
						Name:        "slides.set-background",
						Description: subCommand.Long,
					}, SlidesSetBackgroundHandler)
				}
			}
		}
	}
}
