package graph

import (
	"calldiff/common"
	"fmt"
	"go/token"
	"io/ioutil"
	"os"
	"regexp"
	"sync"
	"unicode"

	"golang.org/x/tools/go/callgraph"
	"golang.org/x/tools/go/callgraph/rta"
	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

func GetCallgraph(diffOptions *common.DiffOptions, graphOptions *common.GraphOptions, wg *sync.WaitGroup) {
	defer wg.Done()

	r := clone(diffOptions.Url, diffOptions.Dir)

	commitHash := getCommitHash(r, graphOptions.Commit)

	path, err := ioutil.TempDir(diffOptions.Dir, "")
	if err != nil {
		common.Error("%s", err)
		return
	}

	graphOptions.TempPath = path

	defer os.RemoveAll(graphOptions.TempPath)

	if err := outputCommitFiles(commitHash, graphOptions.TempPath); err != nil {
		common.Error("%s", err)
		return
	}

	if err := doCallgraph(diffOptions, graphOptions); err != nil {
		common.Error("%s", err)
		return
	}
}

func isPublic(f *ssa.Function) bool {
	match, _ := regexp.MatchString("(\\([1-9][0-9]*\\))init", f.Name())
	return unicode.IsUpper(rune(f.Name()[0])) || f.Name() == "main" || match
}

func isAutoInit(f *ssa.Function) bool {
	return f.Name() == "init"
}

func getAllFunctions(s *ssa.Package) *[]*ssa.Function {
	var result []*ssa.Function
	for _, m := range s.Members {
		if f, ok := m.(*ssa.Function); ok {
			result = append(result, f)
		}
	}
	return &result
}

func doCallgraph(diffOptions *common.DiffOptions, graphOptions *common.GraphOptions) error {
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles |
			packages.NeedImports | packages.NeedTypes | packages.NeedTypesSizes |
			packages.NeedSyntax | packages.NeedTypesInfo,
		Tests: diffOptions.Test,
		Dir:   graphOptions.TempPath,
	}
	initial, err := packages.Load(cfg, "./...")
	if err != nil {
		return err
	}
	if packages.PrintErrors(initial) > 0 {
		return fmt.Errorf("packages contain errors")
	}

	// Create and build SSA-form program representation.
	prog, pkgs := ssautil.Packages(initial, 0)
	prog.Build()

	// -- call graph construction ------------------------------------------

	mains, err := mainPackages(pkgs, diffOptions.Pkg)
	if err != nil {
		return err
	}
	var roots []*ssa.Function
	for _, main := range mains {
		funcList := getAllFunctions(main)
		for _, ssaFunc := range *funcList {
			if isAutoInit(ssaFunc) {
				continue
			}
			if diffOptions.PrintPrivate || isPublic(ssaFunc) {
				roots = append(roots, ssaFunc)
				//fmt.Println("name:", ssaFunc.Pkg.Pkg.Name(), ssaFunc.Name())
			}
		}
	}
	rtares := rta.Analyze(roots, true)
	graphOptions.CallGraph = rtares.CallGraph

	// NB: RTA gives us Reachable and RuntimeTypes too.

	graphOptions.CallGraph.DeleteSyntheticNodes()
	return nil
}

// mainPackages returns the main packages to analyze.
// Each resulting package is named "main" and has a main function.
func mainPackages(pkgs []*ssa.Package, pkg string) ([]*ssa.Package, error) {
	var mains []*ssa.Package
	for _, p := range pkgs {
		if p != nil && p.Pkg.Name() == pkg {
			mains = append(mains, p)
		}
	}
	if len(mains) == 0 {
		return nil, fmt.Errorf("no %s packages", pkg)
	}
	return mains, nil
}

type Edge struct {
	Caller *ssa.Function
	Callee *ssa.Function

	edge     *callgraph.Edge
	fset     *token.FileSet
	position token.Position // initialized lazily
}

func (e *Edge) pos() *token.Position {
	if e.position.Offset == -1 {
		e.position = e.fset.Position(e.edge.Pos()) // called lazily
	}
	return &e.position
}

func (e *Edge) Filename() string { return e.pos().Filename }
func (e *Edge) Column() int      { return e.pos().Column }
func (e *Edge) Line() int        { return e.pos().Line }
func (e *Edge) Offset() int      { return e.pos().Offset }

func (e *Edge) Dynamic() string {
	if e.edge.Site != nil && e.edge.Site.Common().StaticCallee() == nil {
		return "dynamic"
	}
	return "static"
}

func (e *Edge) Description() string { return e.edge.Description() }
