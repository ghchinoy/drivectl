package drive

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/text"
	"golang.org/x/tools/present"
	googledrive "google.golang.org/api/drive/v3"
	"google.golang.org/api/slides/v1"
)

// GetPresentation retrieves a presentation.
func GetPresentation(slidesSvc *slides.Service, presentationId string) (*slides.Presentation, error) {
	prez, err := slidesSvc.Presentations.Get(presentationId).Fields("*").Do()
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve presentation: %w", err)
	}
	return prez, nil
}

// ExportSlidesAsImages exports each slide of a presentation as an image.
func ExportSlidesAsImages(slidesSvc *slides.Service, presentationId string, outputDir string, format string) error {
	prez, err := GetPresentation(slidesSvc, presentationId)
	if err != nil {
		return err
	}

	for i, slide := range prez.Slides {
		thumb, err := slidesSvc.Presentations.Pages.GetThumbnail(presentationId, slide.ObjectId).Do()
		if err != nil {
			return fmt.Errorf("unable to get thumbnail for slide %s: %w", slide.ObjectId, err)
		}

		resp, err := http.Get(thumb.ContentUrl)
		if err != nil {
			return fmt.Errorf("failed to download thumbnail from %s: %w", thumb.ContentUrl, err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read thumbnail content: %w", err)
		}

		fileName := fmt.Sprintf("slide_%d.%s", i+1, format)
		filePath := filepath.Join(outputDir, fileName)
		err = os.WriteFile(filePath, body, 0644)
		if err != nil {
			return fmt.Errorf("failed to write image to %s: %w", filePath, err)
		}
	}

	return nil
}

// GetPresentationNotes retrieves the speaker notes from a presentation.
func GetPresentationNotes(slidesSvc *slides.Service, presentationId string, slideNumber int, format string) (string, error) {
	prez, err := GetPresentation(slidesSvc, presentationId)
	if err != nil {
		return "", err
	}

	var notes strings.Builder

	processSlide := func(slide *slides.Page, slideIndex int) error {
		page, err := slidesSvc.Presentations.Pages.Get(presentationId, slide.ObjectId).Do()
		if err != nil {
			return fmt.Errorf("unable to get page %s: %w", slide.ObjectId, err)
		}

		if page.NotesProperties == nil || page.NotesProperties.SpeakerNotesObjectId == "" {
			return nil
		}

		note, err := extractNotesFromSlide(slidesSvc, presentationId, page, format)
		if err != nil {
			return err
		}
		if format == "md" {
			notes.WriteString(fmt.Sprintf("## Slide %d\n\n", slideIndex))
		} else {
			notes.WriteString(fmt.Sprintf("Slide %d:\n", slideIndex))
		}
		notes.WriteString(note)
		notes.WriteString("\n")
		return nil
	}

	if slideNumber > 0 {
		if slideNumber > len(prez.Slides) {
			return "", fmt.Errorf("invalid slide number: %d. Presentation has only %d slides", slideNumber, len(prez.Slides))
		}
		slide := prez.Slides[slideNumber-1]
		if err := processSlide(slide, slideNumber); err != nil {
			return "", err
		}
	} else {
		for i, slide := range prez.Slides {
			if err := processSlide(slide, i+1); err != nil {
				return "", err
			}
		}
	}

	return notes.String(), nil
}

func extractNotesFromSlide(slidesSvc *slides.Service, presentationId string, slide *slides.Page, format string) (string, error) {
	if slide.NotesProperties == nil || slide.NotesProperties.SpeakerNotesObjectId == "" {
		return "", nil
	}

	notesPage, err := slidesSvc.Presentations.Pages.Get(presentationId, slide.NotesProperties.SpeakerNotesObjectId).Do()
	if err != nil {
		return "", fmt.Errorf("unable to get notes page %s: %w", slide.NotesProperties.SpeakerNotesObjectId, err)
	}

	var notes strings.Builder
	for _, element := range notesPage.PageElements {
		if element.Shape != nil && element.Shape.Placeholder != nil && element.Shape.Placeholder.Type == "BODY" {
			if element.Shape.Text != nil {
				for _, textElement := range element.Shape.Text.TextElements {
					if textElement.TextRun != nil {
						notes.WriteString(textElement.TextRun.Content)
					}
				}
			}
		}
	}

	return notes.String(), nil
}

// CreatePresentation creates a new, blank presentation.
func CreatePresentation(slidesSvc *slides.Service, title string) (*slides.Presentation, error) {
	prez := &slides.Presentation{
		Title: title,
	}
	createdPrez, err := slidesSvc.Presentations.Create(prez).Do()
	if err != nil {
		return nil, fmt.Errorf("unable to create presentation: %w", err)
	}
	return createdPrez, nil
}

// CreatePresentationFromMarkdown creates a new presentation from a Markdown string.
func CreatePresentationFromMarkdown(slidesSvc *slides.Service, title string, markdownContent string) (*slides.Presentation, error) {
	// 1. Create a new presentation
	prez := &slides.Presentation{
		Title: title,
	}
	createdPrez, err := slidesSvc.Presentations.Create(prez).Do()
	if err != nil {
		return nil, fmt.Errorf("unable to create presentation: %w", err)
	}

	// 2. Parse the Markdown
	parser := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
		),
	).Parser()
	root := parser.Parse(text.NewReader([]byte(markdownContent)))

	// 3. Create slides and add content
	var requests []*slides.Request
	var isFirstSlide = true

	for n := root.FirstChild(); n != nil; n = n.NextSibling() {
		switch n.Kind() {
		case ast.KindHeading:
			heading := n.(*ast.Heading)
			var headingText strings.Builder
			for c := n.FirstChild(); c != nil; c = c.NextSibling() {
				if c.Kind() == ast.KindText {
					headingText.WriteString(string(c.(*ast.Text).Segment.Value([]byte(markdownContent))))
				}
			}
			text := headingText.String()

			if isFirstSlide && heading.Level == 1 {
				// This is the title slide
				slide := createdPrez.Slides[0]
				var titleShapeId string
				for _, el := range slide.PageElements {
					if el.Shape != nil && el.Shape.Placeholder != nil && el.Shape.Placeholder.Type == "CENTERED_TITLE" {
						titleShapeId = el.ObjectId
						break
					}
				}

				if titleShapeId != "" {
					requests = append(requests, &slides.Request{
						InsertText: &slides.InsertTextRequest{
							ObjectId: titleShapeId,
							Text:     text,
						},
					})
				}

				// Check for subtitle
				if nextNode := n.NextSibling(); nextNode != nil && nextNode.Kind() == ast.KindParagraph {
					var paragraphText strings.Builder
					for c := nextNode.FirstChild(); c != nil; c = c.NextSibling() {
						if c.Kind() == ast.KindText {
							paragraphText.WriteString(string(c.(*ast.Text).Segment.Value([]byte(markdownContent))))
						}
					}
					subtitleText := paragraphText.String()

					var subtitleShapeId string
					for _, el := range slide.PageElements {
						if el.Shape != nil && el.Shape.Placeholder != nil && el.Shape.Placeholder.Type == "SUBTITLE" {
							subtitleShapeId = el.ObjectId
							break
						}
					}

					if subtitleShapeId != "" {
						requests = append(requests, &slides.Request{
							InsertText: &slides.InsertTextRequest{
								ObjectId: subtitleShapeId,
								Text:     subtitleText,
							},
						})
					}
					n = nextNode // Skip the paragraph node
				}

				isFirstSlide = false
			} else {
				// This is a new slide
				slideId := fmt.Sprintf("slide_%d", len(requests))
				titleId := fmt.Sprintf("title_%d", len(requests))
				bodyId := fmt.Sprintf("body_%d", len(requests))

				requests = append(requests, &slides.Request{
					CreateSlide: &slides.CreateSlideRequest{
						ObjectId: slideId,
						SlideLayoutReference: &slides.LayoutReference{
							PredefinedLayout: "TITLE_AND_BODY",
						},
						PlaceholderIdMappings: []*slides.LayoutPlaceholderIdMapping{
							{
								LayoutPlaceholder: &slides.Placeholder{Type: "TITLE"},
								ObjectId:          titleId,
							},
							{
								LayoutPlaceholder: &slides.Placeholder{Type: "BODY"},
								ObjectId:          bodyId,
							},
						},
					},
				})

				requests = append(requests, &slides.Request{
					InsertText: &slides.InsertTextRequest{
						ObjectId: titleId,
						Text:     text,
					},
				})

				// Check for body
				if nextNode := n.NextSibling(); nextNode != nil && nextNode.Kind() == ast.KindParagraph {
					var paragraphText strings.Builder
					for c := nextNode.FirstChild(); c != nil; c = c.NextSibling() {
						if c.Kind() == ast.KindText {
							paragraphText.WriteString(string(c.(*ast.Text).Segment.Value([]byte(markdownContent))))
						}
					}
					bodyText := paragraphText.String()

					requests = append(requests, &slides.Request{
						InsertText: &slides.InsertTextRequest{
							ObjectId: bodyId,
							Text:     bodyText,
						},
					})
					n = nextNode // Skip the paragraph node
				}
			}
		}
	}

	if len(requests) > 0 {
		_, err = slidesSvc.Presentations.BatchUpdate(createdPrez.PresentationId, &slides.BatchUpdatePresentationRequest{
			Requests: requests,
		}).Do()
		if err != nil {
			return nil, fmt.Errorf("unable to update presentation: %w", err)
		}
	}

	return createdPrez, nil
}

