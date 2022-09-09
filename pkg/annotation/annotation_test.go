package annotation

import (
	"bytes"
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/tools/cover"
)

func TestIgnoreRegexp(t *testing.T) {
	t.Run("validate IgnoreRegexp", func(t *testing.T) {
		var testSuites = []struct {
			input  string
			expect []string
		}{
			{input: "//+gocover:ignore:file don't want test this block", expect: []string{"//+gocover:ignore:file don't want test this block", "file", " ", "don't want test this block"}},
			{input: "// +gocover:ignore:file don't want test this block", expect: []string{"// +gocover:ignore:file don't want test this block", "file", " ", "don't want test this block"}},
			{input: "//    +gocover:ignore:file don't want test this block", expect: []string{"//    +gocover:ignore:file don't want test this block", "file", " ", "don't want test this block"}},
			{input: "//	+gocover:ignore:file don't want test this block", expect: []string{"//	+gocover:ignore:file don't want test this block", "file", " ", "don't want test this block"}},
			{input: "//+gocover:ignore:file some comments", expect: []string{"//+gocover:ignore:file some comments", "file", " ", "some comments"}},
			{input: "    //+gocover:ignore:file some comments", expect: []string{"    //+gocover:ignore:file some comments", "file", " ", "some comments"}},
			{input: "	//+gocover:ignore:file some comments", expect: []string{"	//+gocover:ignore:file some comments", "file", " ", "some comments"}},
			{input: "//+gocover:ignore:block some comments", expect: []string{"//+gocover:ignore:block some comments", "block", " ", "some comments"}},
			{input: "    //+gocover:ignore:block some comments", expect: []string{"    //+gocover:ignore:block some comments", "block", " ", "some comments"}},
			{input: "	//+gocover:ignore:block some comments", expect: []string{"	//+gocover:ignore:block some comments", "block", " ", "some comments"}},
			{input: "  {  //+gocover:ignore:block some comments", expect: []string{"  {  //+gocover:ignore:block some comments", "block", " ", "some comments"}},
			{input: "  //  //+gocover:ignore:block some comments", expect: []string{"  //  //+gocover:ignore:block some comments", "block", " ", "some comments"}},
			{input: "//+gocover:ignore:file ", expect: []string{"//+gocover:ignore:file ", "file", " ", ""}},
			{input: "//+gocover:ignore:file  ", expect: []string{"//+gocover:ignore:file  ", "file", "  ", ""}},
			{input: "//+gocover:ignore:file", expect: []string{"//+gocover:ignore:file", "file", "", ""}},
			{input: "//+gocover:ignore:block", expect: []string{"//+gocover:ignore:block", "block", "", ""}},
			{input: "// +gocover:ignore:block", expect: []string{"// +gocover:ignore:block", "block", "", ""}},
			{input: "//+gocover:ignore:file: ", expect: []string{"//+gocover:ignore:file: ", "file", "", ": "}},
			{input: "//+gocover:ignore:file: comments", expect: []string{"//+gocover:ignore:file: comments", "file", "", ": comments"}},
			{input: "//+gocover:ignore:abc", expect: nil},
			{input: "//+gocover:ignore:123", expect: nil},
			{input: "//+gocover:ignore:", expect: nil},
		}

		for _, testSuite := range testSuites {
			match := IgnoreRegexp.FindStringSubmatch(testSuite.input)
			if len(match) != len(testSuite.expect) {
				t.Errorf("for input '%s', expect %d items, but get %d", testSuite.input, len(testSuite.expect), len(match))
			}
			n := len(match)
			for i := 0; i < n; i++ {
				if match[i] != testSuite.expect[i] {
					t.Errorf("for input '%s', expect item %d %s, but %s", testSuite.input, i, testSuite.expect[i], match[i])
				}
			}
		}
	})
}

