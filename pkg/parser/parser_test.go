package parser

import (
	"testing"

	"github.com/Azure/gocover/pkg/gittool"
)

func TestFindState(t *testing.T) {
	t.Run("findState", func(t *testing.T) {
		// change is nil
		state := findState(&StmtExtent{}, nil)
		if state != Original {
			t.Errorf("nil change should return %s, but get %s", Original, state)
		}

		state = findState(&StmtExtent{startLine: 1, endLine: 10}, &gittool.Change{Sections: []*gittool.Section{{StartLine: 11, EndLine: 20}}})
		if state != Original {
			t.Errorf("change and statment does not overlap should return %s, but get %s", Original, state)
		}

		state = findState(&StmtExtent{startLine: 1, endLine: 1}, &gittool.Change{Sections: []*gittool.Section{{StartLine: 1, EndLine: 1}}})
		if state != Changed {
			t.Errorf("change and statment overlap should return %s, but get %s", Changed, state)
		}

		state = findState(&StmtExtent{startLine: 1, endLine: 10}, &gittool.Change{Sections: []*gittool.Section{{StartLine: 5, EndLine: 13}}})
		if state != Changed {
			t.Errorf("change and statment overlap should return %s, but get %s", Changed, state)
		}
	})
}
