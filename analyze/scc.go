package analyze

// Component 该文件中即为强连通分量部分用于建立新图以进行步骤四五
type Component struct {
	id         int //用以区别不同的强连通分量
	member     []*Node
	callByEdge map[int]*Component //指向所有被调用的函数（即a调用b，b向a连边）
	callEdge   map[int]*Component //所有调用边

	//以下用以过程中计算
	callNum   int  //调用的函数数
	isChanged bool //调用的函数中有无发生改变的
}

// ComponentGraph 强连通图
type ComponentGraph struct {
	belongs map[string]*Component
	nodes   []*Component
}

func newComponentHelper(id int) *Component {
	var c = new(Component)
	c.id = id
	c.callByEdge = make(map[int]*Component)
	c.callEdge = make(map[int]*Component)
	return c
}

func newComponentGraphHelper() *ComponentGraph {
	var g = new(ComponentGraph)
	g.belongs = make(map[string]*Component)
	return g
}

//遍历整张强连通图并标记上是否改变
func (cg *ComponentGraph) traverse() {
	var queue []int
	//找出调用次数为0的分量
	for _, com := range cg.nodes {
		if com.callNum == 0 {
			queue = append(queue, com.id)
		}
	}
	//按拓扑序求解,得出所有变化的分量
	for len(queue) != 0 {
		id := queue[0]
		queue = queue[1:]
		com := cg.nodes[id]
		for key, value := range com.callByEdge {
			if com.isChanged {
				value.isChanged = true
			}
			value.callNum--
			if value.callNum == 0 {
				queue = append(queue, key)
			}
		}
	}
}

//强连通图添加边并标上isChanged等信息
func makeSccGraph(g *Graph) *ComponentGraph {
	var cg = makeSccNodes(g)
	for _, com := range cg.nodes {
		com.callNum = 0
		com.isChanged = false
		for _, mem := range com.member {
			//标志是否发生改变
			if mem.isChanged {
				com.isChanged = true
			}
			for callName := range mem.callEdge {
				if cg.belongs[callName].id == com.id {
					continue
				}
				(*com).callEdge[cg.belongs[callName].id] = cg.belongs[callName]
				com.callNum++
			}
			for callName := range mem.callByEdge {
				if cg.belongs[callName].id == com.id {
					continue
				}
				(*com).callByEdge[cg.belongs[callName].id] = cg.belongs[callName]
			}
		}
	}
	//遍历强连通图传递变化
	cg.traverse()
	return cg
}

//强连通图添加点
func makeSccNodes(g *Graph) *ComponentGraph {
	var cg = newComponentGraphHelper()
	var vs []*Node
	vis := make(map[string]bool)
	for key, value := range g.nodes {
		if _, ok := vis[key]; !ok {
			dfs1(value, &vis, &vs)
		}
	}

	sz := len(vs)
	id := 0
	for k := sz - 1; k >= 0; k-- {
		n := vs[k]
		if _, ok := cg.belongs[n.name]; !ok {
			nowCom := newComponentHelper(id)
			cg.nodes = append(cg.nodes, nowCom)
			dfs2(n, nowCom, &cg.belongs)
			id++
		}
	}
	return cg
}

//求强连通分量，正向dfs
func dfs1(n *Node, vis *map[string]bool, vs *[]*Node) {
	(*vis)[n.name] = true
	for key, value := range n.callEdge {
		if _, ok := (*vis)[key]; !ok {
			dfs1(value, vis, vs)
		}
	}
	*vs = append(*vs, n)
}

//求强连通分量，反向dfs
func dfs2(n *Node, com *Component, belongs *map[string]*Component) {
	(*belongs)[n.name] = com
	com.member = append(com.member, n)
	for key, value := range n.callByEdge {
		if _, ok := (*belongs)[key]; !ok {
			dfs2(value, com, belongs)
		}
	}
}
