package diff

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"go/types"
	"strings"

	"golang.org/x/tools/go/callgraph"
	"golang.org/x/tools/go/ssa"
)

type Node struct {
	name       string           //函数的名称
	hashNum    [32]byte         //代码部分求hash过后的值,在两图的交集中0表示两图hashNum一样，否则不一样
	isChanged  bool             //判断有无改变
	callbyedge map[string]*Node //指向所有被调用的函数（即a调用b，b向a连边）
	calledge   map[string]*Node //所有调用边
}

type Graph struct {
	nodes map[string]*Node
}

func newGraphHelper() *Graph {
	var g = new(Graph)
	g.nodes = make(map[string]*Node)
	return g
}

func newNodeHelper() *Node {
	var n = new(Node)
	n.callbyedge = make(map[string]*Node)
	n.calledge = make(map[string]*Node)
	return n
}

func isEqual(n1 *Node, n2 *Node) bool {
	return n1.hashNum == n2.hashNum
}

func input() (*Graph, *Graph) {
	var g1 = new(Graph)
	var g2 = new(Graph)
	return g1, g2
}

//求图的交，方便求强连通
func intersectGraph(g1 *Graph, g2 *Graph) *Graph {
	var g3 = newGraphHelper()
	//构造出新的结点
	for key, n2 := range g2.nodes {
		if n1, ok := g1.nodes[key]; ok {
			g3.nodes[key] = newNodeHelper()
			g3.nodes[key].name = key
			if n2.hashNum != n1.hashNum {
				g3.nodes[key].isChanged = true //两hash值做差判断是否发生改变
			}
		}
	}
	//把边加上
	for key, n2 := range g2.nodes {
		if n1, ok := g1.nodes[key]; ok {
			for callname := range n2.calledge {
				if _, ok := n1.calledge[callname]; ok {
					g3.nodes[key].calledge[callname] = g3.nodes[callname]
				}
			}
			for callname := range n2.callbyedge {
				if _, ok := n1.callbyedge[callname]; ok {
					g3.nodes[key].callbyedge[callname] = g3.nodes[callname]
				}
			}
		}
	}
	return g3
}

func func2str(ssaFunction *ssa.Function) string {
	var result string
	result = fmt.Sprintf("%v#%v#%v#", ssaFunction.Pkg.Pkg.Path(), ssaFunction.Pkg.Pkg.Name(), ssaFunction.Name()) // 包名#函数名
	// fmt.Println("xxx",result)
	if ssaFunction.Signature.Recv() != nil {
		var buf bytes.Buffer
		types.WriteType(&buf, ssaFunction.Signature.Recv().Type(), nil)
		// fmt.Println(buf.String())
		result += buf.String()
	}
	return result
}

func getFuncHash(ssaFunction *ssa.Function) [32]byte {
	var b bytes.Buffer
	ssa.WriteFunction(&b, ssaFunction)
	lines := strings.Split(string(b.Bytes()), "\n")
	resultString := ""
	state := false
	for _, line := range lines {
		if !state {
			if !strings.HasPrefix(line, "#") {
				state = true
			}
		}
		if state {
			resultString += line
		}
	}
	// fmt.Println(resultString)
	return sha256.Sum256([]byte(resultString))
}

func callgraph2graph(cg *callgraph.Graph) *Graph {
	var g = newGraphHelper()
	nodeMap := make(map[*callgraph.Node]struct{})
	for key, value := range cg.Nodes {
		nodeMap[value] = struct{}{}
		s := func2str(key)
		g.nodes[s] = newNodeHelper()
		g.nodes[s].name = s
		g.nodes[s].hashNum = getFuncHash(key)
	}
	for node, _ := range nodeMap {
		for _, edge := range node.Out {
			if _, ok := nodeMap[edge.Callee]; ok {
				calleeName := func2str(edge.Callee.Func)
				callerName := func2str(edge.Caller.Func)
				g.nodes[callerName].calledge[calleeName] = g.nodes[calleeName]
				g.nodes[calleeName].callbyedge[callerName] = g.nodes[callerName]
			}
		}
	}
	/*
		callgraph.GraphVisitEdges(cg, func(edge *callgraph.Edge) error {
			if inStd(edge.Caller) || inStd(edge.Callee) {
				return nil
			}
			calleeName := func2str(edge.Callee.Func)
			callerName := func2str(edge.Caller.Func)
			g.nodes[callerName].calledge[calleeName] = g.nodes[calleeName]
			g.nodes[calleeName].callbyedge[callerName] = g.nodes[callerName]
			return nil
		})
	*/
	return g
}
