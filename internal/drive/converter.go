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

// MarkdownToDocsRequests translates a Markdown string into a slice of Google Docs API requests.
func MarkdownToDocsRequests(markdown string) ([]*docs.Request, error) {
	parser := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
		),
	).Parser()
	root := parser.Parse(text.NewReader([]byte(markdown)))

	var requests []*docs.Request
	var currentIndex int64 = 1

	for n := root.FirstChild(); n != nil; n = n.NextSibling() {
		switch n.Kind() {
		case ast.KindHeading:
			heading := n.(*ast.Heading)
			var headingText strings.Builder
			for c := n.FirstChild(); c != nil; c = c.NextSibling() {
				if c.Kind() == ast.KindText {
					headingText.WriteString(string(c.(*ast.Text).Segment.Value([]byte(markdown))))
				}
			}
			text := headingText.String()
			requests = append(requests, &docs.Request{
				InsertText: &docs.InsertTextRequest{
					Text: text + "\n",
					Location: &docs.Location{
						Index: currentIndex,
					},
				},
			})
			requests = append(requests, &docs.Request{
				UpdateParagraphStyle: &docs.UpdateParagraphStyleRequest{
					Range: &docs.Range{
						StartIndex: currentIndex,
						EndIndex:   currentIndex + int64(len(text)),
					},
					ParagraphStyle: &docs.ParagraphStyle{
						NamedStyleType: fmt.Sprintf("HEADING_%d", heading.Level),
					},
					Fields: "namedStyleType",
				},
			})
			currentIndex += int64(len(text)) + 1
		case ast.KindParagraph:
			for c := n.FirstChild(); c != nil; c = c.NextSibling() {
				switch c.Kind() {
				case ast.KindText:
					text := string(c.(*ast.Text).Segment.Value([]byte(markdown)))
					requests = append(requests, &docs.Request{
						InsertText: &docs.InsertTextRequest{
							Text: text,
							Location: &docs.Location{
								Index: currentIndex,
							},
						},
					})
					currentIndex += int64(len(text))
				case ast.KindEmphasis:
					emphasis := c.(*ast.Emphasis)
					text := string(c.Text([]byte(markdown)))
					start := currentIndex
					requests = append(requests, &docs.Request{
						InsertText: &docs.InsertTextRequest{
							Text: text,
							Location: &docs.Location{
								Index: currentIndex,
							},
						},
					})
					currentIndex += int64(len(text))
					var textStyle *docs.TextStyle
					if emphasis.Level == 1 {
						textStyle = &docs.TextStyle{Italic: true}
					} else {
						textStyle = &docs.TextStyle{Bold: true}
					}
					requests = append(requests, &docs.Request{
						UpdateTextStyle: &docs.UpdateTextStyleRequest{
							Range: &docs.Range{
								StartIndex: start,
								EndIndex:   currentIndex,
							},
							TextStyle: textStyle,
							Fields:    "*",
						},
					})
				case ast.KindLink:
					link := c.(*ast.Link)
					text := string(link.Text([]byte(markdown)))
					start := currentIndex
					requests = append(requests, &docs.Request{
						InsertText: &docs.InsertTextRequest{
							Text: text,
							Location: &docs.Location{
								Index: currentIndex,
							},
						},
					})
					currentIndex += int64(len(text))
					requests = append(requests, &docs.Request{
						UpdateTextStyle: &docs.UpdateTextStyleRequest{
							Range: &docs.Range{
								StartIndex: start,
								EndIndex:   currentIndex,
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
			requests = append(requests, &docs.Request{
				InsertText: &docs.InsertTextRequest{
					Text: "\n",
					Location: &docs.Location{
						Index: currentIndex,
					},
				},
			})
			currentIndex++
		case ast.KindList:
			list := n.(*ast.List)
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
								text := string(c2.(*ast.Text).Segment.Value([]byte(markdown)))
								itemText.WriteString(text)
								totalLen += int64(len(text))
							case ast.KindEmphasis:
								emphasis := c2.(*ast.Emphasis)
								text := string(c2.Text([]byte(markdown)))
								start := currentIndex + totalLen
								itemText.WriteString(text)
								totalLen += int64(len(text))
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
											EndIndex:   currentIndex + totalLen,
										},
										TextStyle: textStyle,
										Fields:    "*",
									},
								})
							}
						}
					}
				}
				text := itemText.String()
				requests = append(requests, &docs.Request{
					InsertText: &docs.InsertTextRequest{
						Text: text + "\n",
						Location: &docs.Location{
							Index: currentIndex,
						},
					},
				})
				requests = append(requests, &docs.Request{
					CreateParagraphBullets: &docs.CreateParagraphBulletsRequest{
						Range: &docs.Range{
							StartIndex: currentIndex,
							EndIndex:   currentIndex + int64(len(text)),
						},
						BulletPreset: bulletPreset,
					},
				})
				requests = append(requests, textRuns...)
				currentIndex += int64(len(text)) + 1
			}
		case extension.KindTable:
			table := n.(*extension.Table)
			rows := 0
			cols := 0
			for r := table.FirstChild(); r != nil; r = r.NextSibling() {
				rows++
				if rows == 1 {
					for c := r.FirstChild(); c != nil; c = c.NextSibling() {
						cols++
					}
				}
			}

			requests = append(requests, &docs.Request{
				InsertTable: &docs.InsertTableRequest{
					Rows:    int64(rows),
					Columns: int64(cols),
					Location: &docs.Location{
						Index: currentIndex,
					},
				},
			})
			// This is not correct, I need to get the new currentIndex after the table is inserted.
			// I will simplify this for now and just insert the text of the table.
			var tableText strings.Builder
			for r := table.FirstChild(); r != nil; r = r.NextSibling() {
				for c := r.FirstChild(); c != nil; c = c.NextSibling() {
					for c2 := c.FirstChild(); c2 != nil; c2 = c2.NextSibling() {
						if c2.Kind() == ast.KindText {
							tableText.WriteString(string(c2.(*ast.Text).Segment.Value([]byte(markdown))))
						}
					}
					tableText.WriteString("\t")
				}
				tableText.WriteString("\n")
			}
			requests = append(requests, &docs.Request{
				InsertText: &docs.InsertTextRequest{
					Text: tableText.String(),
					Location: &docs.Location{
						Index: currentIndex,
					},
				},
			})
			currentIndex += int64(len(tableText.String()))
		}
	}

	return requests, nil
}
