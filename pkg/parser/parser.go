package parser

import (
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Azure/gocover/pkg/annotation"
	"github.com/Azure/gocover/pkg/gittool"
	"github.com/sirupsen/logrus"
	"golang.org/x/tools/cover"
)

type packagesCache map[string]*build.Package

func NewParser(
	coverProfileFiles []string,
	logger logrus.FieldLogger,
) *Parser {
	return &Parser{
		coverProfileFiles: coverProfileFiles,
		coverProfiles:     make([]*cover.Profile, 0),
		packages:          make(map[string]*Package),
		packagesCache:     make(packagesCache),
		logger:            logger.WithField("source", "Parser"),
	}
}

// Parser wrapper for parsing
type Parser struct {
	packages          map[string]*Package
	packagesCache     packagesCache
	coverProfileFiles []string
	coverProfiles     []*cover.Profile

	logger logrus.FieldLogger
}

// Parse parses cover profiles into statements, and modify their state based on git changes.
func (parser *Parser) Parse(changes []*gittool.Change) (Packages, error) {
	if err := parser.filterCoverProfiles(changes); err != nil {
		parser.logger.WithError(err).Error("filter cover profiles")
		return nil, err
	}
	if err := parser.buildPackageCache(); err != nil {
		parser.logger.WithError(err).Error("build package cache")
		return nil, err
	}

	var result Packages

	for _, p := range parser.coverProfiles {
		if err := parser.convertProfile(p, findChange(p, changes)); err != nil {
			parser.logger.WithError(err).Error("covert cover profile")
			return nil, err
		}
	}

	for _, pkg := range parser.packages {
		result.AddPackage(pkg)
	}

	return result, nil
}

// filterCoverProfiles filters cover profiles based on git changes.
// If changes is nil, all cover profiles will be kept.
// If changes is not nil, only cover profiles that are changed will be kept.
func (parser *Parser) filterCoverProfiles(changes []*gittool.Change) error {

	for _, coverProfile := range parser.coverProfileFiles {
		profiles, err := cover.ParseProfiles(coverProfile)
		if err != nil {
			return err
		}

		if changes == nil {
			parser.coverProfiles = append(parser.coverProfiles, profiles...)
			continue
		}

		for _, p := range profiles {
			if findChange(p, changes) != nil {
				parser.coverProfiles = append(parser.coverProfiles, p)
			}
		}
	}

	return nil
}

// buildPackageCache builds a cache of packages for all cover profiles.
func (parser *Parser) buildPackageCache() error {

	for _, profile := range parser.coverProfiles {
		dir, _ := filepath.Split(profile.FileName)
		if dir != "" {
			dir = strings.TrimSuffix(dir, "/")
		}
		_, ok := parser.packagesCache[dir]
		if !ok {
			pkg, err := build.Import(dir, ".", build.FindOnly)
			if err != nil {
				return err
			}
			parser.packagesCache[dir] = pkg
			parser.packages[pkg.ImportPath] = &Package{Name: pkg.ImportPath}
		}
	}

	return nil
}

// wrapper for Statement
type statement struct {
	*Statement
	*StmtExtent
}

func (parser *Parser) convertProfile(p *cover.Profile, change *gittool.Change) error {
	file, pkgpath, err := findFile(parser.packagesCache, p.FileName)
	if err != nil {
		parser.logger.WithError(err).Error("find file")
		return err
	}
	parser.logger.Debugf("[file=%s, pkgPath=%s]", file, pkgpath)

	pkg := parser.packages[pkgpath]
	if pkg == nil {
		pkg = &Package{Name: pkgpath}
		parser.packages[pkgpath] = pkg
	}

	ignoreProfile, err := annotation.ParseIgnoreProfiles(file, p)
	if err != nil {
		parser.logger.WithError(err).Error("parse ignore profile")
		return err
	}
	if ignoreProfile != nil {
		if ignoreProfile.Type == annotation.FILE_IGNORE {
			pkg.IgnoreProfiles = append(pkg.IgnoreProfiles, ignoreProfile)
		} else {
			if len(ignoreProfile.IgnoreBlocks) != 0 {
				pkg.IgnoreProfiles = append(pkg.IgnoreProfiles, ignoreProfile)
			}
		}
	}

	// Find function and statement extents; create corresponding
	// Functions and Statements, and keep a separate
	// slice of Statements so we can match them with profile
	// blocks.
	extents, err := findFuncs(file)
	if err != nil {
		parser.logger.WithError(err).Error("find Functions")
		return err
	}
	var stmts []*statement
	for _, fe := range extents {
		f := &Function{
			Name:      fe.name,
			File:      file,
			Start:     fe.startOffset,
			End:       fe.endOffset,
			StartLine: fe.startLine,
			EndLine:   fe.endLine,
		}
		for _, se := range fe.stmts {
			s := &statement{
				Statement: &Statement{
					StartLine: se.startLine,
					EndLine:   se.endLine,
					Start:     se.startOffset,
					End:       se.endOffset,
					Mode:      Keep,
					State:     Original,
				},
				StmtExtent: se,
			}
			f.Statements = append(f.Statements, s.Statement)
			stmts = append(stmts, s)
		}
		pkg.Functions = append(pkg.Functions, f)
	}
	// For each profile block in the file, find the statement(s) it
	// covers and increment the Reached field(s).
	blocks := p.Blocks
	for _, s := range stmts {

		for i, b := range blocks {
			if b.StartLine > s.endLine || (b.StartLine == s.endLine && b.StartCol >= s.endCol) {
				// Past the end of the statement
				blocks = blocks[i:]
				break
			}
			if b.EndLine < s.startLine || (b.EndLine == s.startLine && b.EndCol <= s.startCol) {
				// Before the beginning of the statement
				continue
			}

			s.Reached += int64(b.Count)

			if ignoreProfile != nil {
				if ignoreProfile.Type == annotation.FILE_IGNORE {
					s.Mode = Ignore
					parser.logger.Debugf("hit file ignore on [%s], ignore statement at line %d", file, s.startLine)
				} else {
					// ignore those statements when block annotated with block ignore annotation
					if _, ok := ignoreProfile.IgnoreBlocks[b]; ok {
						s.Mode = Ignore
						parser.logger.Debugf("hit block ignore on [%s], ignore statement at line %d", file, s.startLine)
					}
				}
			}

			break
		}
	}

	parser.setStatementsState(change, stmts)
	return nil
}

