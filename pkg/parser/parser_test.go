package parser

import (
	"testing"

	"github.com/Azure/gocover/pkg/gittool"
	"github.com/sirupsen/logrus"
)

func TestSetStatementsState(t *testing.T) {
	t.Run("change is nil", func(t *testing.T) {
		parser := &Parser{}
		parser.setStatementsState(nil, []*statement{})
	})

	t.Run("statements is empty", func(t *testing.T) {
		parser := &Parser{}
		parser.setStatementsState(&gittool.Change{}, []*statement{})
	})

	t.Run("setStatementsState", func(t *testing.T) {
		parser := &Parser{logger: logrus.New()}

		change := &gittool.Change{
			FileName: "foo.go",
			Sections: []*gittool.Section{
				{
					StartLine: 1,
					EndLine:   1,
					Contents:  []string{"type foo string"},
				},
				{
					StartLine: 3,
					EndLine:   3,
					Contents:  []string{"i := 0"},
				},
				{
					StartLine: 5,
					EndLine:   8,
					Contents: []string{
						"  ",
						"// check it",
						"    if statements[i].startLine <= lineNumber && statements[i].endLine >= lineNumber {",
						"        stmt = statements[i]",
					},
				},
				{
					StartLine: 13,
					EndLine:   14,
					Contents:  []string{"// increase i", "i++"},
				},
				{
					StartLine: 17,
					EndLine:   17,
					Contents:  []string{"if err != nil { return err }"},
				},
				{
					StartLine: 18,
					EndLine:   18,
					Contents:  []string{"if err != nil { return err"},
				},
				{
					StartLine: 21,
					EndLine:   21,
					Contents:  []string{" return err"},
				},
				{
					StartLine: 22,
					EndLine:   22,
					Contents:  []string{"var a = 1"},
				},
				{
					StartLine: 24,
					EndLine:   24,
					Contents:  []string{"type myStruct {}"},
				},
				{
					StartLine: 27,
					EndLine:   29,
					Contents: []string{
						"func myFunc() {",
						`    fmt.Println("hello")`,
						"}",
					},
				},
			},
		}

		/////////// * means the content is changed at this line
		//*1  type foo string  // this is a definition, not a statement
		// 2  j := 0
		//*3  i := 0
		// 4  for i < len(statements) {
		//*5
		//*6      // check it
		//*7      if statements[i].startLine <= lineNumber && statements[i].endLine >= lineNumber {
		//*8          stmt = statements[i]
		// 9          break
		// 10     }
		// 11
		// 12     fmt.Println(i)
		//*13     // increase i
		//*14     i++
		// 15 }
		// 16
		//*17 if err != nil { return err }
		//*18 if err != nil { return err
		// 19 }
		// 20 if err != nil {
		//*21     return err }
		//*22 var a = 1
		// 23 var b = 2
		//*24 type myStruct {} // this is a definition, not a statement
		// 25
		// 26 var c = 3
		//*27 func myFunc() { // note, function definition is not a statement
		//*28     fmt.Println("hello")
		//*29 }
		testSuites := []struct {
			stat  *statement
			state State
		}{
			{ // 4   for i < len(statements)
				stat:  &statement{Statement: &Statement{State: Original}, StmtExtent: &StmtExtent{startLine: 4, endLine: 15}},
				state: Original,
			},
			{ // 2   j := 0
				stat:  &statement{Statement: &Statement{State: Original}, StmtExtent: &StmtExtent{startLine: 2, endLine: 2}},
				state: Original,
			},
			{ //*3   i := 0
				stat:  &statement{Statement: &Statement{State: Original}, StmtExtent: &StmtExtent{startLine: 3, endLine: 3}},
				state: Changed,
			},
			{ //*8   stmt = statements[i]
				stat:  &statement{Statement: &Statement{State: Original}, StmtExtent: &StmtExtent{startLine: 8, endLine: 8}},
				state: Changed,
			},
			{ //*7   if statements[i].startLine <= lineNumber && statements[i].endLine >= lineNumber
				stat:  &statement{Statement: &Statement{State: Original}, StmtExtent: &StmtExtent{startLine: 7, endLine: 10}},
				state: Changed,
			},
			{ // 9   break
				stat:  &statement{Statement: &Statement{State: Original}, StmtExtent: &StmtExtent{startLine: 9, endLine: 9}},
				state: Original,
			},
			{ // 12  fmt.Println(i)
				stat:  &statement{Statement: &Statement{State: Original}, StmtExtent: &StmtExtent{startLine: 12, endLine: 12}},
				state: Original,
			},
			{ //*14  i++
				stat:  &statement{Statement: &Statement{State: Original}, StmtExtent: &StmtExtent{startLine: 14, endLine: 14}},
				state: Changed,
			},
			{ //*17  { return err }
				stat:  &statement{Statement: &Statement{State: Original}, StmtExtent: &StmtExtent{startLine: 17, endLine: 17, startCol: 17, endCol: 26}},
				state: Changed,
			},
			{ // 17  if err != nil
				stat:  &statement{Statement: &Statement{State: Original}, StmtExtent: &StmtExtent{startLine: 17, endLine: 17, startCol: 1, endCol: 28}},
				state: Original,
			},
			{ // 18   if err != nil {
				stat:  &statement{Statement: &Statement{State: Original}, StmtExtent: &StmtExtent{startLine: 18, endLine: 19, startCol: 1, endCol: 1}},
				state: Original,
			},
			{ //*18  return err
				stat:  &statement{Statement: &Statement{State: Original}, StmtExtent: &StmtExtent{startLine: 18, endLine: 18, startCol: 17, endCol: 26}},
				state: Changed,
			},
			{ // 20  if err != nil {
				stat:  &statement{Statement: &Statement{State: Original}, StmtExtent: &StmtExtent{startLine: 20, endLine: 21, startCol: 1, endCol: 16}},
				state: Original,
			},
			{ //*21    return err }
				stat:  &statement{Statement: &Statement{State: Original}, StmtExtent: &StmtExtent{startLine: 21, endLine: 21, startCol: 5, endCol: 14}},
				state: Changed,
			},
			{ //*22  var a = 1
				stat:  &statement{Statement: &Statement{State: Original}, StmtExtent: &StmtExtent{startLine: 22, endLine: 22}},
				state: Changed,
			},
			{ // 23   var b = 2
				stat:  &statement{Statement: &Statement{State: Original}, StmtExtent: &StmtExtent{startLine: 23, endLine: 23}},
				state: Original,
			},
			{ // 28   fmt.Println("hello")
				stat:  &statement{Statement: &Statement{State: Original}, StmtExtent: &StmtExtent{startLine: 28, endLine: 28}},
				state: Changed,
			},
			{ // 26   var c = 3
				stat:  &statement{Statement: &Statement{State: Original}, StmtExtent: &StmtExtent{startLine: 26, endLine: 26}},
				state: Original,
			},
		}

		var statements []*statement
		for _, item := range testSuites {
			statements = append(statements, item.stat)
		}
		parser.setStatementsState(change, statements)

		for i, item := range testSuites {
			if item.stat.State != item.state {
				t.Errorf("expect item %d state %s, but get %s", i+1, item.state, item.stat.State)
			}
		}

	})
}
