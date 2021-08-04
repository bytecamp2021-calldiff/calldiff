package diff

import (
	"golang.org/x/tools/go/callgraph"
)

//给diffGraph添加上点集
func makeDiffNode(g1 *Graph, g2 *Graph, g3 *DiffGraph) {
	//求出删去的接口
	for key := range g1.nodes {
		if _, ok := g2.nodes[key]; !ok {
			g3.Nodes[key] = newDiffNodeHelper()
			g3.Nodes[key].Name = key
			g3.Nodes[key].Difference = REMOVED
		}
	}
	//求出新增的接口和一直有的接口（class暂标为1）
	for key, node2 := range g2.nodes {
		g3.Nodes[key] = newDiffNodeHelper()
		g3.Nodes[key].Name = key
		if node1, ok := g1.nodes[key]; !ok {
			g3.Nodes[key].Difference = INSERTED
		} else {
			if isEqual(node1, node2) {
				g3.Nodes[key].Difference = UNCHANGED
			} else {
				g3.Nodes[key].Difference = CHANGED
			}
		}
	}
}

//给diffGraph添加上两边均有的调用
//建立强连通图,并求出各连通部分的是否改变,并将强连通图上的改变映射回原图
func makeSameEdge(g1 *Graph, g2 *Graph, dg *DiffGraph) {
	var g3 = intersectGraph(g1, g2)
	var cg = makeSccGraph(g3)
	//按分量影响到每个实际的node
	for key, value := range g3.nodes {
		for callname := range value.calledge {
			dg.Nodes[key].CallEdge[callname] = newDiffEdgeHelper(dg.Nodes[callname])
			if cg.belongs[callname].isChanged {
				dg.Nodes[key].CallEdge[callname].Difference = CHANGED
			} else {
				dg.Nodes[key].CallEdge[callname].Difference = UNCHANGED
			}
		}
	}
}

//给diffGraph添加上新增或者删去的边集
func makeDiffEdge(g1 *Graph, g2 *Graph, g3 *DiffGraph) {
	for key, value := range g3.Nodes {
		if value.Difference == UNCHANGED || value.Difference == CHANGED {
			//添加新增的调用
			for callname := range g2.nodes[key].calledge {
				if _, ok := g1.nodes[key].calledge[callname]; !ok {
					value.CallEdge[callname] = newDiffEdgeHelper(g3.Nodes[callname])
					value.CallEdge[callname].Difference = INSERTED
				}
			}
			//添加删去的调用
			for callname := range g1.nodes[key].calledge {
				if _, ok := g2.nodes[key].calledge[callname]; !ok {
					value.CallEdge[callname] = newDiffEdgeHelper(g3.Nodes[callname])
					value.CallEdge[callname].Difference = REMOVED
				}
			}
		} else if value.Difference == INSERTED {
			for callname := range g2.nodes[key].calledge {
				value.CallEdge[callname] = newDiffEdgeHelper(g3.Nodes[callname])
				value.CallEdge[callname].Difference = INSERTED
			}
		} else if value.Difference == REMOVED {
			for callName := range g1.nodes[key].calledge {
				value.CallEdge[callName] = newDiffEdgeHelper(g3.Nodes[callName])
				value.CallEdge[callName].Difference = REMOVED
			}
		}
	}
}

func GetDiff(cg1 *callgraph.Graph, cg2 *callgraph.Graph) *DiffGraph {
	var g1 = callgraph2graph(cg1)
	var g2 = callgraph2graph(cg2)
	var g3 = newDiffGraphHelper()
	makeDiffNode(g1, g2, g3)
	makeSameEdge(g1, g2, g3)
	makeDiffEdge(g1, g2, g3)
	g3.calcAffected() // 计算哪些节点是黄色节点/受影响节点
	return g3
}