// CreatePresentationFromGoSlides creates a new presentation from a Go Slides file.
func CreatePresentationFromGoSlides(slidesSvc *slides.Service, title string, goSlidesContent string) (*slides.Presentation, error) {
	// 1. Parse the Go Slides content
	doc, err := present.Parse(strings.NewReader(goSlidesContent), "prez.slide", 0)
	if err != nil {
		return nil, fmt.Errorf("unable to parse Go Slides content: %w", err)
	}

	// 2. Create a new presentation
	if title == "" {
		title = doc.Title
	}
	prez := &slides.Presentation{
		Title: title,
	}
	createdPrez, err := slidesSvc.Presentations.Create(prez).Do()
	if err != nil {
		return nil, fmt.Errorf("unable to create presentation: %w", err)
	}

	// 3. Create slides and add content
	var requests []*slides.Request

	// Title slide
	var titleShapeId string
	for _, el := range createdPrez.Slides[0].PageElements {
		if el.Shape != nil && el.Shape.Placeholder != nil && el.Shape.Placeholder.Type == "CENTERED_TITLE" {
			titleShapeId = el.ObjectId
			break
		}
	}
	if titleShapeId != "" {
		requests = append(requests, &slides.Request{
			InsertText: &slides.InsertTextRequest{
				ObjectId: titleShapeId,
				Text:     doc.Title,
			},
		})
	}

	// Subtitle
	var subtitleShapeId string
	for _, el := range createdPrez.Slides[0].PageElements {
		if el.Shape != nil && el.Shape.Placeholder != nil && el.Shape.Placeholder.Type == "SUBTITLE" {
			subtitleShapeId = el.ObjectId
			break
		}
	}
	if subtitleShapeId != "" {
		requests = append(requests, &slides.Request{
			InsertText: &slides.InsertTextRequest{
				ObjectId: subtitleShapeId,
				Text:     doc.Subtitle,
			},
		})
	}

	// Sections
	for _, section := range doc.Sections {
		slideId := fmt.Sprintf("slide_%d", len(requests))
		titleId := fmt.Sprintf("title_%d", len(requests))
		bodyId := fmt.Sprintf("body_%d", len(requests))

		requests = append(requests, &slides.Request{
			CreateSlide: &slides.CreateSlideRequest{
				ObjectId: slideId,
				SlideLayoutReference: &slides.LayoutReference{
					PredefinedLayout: "TITLE_AND_BODY",
				},
				PlaceholderIdMappings: []*slides.LayoutPlaceholderIdMapping{
					{
						LayoutPlaceholder: &slides.Placeholder{Type: "TITLE"},
						ObjectId:          titleId,
					},
					{
						LayoutPlaceholder: &slides.Placeholder{Type: "BODY"},
						ObjectId:          bodyId,
					},
				},
			},
		})

		requests = append(requests, &slides.Request{
			InsertText: &slides.InsertTextRequest{
				ObjectId: titleId,
				Text:     section.Title,
			},
		})

		var bodyText strings.Builder
		for _, elem := range section.Elem {
			switch v := elem.(type) {
			case present.Text:
				bodyText.WriteString(strings.Join(v.Lines, "\n"))
			}
		}

		requests = append(requests, &slides.Request{
			InsertText: &slides.InsertTextRequest{
				ObjectId: bodyId,
				Text:     bodyText.String(),
			},
		})
	}

	if len(requests) > 0 {
		_, err = slidesSvc.Presentations.BatchUpdate(createdPrez.PresentationId, &slides.BatchUpdatePresentationRequest{
			Requests: requests,
		}).Do()
		if err != nil {
			return nil, fmt.Errorf("unable to update presentation: %w", err)
		}
	}

	return createdPrez, nil
}

