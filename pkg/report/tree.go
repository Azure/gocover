package report

import (
	"path/filepath"
	"strings"

	"golang.org/x/tools/cover"
)

const (
	seperator = "/"
)

type CoverageTree interface {
	// FindOrCreate returns the leaf node that represents the source file (go) if found,
	// otherwise, it will creates the all the nodes along the path to the leaf, and finally return it.
	FindOrCreate(file string) *TreeNode
	Find(pkgPath string) *TreeNode
	CollectCoverageData()
	All() []*AllInformation
}

type Tree *TreeNode

// TreeNode represents the node of multi branches tree.
// Each node contains the basic coverage information,
// includes covered lines count, totol lines count and vioaltion lines count.
// Each internal node has one or many sub node, which is stored in a map, and can be retrieved by node name.
// For the leaf node, it does not have sub node.
type TreeNode struct {
	Name                string               // name
	TotalLines          int64                // total lines account for coverage
	TotalCoveredLines   int64                // covered lines account for coverage
	TotalViolationLines int64                // violation lines that not covered for coverage
	Nodes               map[string]*TreeNode // sub nodes that store in map
	isLeaf              bool                 // whether the node is leaf or internal node
}

func NewCoverageTree(hostpath string) CoverageTree {
	return &coverageTree{
		ModuleHostPath: hostpath,
		Root:           NewTreeNode(hostpath, false),
	}
}

func NewCoverageTreeFromProfiles(hostpath string, profiles []*cover.Profile) CoverageTree {
	coverageTree := NewCoverageTree(hostpath)

	for _, profile := range profiles {
		node := coverageTree.FindOrCreate(profile.FileName)
		for _, b := range profile.Blocks {
			node.TotalLines += int64(b.NumStmt)
			if b.Count > 0 {
				node.TotalCoveredLines += int64(b.NumStmt)
			}
		}
	}

	coverageTree.CollectCoverageData()
	return coverageTree
}

func NewTreeNode(name string, isLeaf bool) *TreeNode {
	return &TreeNode{
		Name:   name,
		Nodes:  make(map[string]*TreeNode),
		isLeaf: isLeaf,
	}
}

type coverageTree struct {
	ModuleHostPath string
	Root           Tree
}

var _ CoverageTree = (*coverageTree)(nil)

type AllInformation struct {
	Path              string
	TotalLines        int64
	TotalCoveredLines int64
}

func (p *coverageTree) All() []*AllInformation {
	var result []*AllInformation

	var dfs func(root *TreeNode, contents []string)
	dfs = func(root *TreeNode, contents []string) {
		if root == nil {
			return
		}

		fullpathname := strings.Join(append(contents, root.Name), seperator)
		if p.ModuleHostPath == "" {
			fullpathname = strings.TrimLeft(fullpathname, seperator)
		}

		result = append(result, &AllInformation{
			Path:              fullpathname,
			TotalLines:        root.TotalLines,
			TotalCoveredLines: root.TotalCoveredLines,
		})

		for _, v := range root.Nodes {
			dfs(v, append(contents, root.Name))
		}

	}

	dfs(p.Root, []string{})
	return result
}

func (p *coverageTree) Find(pkgPath string) *TreeNode {
	trimed := strings.TrimPrefix(pkgPath, p.ModuleHostPath)
	tokens := strings.Split(strings.Trim(trimed, seperator), seperator)

	currentNode := p.Root
	for _, name := range tokens {
		if node, ok := currentNode.Nodes[name]; ok {
			currentNode = node
		} else {
			return nil
		}
	}
	return currentNode
}

func (p *coverageTree) FindOrCreate(file string) *TreeNode {
	trimed := strings.TrimPrefix(file, p.ModuleHostPath)
	dir, f := filepath.Split(trimed)
	tokens := strings.Split(strings.Trim(dir, seperator), seperator)

	currentNode := p.Root
	for _, name := range tokens {
		if node, ok := currentNode.Nodes[name]; ok {
			currentNode = node
		} else {
			newNode := NewTreeNode(name, false)
			currentNode.Nodes[name] = newNode
			currentNode = newNode
		}
	}

	leaf := NewTreeNode(f, true)
	currentNode.Nodes[f] = leaf
	return leaf
}

func (p *coverageTree) CollectCoverageData() {
	collect(p.Root)
}

// collect collects coverage data bottom-up.
// After collecting, the root node contains the whole coverage view of the go module,
// and return three values: total, covered, violation.
func collect(root *TreeNode) (int64, int64, int64) {
	// when node is nil, return 0 for total, covered, and violation
	if root == nil {
		return 0, 0, 0
	}

	// iterates over each sub node, and collects all the total, covered, and violation to current node.
	var total int64 = 0
	var covered int64 = 0
	var violation int64 = 0
	for _, node := range root.Nodes {
		t, c, v := collect(node)
		total += t
		covered += c
		violation += v
	}

	root.TotalLines += total
	root.TotalCoveredLines += covered
	root.TotalViolationLines += violation

	return root.TotalLines, root.TotalCoveredLines, root.TotalViolationLines
}