// findFile finds the location of the named file in GOROOT, GOPATH etc.
func findFile(packages packagesCache, file string) (filename, pkgpath string, err error) {
	dir, file := filepath.Split(file)
	if dir != "" {
		dir = strings.TrimSuffix(dir, "/")
	}
	pkg, ok := packages[dir]
	if !ok {
		return "", "", fmt.Errorf("no package found for %s", file)
	}

	return filepath.Join(pkg.Dir, file), pkg.ImportPath, nil
}

// findFuncs parses the file and returns a slice of FuncExtent descriptors.
func findFuncs(name string) ([]*FuncExtent, error) {
	fset := token.NewFileSet()
	parsedFile, err := parser.ParseFile(fset, name, nil, 0)
	if err != nil {
		return nil, err
	}
	visitor := &FuncVisitor{fset: fset}
	ast.Walk(visitor, parsedFile)
	return visitor.funcs, nil
}

type extent struct {
	startOffset int
	startLine   int
	startCol    int
	endOffset   int
	endLine     int
	endCol      int
}

// FuncExtent describes a function's extent in the source by file and position.
type FuncExtent struct {
	extent
	name  string
	stmts []*StmtExtent
}

// StmtExtent describes a statements's extent in the source by file and position.
type StmtExtent extent

// FuncVisitor implements the visitor that builds the function position list for a file.
type FuncVisitor struct {
	fset  *token.FileSet
	funcs []*FuncExtent
}

func functionName(f *ast.FuncDecl) string {
	name := f.Name.Name
	if f.Recv == nil {
		return name
	} else {
		// Function name is prepended with "T." if there is a receiver, where
		// T is the type of the receiver, dereferenced if it is a pointer.
		return exprName(f.Recv.List[0].Type) + "." + name
	}
}

func exprName(x ast.Expr) string {
	switch y := x.(type) {
	case *ast.StarExpr:
		return exprName(y.X)
	case *ast.IndexExpr:
		return fmt.Sprintf("%s[%s]", exprName(y.X), exprName(y.Index))
	case *ast.Ident:
		return y.Name
	default:
		return ""
	}
}

// Visit implements the ast.Visitor interface.
func (v *FuncVisitor) Visit(node ast.Node) ast.Visitor {
	var body *ast.BlockStmt
	var name string
	switch n := node.(type) {
	case *ast.FuncLit:
		body = n.Body
	case *ast.FuncDecl:
		body = n.Body
		name = functionName(n)
	}
	if body != nil {
		start := v.fset.Position(node.Pos())
		end := v.fset.Position(node.End())
		if name == "" {
			name = fmt.Sprintf("@%d:%d", start.Line, start.Column)
		}
		fe := &FuncExtent{
			name: name,
			extent: extent{
				startOffset: start.Offset,
				startLine:   start.Line,
				startCol:    start.Column,
				endOffset:   end.Offset,
				endLine:     end.Line,
				endCol:      end.Column,
			},
		}
		v.funcs = append(v.funcs, fe)
		sv := StmtVisitor{fset: v.fset, function: fe}
		sv.VisitStmt(body)
	}
	return v
}

type StmtVisitor struct {
	fset     *token.FileSet
	function *FuncExtent
}