func AddImage(driveSvc *googledrive.Service, slidesSvc *slides.Service, presentationId string, slideId string, imagePath string, left float64, top float64, width float64, height float64) error {
	// 1. Upload the image to Google Drive
	file, err := os.Open(imagePath)
	if err != nil {
		return fmt.Errorf("unable to open image file: %w", err)
	}
	defer file.Close()

	// Get the content type of the image
	buf := make([]byte, 512)
	_, err = file.Read(buf)
	if err != nil && err != io.EOF {
		return err
	}
	_, err = file.Seek(0, 0)
	if err != nil {
		return err
	}
	contentType := http.DetectContentType(buf)

	driveFile, err := UploadFile(driveSvc, filepath.Base(imagePath), contentType, file)
	if err != nil {
		return fmt.Errorf("unable to upload image to drive: %w", err)
	}

	// 2. Get the URL of the uploaded image
	driveFile, err = driveSvc.Files.Get(driveFile.Id).Fields("webContentLink").Do()
	if err != nil {
		return fmt.Errorf("unable to get image web content link: %w", err)
	}
	imageUrl := driveFile.WebContentLink

	// 3. Add the image to the slide
	imageId := fmt.Sprintf("image_%d", time.Now().UnixNano())
	requests := []*slides.Request{
		{
			CreateImage: &slides.CreateImageRequest{
				ObjectId: imageId,
				Url:      imageUrl,
				ElementProperties: &slides.PageElementProperties{
					PageObjectId: slideId,
					Size: &slides.Size{
						Height: &slides.Dimension{Magnitude: height, Unit: "PT"},
						Width:  &slides.Dimension{Magnitude: width, Unit: "PT"},
					},
					Transform: &slides.AffineTransform{
						ScaleX:     1,
						ScaleY:     1,
						TranslateX: left,
						TranslateY: top,
						Unit:       "PT",
					},
				},
			},
		},
	}

	batchUpdate := &slides.BatchUpdatePresentationRequest{
		Requests: requests,
	}

	_, err = slidesSvc.Presentations.BatchUpdate(presentationId, batchUpdate).Do()
	if err != nil {
		return fmt.Errorf("unable to add image to slide: %w", err)
	}

	return nil
}

