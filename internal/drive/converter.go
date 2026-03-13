package drive

import (
	"fmt"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/text"
	"google.golang.org/api/docs/v1"
)

// extractText recursively extracts text from an ast.Node
func extractText(n ast.Node, source []byte) string {
	var b strings.Builder
	err := ast.Walk(n, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering && node.Kind() == ast.KindText {
			b.Write(node.(*ast.Text).Segment.Value(source))
		}
		return ast.WalkContinue, nil
	})
	if err != nil {
		return ""
	}
	return b.String()
}

func handleHeading(heading *ast.Heading, markdown []byte, currentIndex *int64, requests *[]*docs.Request) {
	textVal := extractText(heading, markdown)
	*requests = append(*requests, &docs.Request{
		InsertText: &docs.InsertTextRequest{
			Text: textVal + "\n",
			Location: &docs.Location{
				Index: *currentIndex,
			},
		},
	})
	*requests = append(*requests, &docs.Request{
		UpdateParagraphStyle: &docs.UpdateParagraphStyleRequest{
			Range: &docs.Range{
				StartIndex: *currentIndex,
				EndIndex:   *currentIndex + int64(len(textVal)),
			},
			ParagraphStyle: &docs.ParagraphStyle{
				NamedStyleType: fmt.Sprintf("HEADING_%d", heading.Level),
			},
			Fields: "namedStyleType",
		},
	})
	*currentIndex += int64(len(textVal)) + 1
}

func handleParagraph(paragraph *ast.Paragraph, markdown []byte, currentIndex *int64, requests *[]*docs.Request) {
	for c := paragraph.FirstChild(); c != nil; c = c.NextSibling() {
		switch c.Kind() {
		case ast.KindText:
			textVal := string(c.(*ast.Text).Segment.Value(markdown))
			*requests = append(*requests, &docs.Request{
				InsertText: &docs.InsertTextRequest{
					Text: textVal,
					Location: &docs.Location{
						Index: *currentIndex,
					},
				},
			})
			*currentIndex += int64(len(textVal))
		case ast.KindEmphasis:
			emphasis := c.(*ast.Emphasis)
			textVal := extractText(emphasis, markdown)
			start := *currentIndex
			*requests = append(*requests, &docs.Request{
				InsertText: &docs.InsertTextRequest{
					Text: textVal,
					Location: &docs.Location{
						Index: *currentIndex,
					},
				},
			})
			*currentIndex += int64(len(textVal))
			var textStyle *docs.TextStyle
			if emphasis.Level == 1 {
				textStyle = &docs.TextStyle{Italic: true}
			} else {
				textStyle = &docs.TextStyle{Bold: true}
			}
			*requests = append(*requests, &docs.Request{
				UpdateTextStyle: &docs.UpdateTextStyleRequest{
					Range: &docs.Range{
						StartIndex: start,
						EndIndex:   *currentIndex,
					},
					TextStyle: textStyle,
					Fields:    "*",
				},
			})
		case ast.KindLink:
			link := c.(*ast.Link)
			textVal := extractText(link, markdown)
			start := *currentIndex
			*requests = append(*requests, &docs.Request{
				InsertText: &docs.InsertTextRequest{
					Text: textVal,
					Location: &docs.Location{
						Index: *currentIndex,
					},
				},
			})
			*currentIndex += int64(len(textVal))
			*requests = append(*requests, &docs.Request{
				UpdateTextStyle: &docs.UpdateTextStyleRequest{
					Range: &docs.Range{
						StartIndex: start,
						EndIndex:   *currentIndex,
					},
					TextStyle: &docs.TextStyle{
						Link: &docs.Link{
							Url: string(link.Destination),
						},
					},
					Fields: "link",
				},
			})
		}
	}
	*requests = append(*requests, &docs.Request{
		InsertText: &docs.InsertTextRequest{
			Text: "\n",
			Location: &docs.Location{
				Index: *currentIndex,
			},
		},
	})
	*currentIndex++
}

func handleList(list *ast.List, markdown []byte, currentIndex *int64, requests *[]*docs.Request) {
	bulletPreset := "BULLET_DISC_CIRCLE_SQUARE"
	if list.IsOrdered() {
		bulletPreset = "NUMBERED_DECIMAL_ALPHA_ROMAN"
	}
	for item := list.FirstChild(); item != nil; item = item.NextSibling() {
		var itemText strings.Builder
		var textRuns []*docs.Request
		var totalLen int64
		for c := item.FirstChild(); c != nil; c = c.NextSibling() {
			if c.Kind() == ast.KindTextBlock {
				for c2 := c.FirstChild(); c2 != nil; c2 = c2.NextSibling() {
					switch c2.Kind() {
					case ast.KindText:
						textVal := string(c2.(*ast.Text).Segment.Value(markdown))
						itemText.WriteString(textVal)
						totalLen += int64(len(textVal))
					case ast.KindEmphasis:
						emphasis := c2.(*ast.Emphasis)
						textVal := extractText(emphasis, markdown)
						start := *currentIndex + totalLen
						itemText.WriteString(textVal)
						totalLen += int64(len(textVal))
						var textStyle *docs.TextStyle
						if emphasis.Level == 1 {
							textStyle = &docs.TextStyle{Italic: true}
						} else {
							textStyle = &docs.TextStyle{Bold: true}
						}
						textRuns = append(textRuns, &docs.Request{
							UpdateTextStyle: &docs.UpdateTextStyleRequest{
								Range: &docs.Range{
									StartIndex: start,
									EndIndex:   *currentIndex + totalLen,
								},
								TextStyle: textStyle,
								Fields:    "*",
							},
						})
					}
				}
			}
			textVal := itemText.String()
			*requests = append(*requests, &docs.Request{
				InsertText: &docs.InsertTextRequest{
					Text: textVal + "\n",
					Location: &docs.Location{
						Index: *currentIndex,
					},
				},
			})
			*requests = append(*requests, &docs.Request{
				CreateParagraphBullets: &docs.CreateParagraphBulletsRequest{
					Range: &docs.Range{
						StartIndex: *currentIndex,
						EndIndex:   *currentIndex + int64(len(textVal)),
					},
					BulletPreset: bulletPreset,
				},
			})
			*requests = append(*requests, textRuns...)
			*currentIndex += int64(len(textVal)) + 1
		}
	}
}

// MarkdownToDocsRequests translates a Markdown string into a slice of Google Docs API requests.
func MarkdownToDocsRequests(markdown string) ([]*docs.Request, error) {
	parser := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
		),
	).Parser()
	source := []byte(markdown)
	root := parser.Parse(text.NewReader(source))

	var requests []*docs.Request
	var currentIndex int64 = 1

	for n := root.FirstChild(); n != nil; n = n.NextSibling() {
		switch n.Kind() {
		case ast.KindHeading:
			handleHeading(n.(*ast.Heading), source, &currentIndex, &requests)
		case ast.KindParagraph:
			handleParagraph(n.(*ast.Paragraph), source, &currentIndex, &requests)
		case ast.KindList:
			handleList(n.(*ast.List), source, &currentIndex, &requests)
		}
	}

	return requests, nil
}