func (v *StmtVisitor) VisitStmt(s ast.Stmt) {
	var statements *[]ast.Stmt
	switch s := s.(type) {
	case *ast.BlockStmt:
		statements = &s.List
	case *ast.CaseClause:
		statements = &s.Body
	case *ast.CommClause:
		statements = &s.Body
	case *ast.ForStmt:
		if s.Init != nil {
			v.VisitStmt(s.Init)
		}
		if s.Post != nil {
			v.VisitStmt(s.Post)
		}
		v.VisitStmt(s.Body)
	case *ast.IfStmt:
		if s.Init != nil {
			v.VisitStmt(s.Init)
		}
		v.VisitStmt(s.Body)
		if s.Else != nil {
			// Code copied from go.tools/cmd/cover, to deal with "if x {} else if y {}"
			const backupToElse = token.Pos(len("else ")) // The AST doesn't remember the else location. We can make an accurate guess.
			switch stmt := s.Else.(type) {
			case *ast.IfStmt:
				block := &ast.BlockStmt{
					Lbrace: stmt.If - backupToElse, // So the covered part looks like it starts at the "else".
					List:   []ast.Stmt{stmt},
					Rbrace: stmt.End(),
				}
				s.Else = block
			case *ast.BlockStmt:
				stmt.Lbrace -= backupToElse // So the block looks like it starts at the "else".
			default:
				panic("unexpected node type in if")
			}
			v.VisitStmt(s.Else)
		}
	case *ast.LabeledStmt:
		v.VisitStmt(s.Stmt)
	case *ast.RangeStmt:
		v.VisitStmt(s.Body)
	case *ast.SelectStmt:
		v.VisitStmt(s.Body)
	case *ast.SwitchStmt:
		if s.Init != nil {
			v.VisitStmt(s.Init)
		}
		v.VisitStmt(s.Body)
	case *ast.TypeSwitchStmt:
		if s.Init != nil {
			v.VisitStmt(s.Init)
		}
		v.VisitStmt(s.Assign)
		v.VisitStmt(s.Body)
	}
	if statements == nil {
		return
	}
	for i := 0; i < len(*statements); i++ {
		s := (*statements)[i]
		switch s.(type) {
		case *ast.CaseClause, *ast.CommClause, *ast.BlockStmt:
			break
		default:
			start, end := v.fset.Position(s.Pos()), v.fset.Position(s.End())
			se := &StmtExtent{
				startOffset: start.Offset,
				startLine:   start.Line,
				startCol:    start.Column,
				endOffset:   end.Offset,
				endLine:     end.Line,
				endCol:      end.Column,
			}
			v.function.stmts = append(v.function.stmts, se)
		}
		v.VisitStmt(s)
	}
}

// setStatementsState sets statements' State field according to the file change.
// If change is nil, means no change is made, it usually runs in full coverage mode
// If change is not nil, loop over each changed lines and find its statement and set the statement to Changed
func (parser *Parser) setStatementsState(change *gittool.Change, statements []*statement) {
	if change == nil {
		return
	}
	if len(statements) == 0 {
		return
	}

	sort.Sort(statementByStart(statements))

	parser.logger.Debugf("processing changed file: %s", change.FileName)
	for _, s := range change.Sections {
		for lineNum := s.StartLine; lineNum <= s.EndLine; lineNum++ {
			if isCodeLine(s.Contents[lineNum-s.StartLine]) {
				parser.setStatementsStateByLineNumber(lineNum, statements)
			}
		}
	}
}

func isCodeLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	return trimmed != "" && !strings.HasPrefix(trimmed, "//")
}

// setStatementsStateByLineNumber sets statements' State field based on code line number.
// It sort statements by startline first, then try to find first statement
// that line number is greater than or equals the startline of the statement.
// There are two edge cases:
//  1. When line number is less than all the statements, `Search` function will return 0,
//     but there is no suitable statement, should return immediately.
//  2. Otherwise, `Search` function will return first statement that its startline is greater than changed line number,
//     then statement of that position minus one is the statement we want,
//     but still need to check whether the changed line is among the statement scope.
func (parser *Parser) setStatementsStateByLineNumber(changedlineNumber int, statements []*statement) {
	idx := sort.Search(len(statements), func(i int) bool {
		return statements[i].startLine > changedlineNumber
	})

	if idx == 0 { // no suitable statement
		return
	}

	idx--
	stmt := statements[idx]

	if stmt != nil && lineNumberInStatement(changedlineNumber, stmt) {
		stmt.State = Changed
		parser.logger.Debugf(
			"for changed line number %d, set statement [%d:%d] to %s",
			changedlineNumber, statements[idx].startLine, statements[idx].endLine, statements[idx].State,
		)
	}
}

func lineNumberInStatement(lineNumber int, stmt *statement) bool {
	return stmt.startLine <= lineNumber && stmt.endLine >= lineNumber
}

type statementByStart []*statement

func (s statementByStart) Len() int      { return len(s) }
func (s statementByStart) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s statementByStart) Less(i, j int) bool {
	si, sj := s[i], s[j]
	return si.startLine < sj.startLine || si.startLine == sj.startLine && si.startCol < sj.startCol
}

// findChange find the expected change by file name.
func findChange(profile *cover.Profile, changes []*gittool.Change) *gittool.Change {
	for _, change := range changes {
		if InFolder(profile.FileName, change.FileName) {
			return change
		}
	}
	return nil
}

// InFolder check whether specified filepath is a part of parent path.
func InFolder(parentDir, filepath string) bool {
	return strings.HasSuffix(parentDir, filepath)
}
