// Package cc implements parsing of C and C++ source files using Clang.
package cc

import (
	"fmt"
	"strings"

	multierror "github.com/hashicorp/go-multierror"
	"github.com/mewspring/go-clang/clang"
	"github.com/pkg/errors"
)

// ParseFile parses the given source file, returning the root node of the AST.
// Note, a (partial) AST is returned even when an error is encountered.
func ParseFile(srcPath string, clangArgs ...string) (*Node, error) {
	// Create index.
	idx := clang.NewIndex(0, 1)
	//defer idx.Dispose()
	// Create translation unit.
	tu := idx.ParseTranslationUnit(srcPath, clangArgs, nil, 0)
	//defer tu.Dispose()
	// Print errors.
	diagnostics := tu.Diagnostics()
	var err error
	for _, d := range diagnostics {
		err = multierror.Append(err, errors.New(d.Spelling()))
	}
	// Parse source file.
	nodeFromHash := make(map[uint32]*Node)
	cursor := tu.TranslationUnitCursor()
	loc := cursor.Location()
	file, line, col := loc.PresumedLocation()
	root := &Node{
		Body:     cursor,
		Def:      cursor.Definition(),
		Kind:     cursor.Kind().String(),
		Spelling: cursor.Spelling(),
		Loc: Location{
			File: file,
			Line: line,
			Col:  col,
		},
	}
	nodeFromHash[root.Body.HashCursor()] = root
	visit := func(cursor, parent clang.Cursor) clang.ChildVisitResult {
		if cursor.IsNull() {
			return clang.ChildVisit_Continue
		}
		parentNode, ok := nodeFromHash[parent.HashCursor()]
		if !ok {
			panic(fmt.Errorf("unable to locate node of parent cursor %v(%v)", parentNode.Kind, parentNode.Spelling))
		}
		loc := cursor.Location()
		file, line, col := loc.PresumedLocation()
		n := &Node{
			Body:     cursor,
			Def:      cursor.Definition(),
			Kind:     cursor.Kind().String(),
			Spelling: cursor.Spelling(),
			Loc: Location{
				File: file,
				Line: line,
				Col:  col,
			},
		}
		nodeFromHash[n.Body.HashCursor()] = n
		parentNode.Children = append(parentNode.Children, n)
		return clang.ChildVisit_Recurse
	}
	cursor.Visit(visit)
	return root, err
}

// Node is a node of the AST.
type Node struct {
	// Node contents.
	Body clang.Cursor
	// Definition of the entry associated with the node.
	Def clang.Cursor
	// String representation of node.
	Spelling string // cached result of Body.Spelling()
	// Node kind.
	Kind string // cached result of Body.Kind().String()
	// Source location of node.
	Loc Location // cached result of Body.Location().PersumedLocation()
	// Child nodes of the node.
	Children []*Node
}

// Location denotes a location in a source file.
type Location struct {
	// Source file.
	File string
	// Line number (1-indexed).
	Line uint32
	// Column (1-indexed).
	Col uint32
}

// NewLocation returns a new location based on the given Clang source location.
func NewLocation(loc clang.SourceLocation) Location {
	file, line, col := loc.PresumedLocation()
	return Location{
		File: file,
		Line: line,
		Col:  col,
	}
}

// String returns a string representation of the source code location.
func (loc Location) String() string {
	return fmt.Sprintf("%s:%d:%d", loc.File, loc.Line, loc.Col)
}

// PrintTree pretty-prints the given AST starting at the root node.
func PrintTree(root *Node) {
	printTree(root, 0)
}

// printTree pretty-prints the given AST node and its children with the
// corresponding indentation level.
func printTree(n *Node, indentLevel int) {
	indent := strings.Repeat("\t", indentLevel)
	fmt.Printf("%s%s\n", indent, n.Body.Kind().String())
	for _, child := range n.Children {
		printTree(child, indentLevel+1)
	}
}

// Walk walks the given AST, invoking f for each node visited.
func Walk(root *Node, f func(n *Node)) {
	f(root)
	for _, child := range root.Children {
		Walk(child, f)
	}
}
