package analyze

import (
	"calldiff/view"
	"fmt"
	"testing"
)

func makeTestGraph(n []int, e [][]int, h []int, g *Graph) {
	name := []string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11", "12", "13", "14"}
	for _, v := range n {
		g.nodes[name[v]] = newNodeHelper()
		g.nodes[name[v]].name = name[v]
	}
	for _, v := range e {
		g.nodes[name[v[0]]].callbyedge[name[v[1]]] = g.nodes[name[v[1]]]
		g.nodes[name[v[1]]].calledge[name[v[0]]] = g.nodes[name[v[0]]]
	}
	for _, v := range h {
		g.nodes[name[v]].isChanged = true
	}
}

func getGraph(id int) (*Graph, *Graph) {
	var g1 = newGraphHelper()
	var g2 = newGraphHelper()
	if id == 1 {
		n1 := []int{1, 2, 3, 4, 5, 6, 8, 9, 10}
		n2 := []int{1, 2, 3, 5, 6, 7, 8, 9, 10}
		e1 := [][]int{{1, 2}, {2, 3}, {1, 3}, {3, 1}, {2, 4}, {2, 5}, {4, 8}, {5, 8}, {9, 6}, {9, 10}}
		e2 := [][]int{{1, 2}, {2, 3}, {1, 3}, {3, 1}, {7, 3}, {2, 5}, {2, 10}, {5, 8}, {9, 10}, {9, 6}}
		var h1 []int
		h2 := []int{3, 10, 8}
		makeTestGraph(n1, e1, h1, g1)
		makeTestGraph(n2, e2, h2, g2)
	}
	return g1, g2
}

func printDiffNode(g *view.DiffGraph) {
	for key := range g.Nodes {
		fmt.Print(key, " ")
	}
}

func TestMakeDiffNode(t *testing.T) {
	var g1, g2 = getGraph(1)
	var g3 = view.NewDiffGraphHelper()
	makeDiffNode(g1, g2, g3)
	printDiffNode(g3)
}

func TestMakeSomeEdge(t *testing.T) {
	var g1, g2 = getGraph(1)
	var g3 = view.NewDiffGraphHelper()
	makeDiffNode(g1, g2, g3)
	makeSameEdge(g1, g2, g3)
	printDiffNode(g3)
}
func TestMakeDiffEdge(t *testing.T) {
	var g1, g2 = getGraph(1)
	var g3 = view.NewDiffGraphHelper()
	makeDiffNode(g1, g2, g3)
	makeSameEdge(g1, g2, g3)
	makeDiffEdge(g1, g2, g3)
	printDiffNode(g3)
}
