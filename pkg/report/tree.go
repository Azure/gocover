package report

import (
	"path/filepath"
	"strings"
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
	Statistics() *AllInformation
}

type Tree *TreeNode

// TreeNode represents the node of multi branches tree.
// Each node contains the basic coverage information,
// includes covered lines count, totol lines count and vioaltion lines count.
// Each internal node has one or many sub node, which is stored in a map, and can be retrieved by node name.
// For the leaf node, it does not have sub node.
type TreeNode struct {
	Name                       string // name
	TotalLines                 int64  // total lines of the entire repo/module.
	TotalEffectiveLines        int64  // the lines that for coverage, total lines - ignored lines
	TotalIgnoredLines          int64  // the lines ignored
	TotalCoveredLines          int64  // covered lines account for coverage
	TotalViolationLines        int64  // violation lines that not covered for coverage
	TotalCoveredButIgnoreLines int64  // the lines that covered but ignored
	CoverageProfile            *CoverageProfile
	Nodes                      map[string]*TreeNode // sub nodes that store in map
	isLeaf                     bool                 // whether the node is leaf or internal node
}

func NewCoverageTree(modulePath string) CoverageTree {
	return &coverageTree{
		ModuleHostPath: modulePath,
		Root:           NewTreeNode(modulePath, false),
	}
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
	Path                       string
	TotalLines                 int64
	TotalEffectiveLines        int64
	TotalIgnoredLines          int64
	TotalCoveredLines          int64
	TotalViolationLines        int64
	TotalCoveredButIgnoreLines int64
}

func (p *coverageTree) Statistics() *AllInformation {
	return &AllInformation{
		Path:                       p.Root.Name,
		TotalLines:                 p.Root.TotalLines,
		TotalEffectiveLines:        p.Root.TotalEffectiveLines,
		TotalIgnoredLines:          p.Root.TotalIgnoredLines,
		TotalCoveredLines:          p.Root.TotalCoveredLines,
		TotalViolationLines:        p.Root.TotalViolationLines,
		TotalCoveredButIgnoreLines: p.Root.TotalCoveredButIgnoreLines,
	}
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
			Path:                       fullpathname,
			TotalLines:                 root.TotalLines,
			TotalEffectiveLines:        root.TotalEffectiveLines,
			TotalIgnoredLines:          root.TotalIgnoredLines,
			TotalCoveredLines:          root.TotalCoveredLines,
			TotalViolationLines:        root.TotalViolationLines,
			TotalCoveredButIgnoreLines: root.TotalCoveredButIgnoreLines,
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

	if leaf, ok := currentNode.Nodes[f]; ok {
		return leaf
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
// and return five values: total, effectived, ignored, covered, violation, coveredButIgnored.
func collect(root *TreeNode) (int64, int64, int64, int64, int64, int64) {
	// when node is nil, return 0 for total, covered, and violation
	if root == nil {
		return 0, 0, 0, 0, 0, 0
	}

	// iterates over each sub node, and collects all the total, covered, and violation to current node.
	var total int64 = 0
	var effectived int64 = 0
	var ignored int64 = 0
	var covered int64 = 0
	var violation int64 = 0
	var coveredButIgnored int64 = 0
	for _, node := range root.Nodes {
		t, e, i, c, v, d := collect(node)
		total += t
		effectived += e
		ignored += i
		covered += c
		violation += v
		coveredButIgnored += d
	}

	root.TotalLines += total
	root.TotalEffectiveLines += effectived
	root.TotalIgnoredLines += ignored
	root.TotalCoveredLines += covered
	root.TotalViolationLines += violation
	root.TotalCoveredButIgnoreLines += coveredButIgnored

	return root.TotalLines,
		root.TotalEffectiveLines,
		root.TotalIgnoredLines,
		root.TotalCoveredLines,
		root.TotalViolationLines,
		root.TotalCoveredButIgnoreLines
}
