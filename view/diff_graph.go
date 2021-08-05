package view

import (
	"calldiff/common"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"unicode"

	"github.com/awalterschulze/gographviz"
)

type DiffType int

const (
	UNCHANGED = iota // 不变
	INSERTED         // 新增
	REMOVED          // 删除
	CHANGED          // 变化
	AFFECTED         // 传播中受到了影响
)

type DiffEdge struct {
	Node       *DiffNode //连接的点
	Difference DiffType
}

type DiffNode struct {
	Name       string               //函数名称
	Difference DiffType             //0本身代码无变化，1新增，2删除，3本身的代码改变
	CallEdge   map[string]*DiffEdge //调用的函数，map[调用的函数名称]
}

func (n *DiffNode) GetPkgName() string {
	splits := strings.Split(n.Name, "#")
	return splits[1]
}

func (n *DiffNode) GetPath() string {
	splits := strings.Split(n.Name, "#")
	return splits[0]
}

func cleanPathSep(p string) string {
	p = strings.ReplaceAll(p, "/", "__")
	p = strings.ReplaceAll(p, "-", "_")
	return strings.ReplaceAll(p, ".", "_")
}

func (n *DiffNode) GetFuncName() string {
	splits := strings.Split(n.Name, "#")
	if len(splits[3]) == 0 {
		return splits[2]
	} else {
		return fmt.Sprintf("(%s)%s", splits[3], splits[2])
	}
}

func (n *DiffNode) IsPrivate() bool {
	splits := strings.Split(n.Name, "#")
	match, _ := regexp.MatchString("(\\([1-9][0-9]*\\))init", splits[2])
	return !unicode.IsUpper([]rune(splits[2])[0]) && splits[2] != "main" && !match
}

func (n *DiffNode) GetPrettyName() string {
	return fmt.Sprintf("%s.%s", n.GetPkgName(), n.GetFuncName())
}

type DiffGraph struct {
	Nodes map[string]*DiffNode
}

//方便申请节点
func NewDiffGraphHelper() *DiffGraph {
	var ans = new(DiffGraph)
	ans.Nodes = make(map[string]*DiffNode)
	return ans
}

func NewDiffNodeHelper() *DiffNode {
	var ans = new(DiffNode)
	ans.CallEdge = make(map[string]*DiffEdge)
	return ans
}

func NewDiffEdgeHelper(n *DiffNode) *DiffEdge {
	var ans = new(DiffEdge)
	ans.Difference = UNCHANGED
	ans.Node = n
	return ans
}

func (g *DiffGraph) DebugDiffGraph() {
	for key, value := range g.Nodes {
		fmt.Println(key)
		for callname := range value.CallEdge {
			fmt.Println("..", callname)
		}
	}
}

func (g *DiffGraph) OutputDiffGraph(o *common.DiffOptions) {
	//g.CalcAffected()  // 计算哪些节点是黄色节点/受影响节点
	outputs := strings.Split(o.Output, ",")
	_ = os.Mkdir("./output", os.ModePerm)
	for _, output := range outputs {
		switch output {
		case "json":
			err := OutputJson(g, o.PrintPrivate, o.PrintUnchanged, o.Pkg)
			if err != nil {
				fmt.Println(err)
			}
		case "graphviz":
			err := g.Visualization(o.PrintPrivate, o.PrintUnchanged, o.Pkg)
			if err != nil {
				fmt.Println(err)
			} // graphviz 可视化
		default:
			fmt.Println("Unsupported output type", output)
		}
	}
}

func dfsDiffNode(n *DiffNode, doPrintPrivate bool, doPrintUnchanged bool, vis *map[*DiffNode]struct{}) {
	(*vis)[n] = struct{}{}
	for _, edge := range n.CallEdge {
		if _, ok := (*vis)[edge.Node]; ok {
			continue
		}
		if !doPrintPrivate && edge.Node.IsPrivate() {
			continue
		}
		if !doPrintUnchanged && edge.Difference == UNCHANGED {
			continue
		}
		dfsDiffNode(edge.Node, doPrintPrivate, doPrintUnchanged, vis)
	}
}