func TestParseIgnoreAnnotation(t *testing.T) {
	t.Run("parse ignore annotation", func(t *testing.T) {
		testSuites := []struct {
			input    string
			kind     string
			comments string
			err      error
		}{
			{input: "//+gocover:ignore:file ignore this file!", kind: "file", comments: "ignore this file!", err: nil},
			{input: "//+gocover:ignore:file  ignore this file! ", kind: "file", comments: "ignore this file!", err: nil},
			{input: "{ //+gocover:ignore:block ignore this block!", kind: "block", comments: "ignore this block!", err: nil},
			{input: "{ //+gocover:ignore:block  ignore this block! ", kind: "block", comments: "ignore this block!", err: nil},
			{input: "//+gocover:ignore:abc ignore this block! ", kind: "", comments: "", err: nil},
			{input: "//+gocover:ignore:file  ", kind: "", comments: "", err: ErrCommentsRequired},
			{input: "//+gocover:ignore:file", kind: "", comments: "", err: ErrCommentsRequired},
			{input: "//+gocover:ignore:block  ", kind: "", comments: "", err: ErrCommentsRequired},
			{input: "//+gocover:ignore:block", kind: "", comments: "", err: ErrCommentsRequired},
			{input: "//+gocover:ignore:block:", kind: "", comments: "", err: ErrCommentsRequired},
		}

		for _, testCase := range testSuites {
			kind, comments, err := parseIgnoreAnnotation(testCase.input, 10)
			if kind != testCase.kind {
				t.Errorf("[%s] expect kind %s, but get %s", testCase.input, testCase.kind, kind)
			}
			if comments != testCase.comments {
				t.Errorf("[%s] expect comments %s, but get %s", testCase.input, testCase.comments, comments)
			}

			if testCase.err == nil && testCase.err != err {
				t.Errorf("[%s] expect error %s, but get %s", testCase.input, testCase.err, err)
			}
			if testCase.err != nil && err == nil {
				t.Errorf("[%s] expect error %s, but get %s", testCase.input, testCase.err, err)
			}
		}
	})
}

func TestParseIgnoreProfiles(t *testing.T) {

	t.Run("read file error", func(t *testing.T) {
		_, err := ParseIgnoreProfiles("/nonexist", nil)
		if err == nil {
			t.Errorf("should return error, but return nil")
		}
	})

	t.Run("ignore file", func(t *testing.T) {
		dir := t.TempDir()
		f := filepath.Join(dir, "foo.go")
		fileContents := `
		//+gocover:ignore:file
		package foo

		func foo() {}
	`
		ioutil.WriteFile(f, []byte(fileContents), 0666)

		_, err := ParseIgnoreProfiles(f, nil)
		if err == nil {
			t.Errorf("should return error, but return nil")
		}
	})

	t.Run("ignore file", func(t *testing.T) {
		dir := t.TempDir()
		f := filepath.Join(dir, "foo.go")
		fileContents := `
		//+gocover:ignore:file ignore this file!
		{
			//+gocover:ignore:block ignore this block!
			a := "Hello world"
			fmt.Println(a)
		}
	
		//+gocover:ignore:block ignore this block!
	
		if err != nil { //+gocover:ignore:block ignore this block!
			return err
		}
	`
		ioutil.WriteFile(f, []byte(fileContents), 0666)

		profile, err := ParseIgnoreProfiles(f, nil)
		if err != nil {
			t.Errorf("should return nil, but get: %s", err)
		}

		if profile.Filename != f {
			t.Errorf("filename should %s, but get %s", f, profile.Filename)
		}
		if profile.Type != FILE_IGNORE {
			t.Errorf("type should %s, but %s", FILE_IGNORE, profile.Type)
		}
		if profile.Comments != "ignore this file!" {
			t.Errorf("ignore comments should be %s, but %s", "ignore this file!", profile.Comments)
		}
	})

	t.Run("ignore block", func(t *testing.T) {
		dir := t.TempDir()
		f := filepath.Join(dir, "foo.go")
		fileContents := `
		{
			//+gocover:ignore:block ignore this block 1!
			a := "Hello world"
			fmt.Println(a)
		}
	
		//+gocover:ignore:block ignore this block 2!
	
		if err != nil { //+gocover:ignore:block ignore this block 3!
			return err
		}
	`
		fileLines := strings.Split(fileContents, "\n")

		coverProfile := &cover.Profile{
			Blocks: []cover.ProfileBlock{
				{StartLine: 2, EndLine: 6},
				{StartLine: 10, EndLine: 12},
			},
		}

		ioutil.WriteFile(f, []byte(fileContents), 0666)

		profile, err := ParseIgnoreProfiles(f, coverProfile)
		if err != nil {
			t.Errorf("should return nil, but get: %s", err)
		}

		if profile.Filename != f {
			t.Errorf("filename should %s, but get %s", f, profile.Filename)
		}
		if profile.Type != BLOCK_IGNORE {
			t.Errorf("type should %s, but %s", BLOCK_IGNORE, profile.Type)
		}

		if len(profile.IgnoreBlocks) != 2 {
			t.Errorf("should have 2 ignore blocks, but get %d", len(profile.IgnoreBlocks))
		}

		ignoreBlock1, ok := profile.IgnoreBlocks[coverProfile.Blocks[0]]
		if !ok {
			t.Errorf("should find first cover profile block")
		}
		if ignoreBlock1.Comments != "ignore this block 1!" {
			t.Errorf("ignore comments should be %s, but %s", "ignore this block 1!", ignoreBlock1.Comments)
		}
		for i := 0; i < len(ignoreBlock1.Lines); i++ {
			if ignoreBlock1.Lines[i] != i+2 {
				t.Errorf("line number should %d, but %d", i+1, ignoreBlock1.Lines[i])
			}
			if ignoreBlock1.Contents[i] != fileLines[i+1] {
				t.Errorf("content should %s, but %s", fileLines[i+1], ignoreBlock1.Contents[i])
			}
		}

		ignoreBlock2, ok := profile.IgnoreBlocks[coverProfile.Blocks[1]]
		if !ok {
			t.Errorf("should find second cover profile block")
		}
		if ignoreBlock2.Comments != "ignore this block 3!" {
			t.Errorf("ignore comments should be %s, but %s", "ignore this block 3!", ignoreBlock2.Comments)
		}
		for i := 0; i < len(ignoreBlock2.Lines); i++ {
			if ignoreBlock2.Lines[i] != i+10 {
				t.Errorf("line number should %d, but %d", i+9, ignoreBlock2.Lines[i])
			}
			if ignoreBlock2.Contents[i] != fileLines[i+9] {
				t.Errorf("content should %s, but %s", fileLines[i+9], ignoreBlock2.Contents[i])
			}
		}
	})
}

