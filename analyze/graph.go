package analyze

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"go/types"
	"strings"

	"golang.org/x/tools/go/callgraph"
	"golang.org/x/tools/go/ssa"
)

// Node 函数调用图中的函数节点
type Node struct {
	name       string           //函数的名称
	hashNum    [32]byte         //代码部分求hash过后的值,在两图的交集中0表示两图hashNum一样，否则不一样
	isChanged  bool             //判断有无改变
	callByEdge map[string]*Node //指向所有被调用的函数（即a调用b，b向a连边）
	callEdge   map[string]*Node //所有调用边
}

// Graph 函数调用图
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
	n.callByEdge = make(map[string]*Node)
	n.callEdge = make(map[string]*Node)
	return n
}

func isEqual(n1 *Node, n2 *Node) bool {
	return n1.hashNum == n2.hashNum
}

//求图的交，方便求强连通
func intersectGraph(oldGraph *Graph, newGraph *Graph) *Graph {
	var interGraph = newGraphHelper()
	//构造出新的结点
	for key, n2 := range newGraph.nodes {
		if n1, ok := oldGraph.nodes[key]; ok {
			interGraph.nodes[key] = newNodeHelper()
			interGraph.nodes[key].name = key
			if n2.hashNum != n1.hashNum {
				interGraph.nodes[key].isChanged = true //两hash值做差判断是否发生改变
			}
		}
	}
	//把边加上
	for key, n2 := range newGraph.nodes {
		if n1, ok := oldGraph.nodes[key]; ok {
			for callName := range n2.callEdge {
				if _, ok := n1.callEdge[callName]; ok {
					interGraph.nodes[key].callEdge[callName] = interGraph.nodes[callName]
				}
			}
			for callName := range n2.callByEdge {
				if _, ok := n1.callByEdge[callName]; ok {
					interGraph.nodes[key].callByEdge[callName] = interGraph.nodes[callName]
				}
			}
		}
	}
	return interGraph
}

func func2str(ssaFunction *ssa.Function) string {
	var result string
	result = fmt.Sprintf("%v#%v#%v#", ssaFunction.Pkg.Pkg.Path(), ssaFunction.Pkg.Pkg.Name(), ssaFunction.Name()) // 包名#函数名
	if ssaFunction.Signature.Recv() != nil {
		var buf bytes.Buffer
		types.WriteType(&buf, ssaFunction.Signature.Recv().Type(), nil)
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
	return sha256.Sum256([]byte(resultString))
}

func callGraph2graph(cg *callgraph.Graph) *Graph {
	var g = newGraphHelper()
	nodeMap := make(map[*callgraph.Node]struct{})
	for key, value := range cg.Nodes {
		nodeMap[value] = struct{}{}
		s := func2str(key)
		g.nodes[s] = newNodeHelper()
		g.nodes[s].name = s
		g.nodes[s].hashNum = getFuncHash(key)
	}
	for node := range nodeMap {
		for _, edge := range node.Out {
			if _, ok := nodeMap[edge.Callee]; ok {
				calleeName := func2str(edge.Callee.Func)
				callerName := func2str(edge.Caller.Func)
				g.nodes[callerName].callEdge[calleeName] = g.nodes[calleeName]
				g.nodes[calleeName].callByEdge[callerName] = g.nodes[callerName]
			}
		}
	}
	return g
}