func (g *DiffGraph) Visualization(doPrintPrivate bool, doPrintUnchanged bool, pkg string) error {
	graphAst, _ := gographviz.ParseString(`digraph G {}`)
	graph := gographviz.NewGraph()
	if err := gographviz.Analyse(graphAst, graph); err != nil {
		return err
	}
	err := graph.AddAttr("G", "rankdir", `"LR"`)
	if err != nil {
		return err
	}
	// 定义属性
	lineColorMap := map[DiffType]string{
		UNCHANGED: "\"#000000\"",
		INSERTED:  "\"#82B366\"",
		REMOVED:   "\"#B85450\"",
		CHANGED:   "\"#D79B00\"",
		AFFECTED:  "\"#D7B953\"",
	}
	lineStyleMap := map[DiffType]string{
		UNCHANGED: `""`,
		INSERTED:  `""`,
		REMOVED:   `dashed`,
		CHANGED:   `""`,
		AFFECTED:  `""`,
	}
	fillColorMap := map[DiffType]string{
		UNCHANGED: "\"#DAE8FC\"",
		INSERTED:  "\"#D5E8D4\"",
		REMOVED:   "\"#F8CECC\"",
		CHANGED:   "\"#FFE6CC\"",
		AFFECTED:  "\"#FFF2CD\"",
	}
	// 遍历确定哪些节点可达
	vis := make(map[*DiffNode]struct{})
	for _, node := range g.Nodes {
		if _, ok := vis[node]; !ok {
			if node.GetPkgName() != pkg {
				continue
			}
			if !doPrintPrivate && node.IsPrivate() {
				continue
			}
			if !doPrintUnchanged && node.Difference == UNCHANGED {
				continue
			}
			dfsDiffNode(node, doPrintPrivate, doPrintUnchanged, &vis)
		}
	}
	// 将所有节点加入到图中
	for _, node := range g.Nodes {
		if _, ok := vis[node]; !ok {
			continue
		}
		if !graph.IsSubGraph(node.GetPkgName()) {
			_ = graph.AddSubGraph("G", `cluster_`+cleanPathSep(node.GetPath()), map[string]string{ // 必须以cluster开头，否则不加框
				"label": "\"" + node.GetPkgName() + "\n(" + node.GetPath() + ")\"",
			})
		}
		_ = graph.AddNode(`cluster_`+cleanPathSep(node.GetPath()), `"`+node.Name+`"`, map[string]string{
			"color":     lineColorMap[node.Difference],
			"label":     `"` + node.GetFuncName() + `"`,
			"style":     "filled",
			"fillcolor": fillColorMap[node.Difference],
		})
	}
	// 添加边
	for _, node := range g.Nodes {
		if _, ok := vis[node]; !ok {
			continue
		}
		for _, edge := range node.CallEdge {
			if _, ok := vis[edge.Node]; !ok {
				continue
			}
			if !doPrintUnchanged && edge.Difference == UNCHANGED {
				continue
			}
			_ = graph.AddEdge(`"`+node.Name+`"`, `"`+edge.Node.Name+`"`, true, map[string]string{
				"color": lineColorMap[edge.Difference],
				"style": lineStyleMap[edge.Difference],
			})
		}
	}
	GenerateLegend(graph, lineColorMap, fillColorMap, lineStyleMap)
	err = ioutil.WriteFile("./output/difference.gv", []byte(graph.String()), 0644)
	if err != nil {
		return err
	}
	err = execCommand(`dot`, "./output/difference.gv", "-Tsvg", "-o", "./output/difference.svg")
	if err != nil {
		return err
	}
	return nil
}