func TestParseIgnoreProfilesFromReader(t *testing.T) {

	t.Run("no comments ignore annotation", func(t *testing.T) {
		fileContents := `
		//+gocover:ignore:file 
		{
			//+gocover:ignore:block ignore this block!
			a := "Hello world"
			fmt.Println(a)
		}
	
		//+gocover:ignore:block ignore this block!
	
		if err != nil { //+gocover:ignore:block ignore this block!
			return err
		}
	`
		r := bytes.NewReader([]byte(fileContents))

		_, err := parseIgnoreProfilesFromReader(r, nil)
		if err == nil {
			t.Errorf("should error, but no error")
		}
	})

	t.Run("ignore file", func(t *testing.T) {
		fileContents := `
		//+gocover:ignore:file ignore this file!
		{
			//+gocover:ignore:block ignore this block!
			a := "Hello world"
			fmt.Println(a)
		}
	
		//+gocover:ignore:block ignore this block!
	
		if err != nil { //+gocover:ignore:block ignore this block!
			return err
		}
	`
		r := bytes.NewReader([]byte(fileContents))

		profile, err := parseIgnoreProfilesFromReader(r, nil)
		if err != nil {
			t.Errorf("should not error, but %s", err)
		}
		if profile.Type != FILE_IGNORE {
			t.Errorf("type should %s, but %s", FILE_IGNORE, profile.Type)
		}
	})

	t.Run("ignore block", func(t *testing.T) {
		fileContents := `
		{
			//+gocover:ignore:block ignore this block!
			a := "Hello world"
			fmt.Println(a)
		}
	
		//+gocover:ignore:block ignore this block!
	
		if err != nil { //+gocover:ignore:block ignore this block!
			return err
		}
	`

		fileLines := strings.Split(fileContents, "\n")

		coverProfile := &cover.Profile{
			Blocks: []cover.ProfileBlock{
				{StartLine: 2, EndLine: 6},
				{StartLine: 10, EndLine: 12},
			},
		}

		r := bytes.NewReader([]byte(fileContents))

		profile, err := parseIgnoreProfilesFromReader(r, coverProfile)
		if err != nil {
			t.Errorf("should not error, but %s", err)
		}
		if profile.Type != BLOCK_IGNORE {
			t.Errorf("type should %s, but %s", BLOCK_IGNORE, profile.Type)
		}
		if len(profile.IgnoreBlocks) != 2 {
			t.Errorf("should have 2 ignore blocks, but get %d", len(profile.IgnoreBlocks))
		}

		ignoreBlock1, ok := profile.IgnoreBlocks[coverProfile.Blocks[0]]
		if !ok {
			t.Errorf("should find first cover profile block")
		}
		for i := 0; i < len(ignoreBlock1.Lines); i++ {
			if ignoreBlock1.Lines[i] != i+2 {
				t.Errorf("line number should %d, but %d", i+1, ignoreBlock1.Lines[i])
			}
			if ignoreBlock1.Contents[i] != fileLines[i+1] {
				t.Errorf("content should %s, but %s", fileLines[i+1], ignoreBlock1.Contents[i])
			}
		}

		ignoreBlock2, ok := profile.IgnoreBlocks[coverProfile.Blocks[1]]
		if !ok {
			t.Errorf("should find second cover profile block")
		}
		for i := 0; i < len(ignoreBlock2.Lines); i++ {
			if ignoreBlock2.Lines[i] != i+10 {
				t.Errorf("line number should %d, but %d", i+9, ignoreBlock2.Lines[i])
			}
			if ignoreBlock2.Contents[i] != fileLines[i+9] {
				t.Errorf("content should %s, but %s", fileLines[i+9], ignoreBlock2.Contents[i])
			}
		}
	})

	t.Run("no cover profile block", func(t *testing.T) {
		fileContents := `
		{
			a := "Hello world"
			fmt.Println(a)
		}
	
		//+gocover:ignore:block ignore this block!
	
		if err != nil {
			return err
		}
	`

		coverProfile := &cover.Profile{
			Blocks: []cover.ProfileBlock{
				{StartLine: 2, EndLine: 6},
				{StartLine: 10, EndLine: 12},
			},
		}

		r := bytes.NewReader([]byte(fileContents))

		profile, err := parseIgnoreProfilesFromReader(r, coverProfile)
		if err != nil {
			t.Errorf("should not error, but %s", err)
		}
		if profile.Type != BLOCK_IGNORE {
			t.Errorf("type should %s, but %s", BLOCK_IGNORE, profile.Type)
		}
		if len(profile.IgnoreBlocks) != 0 {
			t.Errorf("should have no ignore block, but get %d", len(profile.IgnoreBlocks))
		}
	})

}

