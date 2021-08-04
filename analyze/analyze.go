package analyze

import (
	"calldiff/view"
	"golang.org/x/tools/go/callgraph"
)

//给diffGraph添加上点集
func makeDiffNode(oldGraph *Graph, newGraph *Graph, diffGraph *view.DiffGraph) {
	//求出删去的接口
	for key := range oldGraph.nodes {
		if _, ok := newGraph.nodes[key]; !ok {
			diffGraph.Nodes[key] = view.NewDiffNodeHelper()
			diffGraph.Nodes[key].Name = key
			diffGraph.Nodes[key].Difference = view.REMOVED
		}
	}
	//求出新增的接口和一直有的接口（class暂标为1）
	for key, node2 := range newGraph.nodes {
		diffGraph.Nodes[key] = view.NewDiffNodeHelper()
		diffGraph.Nodes[key].Name = key
		if node1, ok := oldGraph.nodes[key]; !ok {
			diffGraph.Nodes[key].Difference = view.INSERTED
		} else {
			if isEqual(node1, node2) {
				diffGraph.Nodes[key].Difference = view.UNCHANGED
			} else {
				diffGraph.Nodes[key].Difference = view.CHANGED
			}
		}
	}
}

//给diffGraph添加上两边均有的调用
//建立强连通图,并求出各连通部分的是否改变,并将强连通图上的改变映射回原图
func makeSameEdge(oldGraph *Graph, newGraph *Graph, diffGraph *view.DiffGraph) {
	var interGraph = intersectGraph(oldGraph, newGraph)
	var sccGraph = makeSccGraph(interGraph)
	//按分量影响到每个实际的node
	for key, value := range interGraph.nodes {
		for callname := range value.calledge {
			diffGraph.Nodes[key].CallEdge[callname] = view.NewDiffEdgeHelper(diffGraph.Nodes[callname])
			if sccGraph.belongs[callname].isChanged {
				diffGraph.Nodes[key].CallEdge[callname].Difference = view.CHANGED
			} else {
				diffGraph.Nodes[key].CallEdge[callname].Difference = view.UNCHANGED
			}
		}
	}
}

//给diffGraph添加上新增或者删去的边集
func makeDiffEdge(oldGraph *Graph, newGraph *Graph, diffGraph *view.DiffGraph) {
	for key, value := range diffGraph.Nodes {
		if value.Difference == view.UNCHANGED || value.Difference == view.CHANGED {
			//添加新增的调用
			for callname := range newGraph.nodes[key].calledge {
				if _, ok := oldGraph.nodes[key].calledge[callname]; !ok {
					value.CallEdge[callname] = view.NewDiffEdgeHelper(diffGraph.Nodes[callname])
					value.CallEdge[callname].Difference = view.INSERTED
				}
			}
			//添加删去的调用
			for callname := range oldGraph.nodes[key].calledge {
				if _, ok := newGraph.nodes[key].calledge[callname]; !ok {
					value.CallEdge[callname] = view.NewDiffEdgeHelper(diffGraph.Nodes[callname])
					value.CallEdge[callname].Difference = view.REMOVED
				}
			}
		} else if value.Difference == view.INSERTED {
			for callname := range newGraph.nodes[key].calledge {
				value.CallEdge[callname] = view.NewDiffEdgeHelper(diffGraph.Nodes[callname])
				value.CallEdge[callname].Difference = view.INSERTED
			}
		} else if value.Difference == view.REMOVED {
			for callName := range oldGraph.nodes[key].calledge {
				value.CallEdge[callName] = view.NewDiffEdgeHelper(diffGraph.Nodes[callName])
				value.CallEdge[callName].Difference = view.REMOVED
			}
		}
	}
}

func GetDiff(oldCallgraph *callgraph.Graph, newCallgraph *callgraph.Graph) *view.DiffGraph {
	var oldGraph = callgraph2graph(oldCallgraph)
	var newGraph = callgraph2graph(newCallgraph)
	var diffGraph = view.NewDiffGraphHelper()
	makeDiffNode(oldGraph, newGraph, diffGraph)
	makeSameEdge(oldGraph, newGraph, diffGraph)
	makeDiffEdge(oldGraph, newGraph, diffGraph)
	diffGraph.CalcAffected() // 计算哪些节点是黄色节点/受影响节点
	return diffGraph
}
