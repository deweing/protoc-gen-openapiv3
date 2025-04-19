package errcode

import (
	"errors"
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"go/types"
	"golang.org/x/tools/go/loader"
	"os"
	"sort"
	"strings"
)

const (
	ErrCodePrefix = "ErrCode"
)

var (
	DirNotExist = errors.New("error dir not exist")
	ImportError = errors.New("import error")
	LoadError   = errors.New("load error")
)

type ErrCode struct {
	Code    string
	Name    string
	Message string
}

func (e ErrCode) String() string {
	return fmt.Sprintf("%s : %s %s", e.Code, e.Name, e.Message)
}

type ErrCodes []*ErrCode

func (e ErrCodes) String() string {
	if len(e) == 0 {
		return "200正常,其他错误"
	}

	sort.Slice(e, func(i, j int) bool {
		return e[i].Code < e[j].Code
	})
	str := "错误代码:\n"
	for _, c := range e {
		str += " - " + c.String() + "\n"
	}

	return str
}

func isDir(path string) (bool, error) {
	s, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		} else {
			return false, err
		}
	}
	return s.IsDir(), nil
}

func checkDir(path string) error {
	b, err := isDir(path)
	if err != nil {
		return err
	}

	if !b {
		return DirNotExist
	}
	return nil
}

func LoadErrCodes(dir string) (string, error) {
	if err := checkDir(dir); err != nil {
		return "", err
	}

	codes, err := loadErrCodes(dir)
	if err != nil {
		return "", err
	}

	return codes.String(), nil
}

func loadErrCodes(dir string) (*ErrCodes, error) {
	ldr := loader.Config{
		AllowErrors: true,
		ParserMode:  parser.ParseComments,
		Cwd:         dir,
	}

	d, err := build.ImportDir(dir, build.FindOnly)
	if err != nil {
		return nil, ImportError
	}

	ldr.Import(d.ImportPath)
	p, err := ldr.Load()
	if err != nil {
		return nil, LoadError
	}

	var errCodes ErrCodes
	for pkg, pkgInfo := range p.AllPackages {
		if pkg.Path() != d.ImportPath {
			continue
		}

		for ident, obj := range pkgInfo.Defs {
			if constObj, ok := obj.(*types.Const); ok {
				key := constObj.Name()
				if !strings.HasPrefix(key, ErrCodePrefix) {
					continue
				}

				doc := CommentsOf(p.Fset, ident, pkgInfo.Files...)
				_, message := ParseComment(doc, key)

				errCodes = append(errCodes, &ErrCode{
					Code:    constObj.Val().ExactString(),
					Name:    key,
					Message: message,
				})
			}
		}

		break
	}

	return &errCodes, nil
}

func CommentsOf(fileSet *token.FileSet, targetNode ast.Node, files ...*ast.File) string {
	file := FileOf(targetNode, files...)
	if file == nil {
		return ""
	}
	commentScanner := NewCommentScanner(fileSet, file)
	doc := commentScanner.CommentsOf(targetNode)
	if doc != "" {
		return doc
	}
	return doc
}

func FileOf(targetNode ast.Node, files ...*ast.File) *ast.File {
	for _, file := range files {
		if file.Pos() <= targetNode.Pos() && file.End() > targetNode.Pos() {
			return file
		}
	}
	return nil
}

func NewCommentScanner(fileSet *token.FileSet, file *ast.File) *CommentScanner {
	commentMap := ast.NewCommentMap(fileSet, file, file.Comments)

	return &CommentScanner{
		file:       file,
		CommentMap: commentMap,
	}
}

type CommentScanner struct {
	file       *ast.File
	CommentMap ast.CommentMap
}

func (scanner *CommentScanner) CommentsOf(targetNode ast.Node) string {
	commentGroupList := scanner.CommentGroupListOf(targetNode)
	return StringifyCommentGroup(commentGroupList...)
}

func (scanner *CommentScanner) CommentGroupListOf(targetNode ast.Node) (commentGroupList []*ast.CommentGroup) {
	if targetNode == nil {
		return
	}

	switch targetNode.(type) {
	case *ast.File, *ast.Field, ast.Stmt, ast.Decl:
		if comments, ok := scanner.CommentMap[targetNode]; ok {
			commentGroupList = comments
		}
	case ast.Spec:
		// Spec should merge with comments of its parent gen decl when empty
		if comments, ok := scanner.CommentMap[targetNode]; ok {
			commentGroupList = append(commentGroupList, comments...)
		}

		if len(commentGroupList) == 0 {
			for node, comments := range scanner.CommentMap {
				if genDecl, ok := node.(*ast.GenDecl); ok {
					for _, spec := range genDecl.Specs {
						if targetNode == spec {
							commentGroupList = append(commentGroupList, comments...)
						}
					}
				}
			}
		}
	default:
		// find nearest parent node which have comments
		{
			var deltaPos token.Pos
			var parentNode ast.Node

			deltaPos = -1

			ast.Inspect(scanner.file, func(node ast.Node) bool {
				switch node.(type) {
				case *ast.Field, ast.Decl, ast.Spec, ast.Stmt:
					if targetNode.Pos() >= node.Pos() && targetNode.End() <= node.End() {
						nextDelta := targetNode.Pos() - node.Pos()
						if deltaPos == -1 || (nextDelta <= deltaPos) {
							deltaPos = nextDelta
							parentNode = node
						}
					}
				}
				return true
			})

			if parentNode != nil {
				commentGroupList = scanner.CommentGroupListOf(parentNode)
			}
		}
	}

	sort.Sort(ByCommentPos(commentGroupList))
	return
}

type ByCommentPos []*ast.CommentGroup

func (a ByCommentPos) Len() int {
	return len(a)
}

func (a ByCommentPos) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a ByCommentPos) Less(i, j int) bool {
	return a[i].Pos() < a[j].Pos()
}

func StringifyCommentGroup(commentGroupList ...*ast.CommentGroup) (comments string) {
	if len(commentGroupList) == 0 {
		return ""
	}
	for _, commentGroup := range commentGroupList {
		for _, line := range strings.Split(commentGroup.Text(), "\n") {
			if strings.HasPrefix(line, "go:generate") {
				continue
			}
			comments = comments + "\n" + line
		}
	}
	return strings.TrimSpace(comments)
}

func ParseComment(text string, codeName string) (comment string, message string) {
	lines := strings.Split(text, "\n")
	comment = strings.TrimSpace(lines[0])
	for i := 1; i < len(lines); i++ {
		str := strings.TrimSpace(lines[i])
		if strings.HasPrefix(str, "@message") {
			message = strings.TrimPrefix(str, "@message")
			break
		}
	}

	comment = strings.TrimPrefix(comment, codeName)
	if message == "" {
		message = strings.TrimPrefix(comment, "@message")
	}

	return strings.TrimSpace(comment), strings.TrimSpace(message)
}