func GenerateLegend(graph *gographviz.Graph, lineColorMap map[DiffType]string, fillColorMap map[DiffType]string, lineStyleMap map[DiffType]string) {
	legendClusterName := `cluster_legend__`
	_ = graph.AddSubGraph("G", legendClusterName, map[string]string{
		"rank":  "sink",
		"label": `"Legend"`,
	})
	_ = graph.AddNode(legendClusterName, "key", map[string]string{
		"label": `<<table border="0" cellpadding="2" cellspacing="0" cellborder="0">
      <tr><td align="right" port="i1">remove call</td></tr>
      <tr><td align="right" port="i2">insert call</td></tr>
      <tr><td align="right" port="i3">affected call</td></tr>
      <tr><td align="right" port="i4">unchanged</td></tr>
      </table>>`,
		"shape":    "plaintext",
		"fontsize": "10",
	})
	_ = graph.AddNode(legendClusterName, "key2", map[string]string{
		"label": `<<table border="0" cellpadding="2" cellspacing="0" cellborder="0">
      <tr><td port="i1">&nbsp;</td></tr>
      <tr><td port="i2">&nbsp;</td></tr>
      <tr><td port="i3">&nbsp;</td></tr>
      <tr><td port="i4">&nbsp;</td></tr>
      </table>>`,
		"shape":    "plaintext",
		"fontsize": "10",
	})
	_ = graph.AddNode(legendClusterName, "legend_removed_api__", map[string]string{
		"label":     `"removed\napi"`,
		"style":     "filled",
		"shape":     "oval",
		"color":     lineColorMap[REMOVED],
		"fillcolor": fillColorMap[REMOVED],
		"fontsize":  "6",
	})
	_ = graph.AddNode(legendClusterName, "legend_new_api__", map[string]string{
		"label":     `"inserted\napi"`,
		"style":     "filled",
		"shape":     "oval",
		"color":     lineColorMap[INSERTED],
		"fillcolor": fillColorMap[INSERTED],
		"fontsize":  "6",
	})
	_ = graph.AddNode(legendClusterName, "legend_affected_api__", map[string]string{
		"label":     `"affected\napi"`,
		"style":     "filled",
		"shape":     "oval",
		"color":     lineColorMap[AFFECTED],
		"fillcolor": fillColorMap[AFFECTED],
		"fontsize":  "6",
	})
	_ = graph.AddNode(legendClusterName, "legend_unchanged_api__", map[string]string{
		"label":     `"unchanged\napi"`,
		"style":     "filled",
		"shape":     "oval",
		"color":     lineColorMap[UNCHANGED],
		"fillcolor": fillColorMap[UNCHANGED],
		"fontsize":  "6",
	})
	_ = graph.AddNode(legendClusterName, "legend_changed_api__", map[string]string{
		"label":     `"unchanged\napi"`,
		"style":     "filled",
		"shape":     "oval",
		"color":     lineColorMap[CHANGED],
		"fillcolor": fillColorMap[CHANGED],
		"fontsize":  "6",
	})
	_ = graph.AddNode(legendClusterName, "key:i1:e -> key2:i1:w", map[string]string{
		"color": lineColorMap[REMOVED],
		"style": lineStyleMap[REMOVED],
	})
	_ = graph.AddNode(legendClusterName, "key:i2:e -> key2:i2:w", map[string]string{
		"color": lineColorMap[INSERTED],
		"style": lineStyleMap[INSERTED],
	})
	_ = graph.AddNode(legendClusterName, "key:i3:e -> key2:i3:w", map[string]string{
		"color": lineColorMap[AFFECTED],
		"style": lineStyleMap[AFFECTED],
	})
	_ = graph.AddNode(legendClusterName, "key:i4:e -> key2:i4:w", map[string]string{
		"color": lineColorMap[UNCHANGED],
		"style": lineStyleMap[UNCHANGED],
	})
	_ = graph.AddNode(legendClusterName, "legend_removed_api__ -> legend_new_api__", map[string]string{
		"style": "invis",
	})
	_ = graph.AddNode(legendClusterName, "legend_affected_api__ -> legend_changed_api__", map[string]string{
		"style": "invis",
	})
}

func execCommand(programName string, programArgs ...string) error {
	cmd := exec.Command(programName, programArgs...)
	stdout, err := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()
	if err != nil {
		return err
	}
	defer func(stdout io.ReadCloser) {
		_ = stdout.Close()
	}(stdout)
	defer func(stderr io.ReadCloser) {
		_ = stderr.Close()
	}(stderr)
	if err := cmd.Start(); err != nil { // 运行命令
		return err
	}
	if opBytes, err := ioutil.ReadAll(stdout); err != nil { // 读取输出结果
		return err
	} else {
		if len(opBytes) >= 2 {
			log.Println(string(opBytes))
		}
	}
	if opBytes, err := ioutil.ReadAll(stderr); err != nil { // 读取输出结果
		log.Fatal(err)
	} else {
		if len(opBytes) >= 2 {
			log.Println(string(opBytes))
		}
	}
	return nil
}

func (g *DiffGraph) CalcAffected() {
	for _, value := range g.Nodes {
		if value.Difference != UNCHANGED {
			continue
		}
		for _, edge := range value.CallEdge {
			if edge.Difference == CHANGED {
				value.Difference = AFFECTED
				break
			}
		}
	}
}
