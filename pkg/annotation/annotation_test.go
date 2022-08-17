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
			{input: "//+gocover:ignore:file:don't want test this block", expect: []string{"//+gocover:ignore:file:don't want test this block", "file", "don't want test this block"}},
			{input: "//+gocover:ignore:file:some comments", expect: []string{"//+gocover:ignore:file:some comments", "file", "some comments"}},
			{input: "    //+gocover:ignore:file:some comments", expect: []string{"    //+gocover:ignore:file:some comments", "file", "some comments"}},
			{input: "	//+gocover:ignore:file:some comments", expect: []string{"	//+gocover:ignore:file:some comments", "file", "some comments"}},
			{input: "//+gocover:ignore:block:some comments", expect: []string{"//+gocover:ignore:block:some comments", "block", "some comments"}},
			{input: "    //+gocover:ignore:block:some comments", expect: []string{"    //+gocover:ignore:block:some comments", "block", "some comments"}},
			{input: "	//+gocover:ignore:block:some comments", expect: []string{"	//+gocover:ignore:block:some comments", "block", "some comments"}},
			{input: "  {  //+gocover:ignore:block:some comments", expect: []string{"  {  //+gocover:ignore:block:some comments", "block", "some comments"}},
			{input: "  //  //+gocover:ignore:block:some comments", expect: []string{"  //  //+gocover:ignore:block:some comments", "block", "some comments"}},
			{input: "//+gocover:ignore:file: ", expect: nil},
			{input: "//+gocover:ignore:file: comments", expect: nil},
			{input: "//+gocover:ignore:block", expect: nil},
			{input: "// +gocover:ignore:block", expect: nil},
			{input: "// +gocover:ignore:file", expect: nil},
			{input: "//+gocover:ignore:abc", expect: nil},
			{input: "//+gocover:ignore:123", expect: nil},
			{input: "//+gocover:ignore:", expect: nil},
		}

		for _, testSuite := range testSuites {
			match := IgnoreRegexp.FindStringSubmatch(testSuite.input)
			if len(match) != len(testSuite.expect) {
				t.Errorf("expect %d items, but get %d", len(testSuite.expect), len(match))
			}
			n := len(match)
			for i := 0; i < n; i++ {
				if match[i] != testSuite.expect[i] {
					t.Errorf("expect item %d %s, but %s", i, testSuite.expect[i], match[i])
				}
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
		//+gocover:ignore:file:don't want test this file
		{
			//+gocover:ignore:block:won't reach here
			a := "Hello world"
			fmt.Println(a)
		}
	
		//+gocover:ignore:block:won't reach here
	
		if err != nil { //+gocover:ignore:block:won't reach here
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

	})

	t.Run("ignore block", func(t *testing.T) {
		dir := t.TempDir()
		f := filepath.Join(dir, "foo.go")
		fileContents := `
		{
			//+gocover:ignore:block:won't reach here
			a := "Hello world"
			fmt.Println(a)
		}
	
		//+gocover:ignore:block:won't reach here
	
		if err != nil { //+gocover:ignore:block:won't reach here
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
}

func TestParseIgnoreProfilesFromReader(t *testing.T) {

	t.Run("ignore file", func(t *testing.T) {
		fileContents := `
		//+gocover:ignore:file:don't want test this file
		{
			//+gocover:ignore:block:won't reach here
			a := "Hello world"
			fmt.Println(a)
		}
	
		//+gocover:ignore:block:won't reach here
	
		if err != nil { //+gocover:ignore:block:won't reach here
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
			//+gocover:ignore:block:won't reach here
			a := "Hello world"
			fmt.Println(a)
		}
	
		//+gocover:ignore:block:won't reach here
	
		if err != nil { //+gocover:ignore:block:won't reach here
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
	
		//+gocover:ignore:block:won't reach here
	
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
		//+gocover:ignore:block:won't reach here
		a := "Hello world"
		fmt.Println(a)
	}

	//+gocover:ignore:block:won't reach here

	if err != nil { //+gocover:ignore:block:won't reach here
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
		nextUnProccessingLineNumber := ignoreOnBlock(fileLines, profile, coverProfile, 3, "	//+gocover:ignore:block")
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

		nextUnProccessingLineNumber := ignoreOnBlock(fileLines, profile, coverProfile, 8, "//+gocover:ignore:block")
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

func case1() { //+gocover:ignore:block:won't reach here
	var c, python, java = true, false, "no!"
	fmt.Println(i, j, c, python, java)
}

func case2(x int) { //+gocover:ignore:block:won't reach here
	var c, python, java = true, false, "yes!"
	if x > 0 {
		fmt.Println(i, j, c, python, java)
	}
	fmt.Println(i, j, c, python, java, x)
}

func case3(x int) { //+gocover:ignore:block:won't reach here
	var c, python, java = true, false, "yes!"
	if x > 0 { //+gocover:ignore:block:won't reach here
		fmt.Println(i, j, c, python, java)
	}
	fmt.Println(i, j, c, python, java, x)
}

func case4(x int) {
	var c, python, java = true, false, "yes!"
	if x > 0 { //+gocover:ignore:block:won't reach here
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

func case1() { //+gocover:ignore:block:won't reach here
	var c, python, java = true, false, "no!"
	fmt.Println(i, j, c, python, java)
}

func case2(x int) { //+gocover:ignore:block:won't reach here
	var c, python, java = true, false, "yes!"
	if x > 0 {
		fmt.Println(i, j, c, python, java)
	}
	fmt.Println(i, j, c, python, java, x)
}

func case3(x int) { //+gocover:ignore:block:won't reach here
	var c, python, java = true, false, "yes!"
	if x > 0 { //+gocover:ignore:block:won't reach here
		fmt.Println(i, j, c, python, java)
	}
	fmt.Println(i, j, c, python, java, x)
}

func case4(x int) {
	var c, python, java = true, false, "yes!"
	if x > 0 { //+gocover:ignore:block:won't reach here
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
