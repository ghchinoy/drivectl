package drive

import (
	"fmt"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
	"google.golang.org/api/docs/v1"
)

// MarkdownToDocsRequests translates a Markdown string into a slice of Google Docs API requests.
func MarkdownToDocsRequests(markdown string) ([]*docs.Request, error) {
	parser := goldmark.DefaultParser()
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
			for item := list.FirstChild(); item != nil; item = item.NextSibling() {
				var itemText strings.Builder
				for c := item.FirstChild(); c != nil; c = c.NextSibling() {
					if c.Kind() == ast.KindTextBlock {
						for c2 := c.FirstChild(); c2 != nil; c2 = c2.NextSibling() {
							if c2.Kind() == ast.KindText {
								itemText.WriteString(string(c2.(*ast.Text).Segment.Value([]byte(markdown))))
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
						BulletPreset: "BULLET_DISC_CIRCLE_SQUARE",
					},
				})
				currentIndex += int64(len(text)) + 1
			}
		}
	}

	return requests, nil
}