// SetSlideBackgroundImage sets the background image of a slide.
func SetSlideBackgroundImage(driveSvc *googledrive.Service, slidesSvc *slides.Service, presentationId string, slideId string, imagePath string) error {
	// 1. Upload the image to Google Drive
	file, err := os.Open(imagePath)
	if err != nil {
		return fmt.Errorf("unable to open image file: %w", err)
	}
	defer file.Close()

	// Get the content type of the image
	buf := make([]byte, 512)
	_, err = file.Read(buf)
	if err != nil && err != io.EOF {
		return err
	}
	_, err = file.Seek(0, 0)
	if err != nil {
		return err
	}
	contentType := http.DetectContentType(buf)

	driveFile, err := UploadFile(driveSvc, filepath.Base(imagePath), contentType, file)
	if err != nil {
		return fmt.Errorf("unable to upload image to drive: %w", err)
	}

	// 2. Make the file public to get a public URL.
	_, err = driveSvc.Permissions.Create(driveFile.Id, &googledrive.Permission{Type: "anyone", Role: "reader"}).Do()
	if err != nil {
		return fmt.Errorf("unable to make image public: %w", err)
	}

	// 3. Get the URL of the uploaded image
	driveFile, err = driveSvc.Files.Get(driveFile.Id).Fields("webContentLink").Do()
	if err != nil {
		return fmt.Errorf("unable to get image web content link: %w", err)
	}
	imageUrl := driveFile.WebContentLink

	// 4. Set the background image of the slide
	requests := []*slides.Request{
		{
			UpdatePageProperties: &slides.UpdatePagePropertiesRequest{
				ObjectId: slideId,
				PageProperties: &slides.PageProperties{
					PageBackgroundFill: &slides.PageBackgroundFill{
						StretchedPictureFill: &slides.StretchedPictureFill{
							ContentUrl: imageUrl,
						},
					},
				},
				Fields: "pageBackgroundFill",
			},
		},
	}

	batchUpdate := &slides.BatchUpdatePresentationRequest{
		Requests: requests,
	}

	_, err = slidesSvc.Presentations.BatchUpdate(presentationId, batchUpdate).Do()
	if err != nil {
		return fmt.Errorf("unable to set background image: %w", err)
	}

	return nil
}

// AddSlide adds a new slide to a presentation.
func AddSlide(slidesSvc *slides.Service, presentationId string, layout string, title string) (*slides.BatchUpdatePresentationResponse, error) {
	slideId := fmt.Sprintf("slide_%d", time.Now().UnixNano())
	titleId := fmt.Sprintf("title_%d", time.Now().UnixNano())

	requests := []*slides.Request{
		{
			CreateSlide: &slides.CreateSlideRequest{
				ObjectId: slideId,
				SlideLayoutReference: &slides.LayoutReference{
					PredefinedLayout: layout,
				},
				PlaceholderIdMappings: []*slides.LayoutPlaceholderIdMapping{
					{
						LayoutPlaceholder: &slides.Placeholder{Type: "TITLE"},
						ObjectId:          titleId,
					},
				},
			},
		},
	}

	if title != "" {
		requests = append(requests, &slides.Request{
			InsertText: &slides.InsertTextRequest{
				ObjectId: titleId,
				Text:     title,
			},
		})
	}

	batchUpdate := &slides.BatchUpdatePresentationRequest{
		Requests: requests,
	}

	resp, err := slidesSvc.Presentations.BatchUpdate(presentationId, batchUpdate).Do()
	if err != nil {
		return nil, fmt.Errorf("unable to add slide: %w", err)
	}

	return resp, nil
}
