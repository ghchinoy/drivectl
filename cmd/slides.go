package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/ghchinoy/drivectl/internal/drive"
	"github.com/spf13/cobra"
)

var (
	slidesFormat         string
	slidesOutputFile     string
	slideNumber          int
	notesFormat          string
	notesOutputFile      string
	createFromFileType   string
	createFromFileTitle  string
	addSlideLayout       string
	addSlideTitle        string
	addImageSlideID      string
	addImageURL          string
	addImageLeft         float64
	addImageTop          float64
	addImageWidth        float64
	addImageHeight       float64
	setBackgroundSlideID string
)

var slidesCmd = &cobra.Command{
	Use:   "slides",
	Short: "Interact with Google Slides",
	Long:  `A set of commands to interact with Google Slides.`,
}

var slidesGetCmd = &cobra.Command{
	Use:   "get [presentationId]",
	Short: "Gets a presentation.",
	Long:  `Retrieves a presentation and outputs it in a specified format.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		presentationId := args[0]

		if slidesFormat == "png" || slidesFormat == "jpg" {
			if slidesOutputFile == "" {
				return fmt.Errorf("output directory must be specified with the -o flag for image export")
			}
			err := drive.ExportSlidesAsImages(slidesSvc, presentationId, slidesOutputFile, slidesFormat)
			if err != nil {
				return err
			}
			fmt.Printf("Successfully exported slides to %s\n", slidesOutputFile)
		} else if slidesFormat != "" {
			content, err := drive.GetFile(driveSvc, docsSvc, presentationId, slidesFormat, "")
			if err != nil {
				return err
			}

			if slidesOutputFile != "" {
				err := os.WriteFile(slidesOutputFile, content, 0644)
				if err != nil {
					return fmt.Errorf("failed to write to output file %s: %w", slidesOutputFile, err)
				}
				fmt.Printf("Successfully saved presentation to %s\n", slidesOutputFile)
			} else {
				fmt.Println(string(content))
			}
		} else {
			prez, err := drive.GetPresentation(slidesSvc, presentationId)
			if err != nil {
				return err
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
			fmt.Println(text.String())
		}

		return nil
	},
}

var slidesNotesCmd = &cobra.Command{
	Use:   "notes [presentationId]",
	Short: "Gets the speaker notes from a presentation.",
	Long:  `Retrieves the speaker notes from a presentation and outputs them in a specified format.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		presentationId := args[0]

		notes, err := drive.GetPresentationNotes(slidesSvc, presentationId, slideNumber, notesFormat)
		if err != nil {
			return err
		}

		if notesOutputFile != "" {
			err := os.WriteFile(notesOutputFile, []byte(notes), 0644)
			if err != nil {
				return fmt.Errorf("failed to write to output file %s: %w", notesOutputFile, err)
			}
			fmt.Printf("Successfully saved notes to %s\n", notesOutputFile)
		} else {
			fmt.Println(notes)
		}

		return nil
	},
}

var slidesCreateCmd = &cobra.Command{
	Use:   "create [title]",
	Short: "Creates a new, blank presentation.",
	Long:  `Creates a new, blank presentation with the given title.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		title := args[0]
		prez, err := drive.CreatePresentation(slidesSvc, title)
		if err != nil {
			return err
		}
		fmt.Printf("Successfully created presentation with ID: %s\n", prez.PresentationId)
		return nil
	},
}

var slidesCreateFromCmd = &cobra.Command{
	Use:   "create-from [file]",
	Short: "Creates a new presentation from a source file.",
	Long:  `Creates a new presentation from a source file. Supported file types are Markdown (.md) and Go's .slides format.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath := args[0]

		content, err := ioutil.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("unable to read file: %w", err)
		}

		fileType := createFromFileType
		if fileType == "" {
			ext := filepath.Ext(filePath)
			if ext == ".md" || ext == ".markdown" {
				fileType = "markdown"
			} else if ext == ".slides" {
				fileType = "go-slides"
			} else {
				return fmt.Errorf("unable to determine file type from extension: %s", ext)
			}
		}

		title := createFromFileTitle
		if title == "" {
			// Get title from file content
			// For now, use the file name as the title
			title = strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))
		}

		if fileType == "markdown" {
			prez, err := drive.CreatePresentationFromMarkdown(slidesSvc, title, string(content))
			if err != nil {
				return err
			}
			fmt.Printf("Successfully created presentation with ID: %s\n", prez.PresentationId)
		} else if fileType == "go-slides" {
			prez, err := drive.CreatePresentationFromGoSlides(slidesSvc, title, string(content))
			if err != nil {
				return err
			}
			fmt.Printf("Successfully created presentation with ID: %s\n", prez.PresentationId)
		} else {
			return fmt.Errorf("unsupported file type: %s", fileType)
		}

		return nil
	},
}

