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
						Name:                       "leaf1",
						TotalLines:                 120,
						TotalEffectiveLines:        100,
						TotalIgnoredLines:          20,
						TotalCoveredLines:          80,
						TotalViolationLines:        20,
						TotalCoveredButIgnoreLines: 1,
						isLeaf:                     true,
					},
					"child3": {
						Name: "child3",
						Nodes: map[string]*TreeNode{
							"leaf3": {
								Name:                       "leaf3",
								TotalLines:                 110,
								TotalEffectiveLines:        80,
								TotalIgnoredLines:          30,
								TotalCoveredLines:          50,
								TotalViolationLines:        30,
								TotalCoveredButIgnoreLines: 2,
								isLeaf:                     true,
							},
						},
					},
				},
			},
			"child2": {
				Name: "child2",
				Nodes: map[string]*TreeNode{
					"leaf20": {
						Name:                       "leaf20",
						TotalLines:                 60,
						TotalEffectiveLines:        50,
						TotalIgnoredLines:          10,
						TotalCoveredLines:          40,
						TotalViolationLines:        10,
						TotalCoveredButIgnoreLines: 3,
						isLeaf:                     true,
					},
					"leaf21": {
						Name:                       "leaf21",
						TotalLines:                 90,
						TotalEffectiveLines:        60,
						TotalIgnoredLines:          30,
						TotalCoveredLines:          30,
						TotalViolationLines:        30,
						TotalCoveredButIgnoreLines: 4,
						isLeaf:                     true,
					},
				},
			},
		},
	}
}

func TestCoverageTree(t *testing.T) {

	t.Run("collect when root is nil", func(t *testing.T) {
		var root *TreeNode
		total, effectived, ignored, covered, violation, coveredButIgnored := collect(root)
		if total != 0 {
			t.Errorf("total expected 0, but get %d", total)
		}
		if effectived != 0 {
			t.Errorf("effectived expected 0, but get %d", effectived)
		}
		if ignored != 0 {
			t.Errorf("ignored expected 0, but get %d", ignored)
		}
		if covered != 0 {
			t.Errorf("covered expected 0, but get %d", covered)
		}
		if violation != 0 {
			t.Errorf("violation expected 0, but get %d", violation)
		}
		if coveredButIgnored != 0 {
			t.Errorf("coveredButIgnored expected 0, but get %d", coveredButIgnored)
		}
	})

	t.Run("collect when root contains all the statistical data", func(t *testing.T) {
		beforeRun()

		total, effectived, ignored, covered, violation, coveredButIgnored := collect(root)
		if total != 380 {
			t.Errorf("total expected 380, but get %d", total)
		}
		if effectived != 290 {
			t.Errorf("effectived expected 290, but get %d", effectived)
		}
		if ignored != 90 {
			t.Errorf("ignored expected 90, but get %d", ignored)
		}
		if covered != 200 {
			t.Errorf("covered expected 200, but get %d", covered)
		}
		if violation != 90 {
			t.Errorf("violation expected 90, but get %d", violation)
		}
		if coveredButIgnored != 10 {
			t.Errorf("coveredButIgnored expected 0, but get %d", coveredButIgnored)
		}
	})

	t.Run("FindOrCreate", func(t *testing.T) {
		coverageTree := NewCoverageTree("github.com/Azure/gocover")
		node := coverageTree.FindOrCreate("pkg/util/bar.go")
		if node.Name != "bar.go" {
			t.Errorf("expect name of leaf node bar.go, but get %s", node.Name)
		}

		node2 := coverageTree.FindOrCreate("pkg/util/bar.go")
		if node != node2 {
			t.Errorf("should same node")
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
		if coverageTree.Root.TotalLines != 380 {
			t.Errorf("total expected 290, but get %d", coverageTree.Root.TotalLines)
		}
		if coverageTree.Root.TotalEffectiveLines != 290 {
			t.Errorf("effectived expected 290, but get %d", coverageTree.Root.TotalEffectiveLines)
		}
		if coverageTree.Root.TotalIgnoredLines != 90 {
			t.Errorf("ignored expected 90, but get %d", coverageTree.Root.TotalIgnoredLines)
		}
		if coverageTree.Root.TotalCoveredLines != 200 {
			t.Errorf("covered expected 200, but get %d", coverageTree.Root.TotalCoveredLines)
		}
		if coverageTree.Root.TotalViolationLines != 90 {
			t.Errorf("violation expected 90, but get %d", coverageTree.Root.TotalViolationLines)
		}
		if coverageTree.Root.TotalCoveredButIgnoreLines != 10 {
			t.Errorf("coveredButIgnored expected 90, but get %d", coverageTree.Root.TotalCoveredButIgnoreLines)
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
		if node.TotalLines != 120 {
			t.Errorf("total should 120, but %d", node.TotalLines)
		}

		node = coverageTree.Find("child2")
		if node == nil {
			t.Errorf("shoud not return nil")
		}
		if node.isLeaf == true {
			t.Errorf("internal node")
		}
		if node.TotalLines != 150 {
			t.Errorf("total should 150, but %d", node.TotalLines)
		}

		node = coverageTree.Find("child3")
		if node != nil {
			t.Errorf("should return nil when not found")
		}
	})
}