func TestIgnoreOnblock(t *testing.T) {
	fileContents := `
	{
		//+gocover:ignore:block ignore this block!
		a := "Hello world"
		fmt.Println(a)
	}

	//+gocover:ignore:block ignore this block!

	if err != nil { //+gocover:ignore:block ignore this block!
		return err
	}
`

	fileLines := strings.Split(fileContents, "\n")

	coverProfile := &cover.Profile{
		Blocks: []cover.ProfileBlock{
			{StartLine: 2, EndLine: 6},
			{StartLine: 10, EndLine: 12},
		},
	}

	t.Run("find cover profile", func(t *testing.T) {
		profile := &IgnoreProfile{
			Type:         BLOCK_IGNORE,
			IgnoreBlocks: make(map[cover.ProfileBlock]*IgnoreBlock),
		}

		// find first block
		nextUnProccessingLineNumber := ignoreOnBlock(fileLines, profile, coverProfile, 3, "	//+gocover:ignore:block ignore this block!", "ignore this block!")
		if profile.IgnoreBlocks == nil {
			t.Errorf("should find cover profile block")
		}
		if nextUnProccessingLineNumber != coverProfile.Blocks[0].EndLine-1 {
			t.Errorf("next unprocessing line number should %d, but %d", coverProfile.Blocks[0].EndLine-1, nextUnProccessingLineNumber)
		}

		ignoreBlock, ok := profile.IgnoreBlocks[coverProfile.Blocks[0]]
		if !ok {
			t.Errorf("should find first cover profile block")
		}

		for i := 0; i < len(ignoreBlock.Lines); i++ {
			if ignoreBlock.Lines[i] != i+2 {
				t.Errorf("line number should %d, but %d", i+1, ignoreBlock.Lines[i])
			}
			if ignoreBlock.Contents[i] != fileLines[i+1] {
				t.Errorf("content should %s, but %s", fileLines[i+1], ignoreBlock.Contents[i])
			}
		}
	})

	t.Run("find no cover profile", func(t *testing.T) {
		profile := &IgnoreProfile{
			Type:         BLOCK_IGNORE,
			IgnoreBlocks: make(map[cover.ProfileBlock]*IgnoreBlock),
		}

		nextUnProccessingLineNumber := ignoreOnBlock(fileLines, profile, coverProfile, 8, "//+gocover:ignore:block ignore this block!", "ignore this block!")
		if nextUnProccessingLineNumber != 9 {
			t.Errorf("next unprocessing line number should %d, but %d", 9, nextUnProccessingLineNumber)
		}
	})
}

