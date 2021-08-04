package main

import (
	"flag"
	"go/build"
	"log"
	"os"
	"runtime"
	"runtime/pprof"

	"calldiff/common"
	"calldiff/diff"
	"calldiff/graph"

	"golang.org/x/tools/go/buildutil"
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
	cpuFile, err := os.Create("cpu_profile")
	if err != nil {
		log.Fatal(err)
	}
	_ = pprof.StartCPUProfile(cpuFile) // 开始记录CPU数据
	defer pprof.StopCPUProfile()       // 停止记录

	var diffOptions common.DiffOptions
	flag.StringVar(&diffOptions.Url, "url", "", `Git repository address`)
	flag.StringVar(&diffOptions.Dir, "dir", ".", `Repository path`)
	flag.StringVar(&diffOptions.Commit[0], "old", "HEAD^", `Old commit ID`)
	flag.StringVar(&diffOptions.Commit[1], "new", "HEAD", `New commit ID`)
	flag.BoolVar(&diffOptions.Test, "test", false, `Loads test code (*_test.go) for imported packages`)
	flag.BoolVar(&diffOptions.PrintPrivate, "private", false, `If output private function`)
	flag.BoolVar(&diffOptions.PrintUnchanged, "unchanged", false, `If output unchanged function`)
	flag.StringVar(&diffOptions.Pkg, "pkg", "main", `Analyse which packages`)
	flag.Parse()

	// Get commits' callgraph
	for i := 0; i < 2; i++ {
		diffOptions.Wg.Add(1)
		go graph.GetCallgraph(&diffOptions, i)
		diffOptions.Wg.Wait()
	}

	diffGraph := diff.GetDiff(diffOptions.Callgraph[0], diffOptions.Callgraph[1])
	diffGraph.OutputDiffGraph(diffOptions.PrintPrivate, diffOptions.PrintUnchanged, diffOptions.Pkg)
}
