package drive

import (
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
	"google.golang.org/api/docs/v1"
)

// MarkdownToDocs translates a Markdown string into a Google Docs Document object.
func MarkdownToDocs(markdown string) (*docs.Document, error) {
	parser := goldmark.DefaultParser()
	root := parser.Parse(text.NewReader([]byte(markdown)))

	// TODO: Implement the AST to Google Docs JSON translation.
	ast.Walk(root, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering {
			// Here is where we will have a big switch statement
			// based on the node kind.
			switch n.Kind() {
			case ast.KindHeading:
				// Create a heading request
			case ast.KindParagraph:
				// Create a paragraph request
			case ast.KindList:
				// Handle lists
			// ... and so on
			}
		}
		return ast.WalkContinue, nil
	})

	// This is a placeholder.
	return &docs.Document{}, nil
}