var slidesAddCmd = &cobra.Command{
	Use:   "add [presentationId]",
	Short: "Adds a new slide to a presentation.",
	Long:  `Adds a new, blank slide to an existing presentation.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		presentationId := args[0]
		_, err := drive.AddSlide(slidesSvc, presentationId, addSlideLayout, addSlideTitle)
		if err != nil {
			return err
		}
		fmt.Println("Successfully added slide.")
		return nil
	},
}

var slidesAddImageCmd = &cobra.Command{
	Use:   "add-image [presentationId] [image-path]",
	Short: "Adds an image to a slide.",
	Long:  `Adds an image to a slide from a local file path. IMPORTANT: This command will upload the image to your Google Drive and make it publicly accessible so that the Slides API can access it.`,
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		presentationId := args[0]
		imagePath := args[1]

		slideId := addImageSlideID
		if slideId == "" {
			prez, err := drive.GetPresentation(slidesSvc, presentationId)
			if err != nil {
				return err
			}
			if len(prez.Slides) > 0 {
				slideId = prez.Slides[0].ObjectId
			} else {
				return fmt.Errorf("presentation has no slides")
			}
		}

		err := drive.AddImage(driveSvc, slidesSvc, presentationId, slideId, imagePath, addImageLeft, addImageTop, addImageWidth, addImageHeight)
		if err != nil {
			return err
		}

		fmt.Println("Successfully added image to slide.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(slidesCmd)
	slidesCmd.AddCommand(slidesGetCmd)
	slidesCmd.AddCommand(slidesNotesCmd)
	slidesCmd.AddCommand(slidesCreateCmd)
	slidesCmd.AddCommand(slidesCreateFromCmd)
	slidesCmd.AddCommand(slidesAddCmd)
	slidesCmd.AddCommand(slidesAddImageCmd)

	slidesCreateFromCmd.Flags().StringVar(&createFromFileType, "type", "", "The type of the source file (markdown, go-slides). If not provided, it will be inferred from the file extension.")
	slidesCreateFromCmd.Flags().StringVar(&createFromFileTitle, "title", "", "The title of the presentation. If not provided, it will be taken from the source file.")

	slidesAddCmd.Flags().StringVar(&addSlideLayout, "layout", "TITLE_AND_BODY", "The layout for the new slide.")
	slidesAddCmd.Flags().StringVar(&addSlideTitle, "title", "", "The title for the new slide.")

	slidesAddImageCmd.Flags().StringVar(&addImageSlideID, "slide-id", "", "The ID of the slide to add the image to. If not provided, the image will be added to the first slide.")
	slidesAddImageCmd.Flags().Float64Var(&addImageLeft, "left", 50, "The left position of the image in points.")
	slidesAddImageCmd.Flags().Float64Var(&addImageTop, "top", 50, "The top position of the image in points.")
	slidesAddImageCmd.Flags().Float64Var(&addImageWidth, "width", 0, "The width of the image in points. If 0, the original width will be used.")
	slidesAddImageCmd.Flags().Float64Var(&addImageHeight, "height", 0, "The height of the image in points. If 0, the original height will be used.")

	slidesGetCmd.Flags().StringVar(&slidesFormat, "format", "", "Format to export the presentation (pdf, png, jpg)")
	slidesGetCmd.Flags().StringVarP(&slidesOutputFile, "output", "o", "", "Path to save the output file")

	slidesNotesCmd.Flags().IntVar(&slideNumber, "slide-number", 0, "The slide number to get notes from (1-based). If 0, get notes from all slides.")
	slidesNotesCmd.Flags().StringVar(&notesFormat, "format", "txt", "Format for the notes (txt, md)")
	slidesNotesCmd.Flags().StringVarP(&notesOutputFile, "output", "o", "", "Path to save the output file")
}
