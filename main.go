package main

import (
	"flag"
	"go/build"
	"os"
	"runtime"
	"sync"

	"golang.org/x/tools/go/buildutil"

	"github.com/bytecamp2021-calldiff/calldiff/analyze"
	"github.com/bytecamp2021-calldiff/calldiff/common"
	"github.com/bytecamp2021-calldiff/calldiff/graph"
)

func init() {
	flag.Var((*buildutil.TagsFlag)(&build.Default.BuildTags), "tags", buildutil.TagsFlagDoc)
}

func init() {
	// If $GOMAXPROCS isn't set, use the full capacity of the machine.
	// For small machines, use at least 4 threads.
	if os.Getenv("GOMAXPROCS") == "" {
		n := runtime.NumCPU()
		if n < 4 {
			n = 4
		}
		runtime.GOMAXPROCS(n)
	}
}

func main() {
	var diffOptions common.DiffOptions
	var source, target common.GraphOptions
	flag.StringVar(&diffOptions.URL, "url", "", `Git repository address`)
	flag.StringVar(&diffOptions.Dir, "dir", ".", `Repository path`)
	flag.StringVar(&source.Commit, "old", "HEAD^", `Old commit ID`)
	flag.StringVar(&target.Commit, "new", "HEAD", `New commit ID`)
	flag.BoolVar(&diffOptions.Test, "test", false, `Loads test code (*_test.go) for imported packages`)
	flag.BoolVar(&diffOptions.PrintPrivate, "private", false, `If output private function`)
	flag.BoolVar(&diffOptions.PrintUnchanged, "unchanged", false, `If output unchanged function`)
	flag.StringVar(&diffOptions.Output, "output", "json,graphviz", `Supported output types are json and graphviz`)
	flag.StringVar(&diffOptions.Pkg, "pkg", "main", `Analyse which packages`)
	flag.Parse()

	// Get commits' callgraph
	var wg sync.WaitGroup
	wg.Add(2)
	go graph.GetCallGraph(&diffOptions, &source, &wg)
	go graph.GetCallGraph(&diffOptions, &target, &wg)
	wg.Wait()

	diffGraph := analyze.GetDiff(source.CallGraph, target.CallGraph)
	diffGraph.OutputDiffGraph(&diffOptions)
}
