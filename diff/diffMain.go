package diff

import (
	"golang.org/x/tools/go/callgraph"
)

//给diffGraph添加上点集
func makeDiffNode(oldGraph *Graph, newGraph *Graph, diffGraph *DiffGraph) {
	//求出删去的接口
	for key := range oldGraph.nodes {
		if _, ok := newGraph.nodes[key]; !ok {
			diffGraph.Nodes[key] = newDiffNodeHelper()
			diffGraph.Nodes[key].Name = key
			diffGraph.Nodes[key].Difference = REMOVED
		}
	}
	//求出新增的接口和一直有的接口（class暂标为1）
	for key, node2 := range newGraph.nodes {
		diffGraph.Nodes[key] = newDiffNodeHelper()
		diffGraph.Nodes[key].Name = key
		if node1, ok := oldGraph.nodes[key]; !ok {
			diffGraph.Nodes[key].Difference = INSERTED
		} else {
			if isEqual(node1, node2) {
				diffGraph.Nodes[key].Difference = UNCHANGED
			} else {
				diffGraph.Nodes[key].Difference = CHANGED
			}
		}
	}
}

//给diffGraph添加上两边均有的调用
//建立强连通图,并求出各连通部分的是否改变,并将强连通图上的改变映射回原图
func makeSameEdge(oldGraph *Graph, newGraph *Graph, diffGraph *DiffGraph) {
	var interGraph = intersectGraph(oldGraph, newGraph)
	var sccGraph = makeSccGraph(interGraph)
	//按分量影响到每个实际的node
	for key, value := range interGraph.nodes {
		for callname := range value.calledge {
			diffGraph.Nodes[key].CallEdge[callname] = newDiffEdgeHelper(diffGraph.Nodes[callname])
			if sccGraph.belongs[callname].isChanged {
				diffGraph.Nodes[key].CallEdge[callname].Difference = CHANGED
			} else {
				diffGraph.Nodes[key].CallEdge[callname].Difference = UNCHANGED
			}
		}
	}
}

//给diffGraph添加上新增或者删去的边集
func makeDiffEdge(oldGraph *Graph, newGraph *Graph, diffGraph *DiffGraph) {
	for key, value := range diffGraph.Nodes {
		if value.Difference == UNCHANGED || value.Difference == CHANGED {
			//添加新增的调用
			for callname := range newGraph.nodes[key].calledge {
				if _, ok := oldGraph.nodes[key].calledge[callname]; !ok {
					value.CallEdge[callname] = newDiffEdgeHelper(diffGraph.Nodes[callname])
					value.CallEdge[callname].Difference = INSERTED
				}
			}
			//添加删去的调用
			for callname := range oldGraph.nodes[key].calledge {
				if _, ok := newGraph.nodes[key].calledge[callname]; !ok {
					value.CallEdge[callname] = newDiffEdgeHelper(diffGraph.Nodes[callname])
					value.CallEdge[callname].Difference = REMOVED
				}
			}
		} else if value.Difference == INSERTED {
			for callname := range newGraph.nodes[key].calledge {
				value.CallEdge[callname] = newDiffEdgeHelper(diffGraph.Nodes[callname])
				value.CallEdge[callname].Difference = INSERTED
			}
		} else if value.Difference == REMOVED {
			for callName := range oldGraph.nodes[key].calledge {
				value.CallEdge[callName] = newDiffEdgeHelper(diffGraph.Nodes[callName])
				value.CallEdge[callName].Difference = REMOVED
			}
		}
	}
}

func GetDiff(oldCallgraph *callgraph.Graph, newCallgraph *callgraph.Graph) *DiffGraph {
	var oldGraph = callgraph2graph(oldCallgraph)
	var newGraph = callgraph2graph(newCallgraph)
	var diffGraph = newDiffGraphHelper()
	makeDiffNode(oldGraph, newGraph, diffGraph)
	makeSameEdge(oldGraph, newGraph, diffGraph)
	makeDiffEdge(oldGraph, newGraph, diffGraph)
	diffGraph.calcAffected() // 计算哪些节点是黄色节点/受影响节点
	return diffGraph
}