// Test case for ignore block, example file and corresponding cover profile,
// the results should have 5 ignore blocks.
/** filename: github.com/Azure/gocover/pkg/report/util.go
package report

import "fmt"

var i, j int = 3, 3

func case1() { //+gocover:ignore:block ignore this block!
	var c, python, java = true, false, "no!"
	fmt.Println(i, j, c, python, java)
}

func case2(x int) { //+gocover:ignore:block ignore this block!
	var c, python, java = true, false, "yes!"
	if x > 0 {
		fmt.Println(i, j, c, python, java)
	}
	fmt.Println(i, j, c, python, java, x)
}

func case3(x int) { //+gocover:ignore:block ignore this block!
	var c, python, java = true, false, "yes!"
	if x > 0 { //+gocover:ignore:block ignore this block!
		fmt.Println(i, j, c, python, java)
	}
	fmt.Println(i, j, c, python, java, x)
}

func case4(x int) {
	var c, python, java = true, false, "yes!"
	if x > 0 { //+gocover:ignore:block ignore this block!
		fmt.Println(i, j, c, python, java)
	}
	fmt.Println(i, j, c, python, java, x)
}
**/

// github.com/Azure/gocover/pkg/report/util.go:7.14,10.2 2 0
// github.com/Azure/gocover/pkg/report/util.go:12.19,14.11 2 0
// github.com/Azure/gocover/pkg/report/util.go:17.2,17.39 1 0
// github.com/Azure/gocover/pkg/report/util.go:14.11,16.3 1 0
// github.com/Azure/gocover/pkg/report/util.go:20.19,22.11 2 0
// github.com/Azure/gocover/pkg/report/util.go:25.2,25.39 1 0
// github.com/Azure/gocover/pkg/report/util.go:22.11,24.3 1 0
// github.com/Azure/gocover/pkg/report/util.go:28.19,30.11 2 0
// github.com/Azure/gocover/pkg/report/util.go:33.2,33.39 1 0
// github.com/Azure/gocover/pkg/report/util.go:30.11,32.3 1 0

func TestIgnoreBlock(t *testing.T) {
	t.Run("", func(t *testing.T) {
		fileContents := `package report

import "fmt"

var i, j int = 3, 3

func case1() { //+gocover:ignore:block ignore this block!
	var c, python, java = true, false, "no!"
	fmt.Println(i, j, c, python, java)
}

func case2(x int) { //+gocover:ignore:block ignore this block!
	var c, python, java = true, false, "yes!"
	if x > 0 {
		fmt.Println(i, j, c, python, java)
	}
	fmt.Println(i, j, c, python, java, x)
}

func case3(x int) { //+gocover:ignore:block ignore this block!
	var c, python, java = true, false, "yes!"
	if x > 0 { //+gocover:ignore:block ignore this block!
		fmt.Println(i, j, c, python, java)
	}
	fmt.Println(i, j, c, python, java, x)
}

func case4(x int) {
	var c, python, java = true, false, "yes!"
	if x > 0 { //+gocover:ignore:block ignore this block!
		fmt.Println(i, j, c, python, java)
	}
	fmt.Println(i, j, c, python, java, x)
}
	`

		coverProfile := &cover.Profile{
			Blocks: []cover.ProfileBlock{
				{StartLine: 7, StartCol: 14, EndLine: 10, EndCol: 2, NumStmt: 2, Count: 0},
				{StartLine: 12, StartCol: 19, EndLine: 14, EndCol: 11, NumStmt: 2, Count: 0},
				{StartLine: 17, StartCol: 2, EndLine: 17, EndCol: 39, NumStmt: 1, Count: 0},
				{StartLine: 14, StartCol: 11, EndLine: 16, EndCol: 3, NumStmt: 1, Count: 0},
				{StartLine: 20, StartCol: 19, EndLine: 22, EndCol: 11, NumStmt: 2, Count: 0},
				{StartLine: 25, StartCol: 2, EndLine: 25, EndCol: 39, NumStmt: 1, Count: 0},
				{StartLine: 22, StartCol: 11, EndLine: 24, EndCol: 3, NumStmt: 1, Count: 0},
				{StartLine: 28, StartCol: 19, EndLine: 30, EndCol: 11, NumStmt: 2, Count: 0},
				{StartLine: 33, StartCol: 2, EndLine: 33, EndCol: 39, NumStmt: 1, Count: 0},
				{StartLine: 30, StartCol: 11, EndLine: 32, EndCol: 3, NumStmt: 1, Count: 0},
			},
		}

		r := bytes.NewReader([]byte(fileContents))

		profile, err := parseIgnoreProfilesFromReader(r, coverProfile)
		if err != nil {
			t.Errorf("should not error, but %s", err)
		}
		if profile.Type != BLOCK_IGNORE {
			t.Errorf("type should %s, but %s", BLOCK_IGNORE, profile.Type)
		}
		// results should have 5 ignore blocks
		if len(profile.IgnoreBlocks) != 5 {
			t.Errorf("should have 5 ignore blocks, but get %d", len(profile.IgnoreBlocks))
		}
	})
}
