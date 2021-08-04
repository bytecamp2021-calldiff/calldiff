package graph

import (
	"calldiff/common"
	"fmt"
	"go/token"
	"io"
	"io/ioutil"
	"os"
	"regexp"
	"unicode"

	"golang.org/x/tools/go/callgraph"
	"golang.org/x/tools/go/callgraph/rta"
	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

const Usage = `callgraph: display the call graph of a Go program.

Usage:

  callgraph [-algo=static|cha|rta|pta] [-test] [-format=...] package...

Flags:

-algo      Specifies the call-graph construction algorithm, one of:

            static      static calls only (unsound)
            cha         Difference Hierarchy Analysis
            rta         Rapid Type Analysis
            pta         inclusion-based Points-To Analysis

           The algorithms are ordered by increasing precision in their
           treatment of dynamic calls (and thus also computational cost).
           RTA and PTA require a whole program (main or test), and
           include only functions reachable from main.

-test      Include the package's tests in the analysis.

-format    Specifies the format in which each call graph edge is displayed.
           One of:

            digraph     output suitable for input to
                        golang.org/x/tools/cmd/digraph.
            graphviz    output in AT&T GraphViz (.dot) format.

           All other values are interpreted using text/template syntax.
           The default value is:

            {{.Caller}}\t--{{.Dynamic}}-{{.Line}}:{{.Column}}-->\t{{.Callee}}

           The structure passed to the template is (effectively):

                   type Edge struct {
                           Caller      *ssa.Function // calling function
                           Callee      *ssa.Function // called function

                           // Call site:
                           Filename    string // containing file
                           Offset      int    // offset within file of '('
                           Line        int    // line number
                           Column      int    // column number of call
                           Dynamic     string // "static" or "dynamic"
                           Description string // e.g. "static method call"
                   }

           Caller and Callee are *ssa.Function values, which print as
           "(*sync/atomic.Mutex).Lock", but other attributes may be
           derived from them, e.g. Caller.Pkg.Pkg.Path yields the
           import dir of the enclosing package.  Consult the go/ssa
           API documentation for details.

Examples:

  Show the call graph of the trivial web server application:

    callgraph -format digraph $GOROOT/src/net/http/triv.go

  Same, but show only the packages of each function:

    callgraph -format '{{.Caller.Pkg.Pkg.Path}} -> {{.Callee.Pkg.Pkg.Path}}' \
      $GOROOT/src/net/http/triv.go | sort | uniq

  Show functions that make dynamic calls into the 'fmt' test package,
  using the pointer analysis algorithm:

    callgraph -format='{{.Caller}} -{{.Dynamic}}-> {{.Callee}}' -test -algo=pta fmt |
      sed -ne 's/-dynamic-/--/p' |
      sed -ne 's/-->.*fmt_test.*$//p' | sort | uniq

  Show all functions directly called by the callgraph tool's main function:

    callgraph -format=digraph golang.org/x/tools/cmd/callgraph |
      digraph succs golang.org/x/tools/cmd/callgraph.main
`

var stdout io.Writer = os.Stdout

func GetCallgraph(diffOptions *common.DiffOptions, i int) {
	defer diffOptions.Wg.Done()

	r := clone(diffOptions.Url, diffOptions.Dir)

	commitHash := getCommitHash(r, diffOptions.Commit[i])

	path, err := ioutil.TempDir(diffOptions.Dir, "")
	if err != nil {
		common.Error("%s", err)
		return
	}

	diffOptions.Path[i] = path

	defer os.RemoveAll(diffOptions.Path[i])

	if err := outputCommitFiles(commitHash, diffOptions.Path[i]); err != nil {
		common.Error("%s", err)
		return
	}

	if err := doCallgraph(diffOptions, i); err != nil {
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

func doCallgraph(diffOptions *common.DiffOptions, i int) error {
	cfg := &packages.Config{
		Mode:  packages.LoadSyntax,
		Tests: diffOptions.Test,
		Dir:   diffOptions.Path[i],
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
				fmt.Println("name:", ssaFunc.Pkg.Pkg.Name(), ssaFunc.Name())
			}
		}
	}
	rtares := rta.Analyze(roots, true)
	diffOptions.Callgraph[i] = rtares.CallGraph

	// NB: RTA gives us Reachable and RuntimeTypes too.

	diffOptions.Callgraph[i].DeleteSyntheticNodes()
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
