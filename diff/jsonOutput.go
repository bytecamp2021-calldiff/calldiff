package diff

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"io/ioutil"
)

type Output struct {
	Pkg        string     `json:"pkg"`
	ChangeList changeList `json:"change_list"`
}

type changeList struct {
	Modified  []modifiedApi `json:"modified"`
	New       []string      `json:"new"`
	Deleted   []string      `json:"deleted"`
	Unchanged []string      `json:"unchanged"`
}

type modifiedApi struct {
	Name         string         `json:"name"`
	AddedCall    []string       `json:"added_call"`
	DeletedCall  []string       `json:"deleted_call"`
	AffectedCall []affectedCall `json:"affected_call"`
	AstChanged   bool           `json:"ast_changed"`
}

type affectedCall struct {
	Name       string   `json:"name"`
	AffectedBy []string `json:"affected_by"`
}

func OutputJson(g *DiffGraph, doPrintPrivate bool, doPrintUnchanged bool, pkg string) error {
	var o Output
	o.Pkg = pkg
	for _, node := range g.Nodes {
		if node.GetPkgName() == pkg {
			if !doPrintPrivate && node.isPrivate() {
				continue
			}
			switch node.Difference {
			case INSERTED:
				if o.ChangeList.New == nil {
					o.ChangeList.New = []string{}
				}
				o.ChangeList.New = append(o.ChangeList.New, node.GetPrettyName())
			case REMOVED:
				if o.ChangeList.Deleted == nil {
					o.ChangeList.Deleted = []string{}
				}
				o.ChangeList.Deleted = append(o.ChangeList.Deleted, node.GetPrettyName())
			case CHANGED, AFFECTED:
				o.ChangeList.Modified = append(o.ChangeList.Modified, getModificationDetail(g, node))
			case UNCHANGED:
				if doPrintUnchanged {
					if o.ChangeList.Unchanged == nil {
						o.ChangeList.Unchanged = []string{}
					}
					o.ChangeList.Unchanged = append(o.ChangeList.Unchanged, node.GetPrettyName())
				}
			}
		}
	}
	marshal, err := json.MarshalIndent(o, "", "    ")
	if err != nil {
		return err
	}
	if err := ioutil.WriteFile("./output.json", marshal, fs.ModePerm); err != nil {
		return err
	}
	return nil
}

func getModificationDetail(g *DiffGraph, node *DiffNode) (result modifiedApi) {
	result.Name = node.GetPrettyName()
	if node.Difference == CHANGED {
		result.AstChanged = true
	} else if node.Difference == AFFECTED {
		result.AstChanged = false
	} else {
		fmt.Println("error")
	}
	for _, edge := range node.CallEdge {
		switch edge.Difference {
		case INSERTED:
			if result.AddedCall == nil {
				result.AddedCall = []string{}
			}
			result.AddedCall = append(result.AddedCall, edge.Node.GetPrettyName())
		case REMOVED:
			if result.DeletedCall == nil {
				result.DeletedCall = []string{}
			}
			result.DeletedCall = append(result.DeletedCall, edge.Node.GetPrettyName())
		case CHANGED:
			flags := make(map[*DiffNode]bool) // 表示节点是否被遍历过
			for _, node := range g.Nodes {
				flags[node] = false
			}
			affectedBys := make([]*DiffNode, 0)
			findAffectedBy(edge.Node, flags, &affectedBys)
			if result.AffectedCall == nil {
				result.AffectedCall = []affectedCall{}
			}
			affectedBysPretty := make([]string, 0)
			for _, affectedBy := range affectedBys {
				affectedBysPretty = append(affectedBysPretty, affectedBy.GetPrettyName())
			}
			result.AffectedCall = append(result.AffectedCall, affectedCall{
				Name:       edge.Node.GetPrettyName(),
				AffectedBy: affectedBysPretty,
			})
		case UNCHANGED:
		}
	}
	return result
}

func findAffectedBy(node *DiffNode, flags map[*DiffNode]bool, result *[]*DiffNode) {
	if flags[node] {
		return
	}
	flags[node] = true
	if node.Difference == CHANGED {
		*result = append(*result, node)
		return
	}
	for _, edge := range node.CallEdge {
		if edge.Difference == CHANGED {
			findAffectedBy(edge.Node, flags, result)
		}
	}
}
