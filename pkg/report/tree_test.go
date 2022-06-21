package report

import "testing"

var root *TreeNode

func beforeRun() {
	// root
	//    |-- child1
	//    |   |-- leaf1
	//    |   |-- child3
	//    |       |-- leaf3
	//    |-- child2
	//        |-- leaf20
	//        |-- leaf21
	root = &TreeNode{
		Name: "github.com/Azure/gocover",
		Nodes: map[string]*TreeNode{
			"child1": {
				Name: "child1",
				Nodes: map[string]*TreeNode{
					"leaf1": {
						Name:                "leaf1",
						TotalLines:          100,
						TotalCoveredLines:   80,
						TotalViolationLines: 20,
						isLeaf:              true,
					},
					"child3": {
						Name: "child3",
						Nodes: map[string]*TreeNode{
							"leaf3": {
								Name:                "leaf3",
								TotalLines:          80,
								TotalCoveredLines:   50,
								TotalViolationLines: 30,
								isLeaf:              true,
							},
						},
					},
				},
			},
			"child2": {
				Name: "child2",
				Nodes: map[string]*TreeNode{
					"leaf20": {
						Name:                "leaf20",
						TotalLines:          50,
						TotalCoveredLines:   40,
						TotalViolationLines: 10,
						isLeaf:              true,
					},
					"leaf21": {
						Name:                "leaf21",
						TotalLines:          60,
						TotalCoveredLines:   30,
						TotalViolationLines: 30,
						isLeaf:              true,
					},
				},
			},
		},
	}
}

func TestCoverageTree(t *testing.T) {

	t.Run("collect when root is nil", func(t *testing.T) {
		var root *TreeNode
		total, covered, violation := collect(root)
		if total != 0 {
			t.Errorf("total expected 0, but get %d", total)
		}
		if covered != 0 {
			t.Errorf("covered expected 0, but get %d", covered)
		}
		if violation != 0 {
			t.Errorf("violation expected 0, but get %d", violation)
		}
	})

	t.Run("collect when root contains all the statistical data", func(t *testing.T) {
		beforeRun()

		total, covered, violation := collect(root)
		if total != 290 {
			t.Errorf("total expected 290, but get %d", total)
		}
		if covered != 200 {
			t.Errorf("covered expected 200, but get %d", covered)
		}
		if violation != 90 {
			t.Errorf("violation expected 90, but get %d", violation)
		}
	})

	t.Run("FindOrCreate", func(t *testing.T) {
		coverageTree := NewCoverageTree("github.com/Azure/gocover")
		node := coverageTree.FindOrCreate("pkg/util/bar.go")
		if node.Name != "bar.go" {
			t.Errorf("expect name of leaf node bar.go, but get %s", node.Name)
		}
	})

	// TODO: handle empty string
	t.Run("FindOrCreate", func(t *testing.T) {
		coverageTree := NewCoverageTree("github.com/Azure/gocover")
		node := coverageTree.FindOrCreate("")
		if node.Name != "" {
			t.Errorf("expect name of leaf node bar.go, but get %s", node.Name)
		}
	})

	t.Run("CollectCoverageData", func(t *testing.T) {
		beforeRun()

		coverageTree := &coverageTree{
			ModuleHostPath: "github.com/Azure/gocover",
			Root:           root,
		}
		coverageTree.CollectCoverageData()
		if coverageTree.Root.TotalLines != 290 {
			t.Errorf("total expected 290, but get %d", coverageTree.Root.TotalLines)
		}
		if coverageTree.Root.TotalCoveredLines != 200 {
			t.Errorf("covered expected 200, but get %d", coverageTree.Root.TotalCoveredLines)
		}
		if coverageTree.Root.TotalViolationLines != 90 {
			t.Errorf("violation expected 90, but get %d", coverageTree.Root.TotalViolationLines)
		}
	})

	t.Run("All", func(t *testing.T) {
		beforeRun()

		coverageTree := &coverageTree{}
		coverageTree.CollectCoverageData()
		all := coverageTree.All()
		if len(all) != 0 {
			t.Errorf("should have 0 items, but get %d", len(all))
		}

		coverageTree.ModuleHostPath = ""
		coverageTree.Root = root
		coverageTree.CollectCoverageData()

		all = coverageTree.All()
		if len(all) != 8 {
			t.Errorf("should have 8 items, but get %d", len(all))
		}
	})

	t.Run("Find", func(t *testing.T) {
		beforeRun()

		coverageTree := &coverageTree{
			ModuleHostPath: "github.com/Azure/gocover",
			Root:           root,
		}
		coverageTree.CollectCoverageData()

		node := coverageTree.Find("child1/leaf1")
		if node == nil {
			t.Errorf("shoud not return nil")
		}
		if node.isLeaf == false {
			t.Errorf("leaf node")
		}
		if node.TotalLines != 100 {
			t.Errorf("total should 100, but %d", node.TotalLines)
		}

		node = coverageTree.Find("child2")
		if node == nil {
			t.Errorf("shoud not return nil")
		}
		if node.isLeaf == true {
			t.Errorf("internal node")
		}
		if node.TotalLines != 110 {
			t.Errorf("total should 110, but %d", node.TotalLines)
		}

		node = coverageTree.Find("child3")
		if node != nil {
			t.Errorf("should return nil when not found")
		}
	})
}
