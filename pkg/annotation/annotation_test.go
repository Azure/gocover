package annotation

// import (
// 	"bufio"
// 	"bytes"
// 	"io/ioutil"
// 	"path/filepath"
// 	"strings"
// 	"testing"

// 	"golang.org/x/tools/cover"
// )

// func TestIgnoreRegexp(t *testing.T) {
// 	t.Run("validate IgnoreRegexp", func(t *testing.T) {
// 		var testSuites = []struct {
// 			input  string
// 			expect []string
// 		}{
// 			{input: "//+gocover:ignore:file", expect: []string{"//+gocover:ignore:file", "file"}},
// 			{input: "    //+gocover:ignore:file", expect: []string{"    //+gocover:ignore:file", "file"}},
// 			{input: "	//+gocover:ignore:file", expect: []string{"	//+gocover:ignore:file", "file"}},
// 			{input: "//+gocover:ignore:block", expect: []string{"//+gocover:ignore:block", "block"}},
// 			{input: "    //+gocover:ignore:block", expect: []string{"    //+gocover:ignore:block", "block"}},
// 			{input: "	//+gocover:ignore:block", expect: []string{"	//+gocover:ignore:block", "block"}},
// 			{input: "  {  //+gocover:ignore:block", expect: []string{"  {  //+gocover:ignore:block", "block"}},
// 			{input: "  //  //+gocover:ignore:block", expect: []string{"  //  //+gocover:ignore:block", "block"}},
// 			{input: "// +gocover:ignore:block", expect: nil},
// 			{input: "// +gocover:ignore:file", expect: nil},
// 			{input: "//+gocover:ignore:abc", expect: nil},
// 			{input: "//+gocover:ignore:123", expect: nil},
// 			{input: "//+gocover:ignore:", expect: nil},
// 		}

// 		for _, testSuite := range testSuites {
// 			match := IgnoreRegexp.FindStringSubmatch(testSuite.input)
// 			if len(match) != len(testSuite.expect) {
// 				t.Errorf("expect %d items, but get %d", len(testSuite.expect), len(match))
// 			}
// 			n := len(match)
// 			for i := 0; i < n; i++ {
// 				if match[i] != testSuite.expect[i] {
// 					t.Errorf("expect item %d %s, but %s", i, testSuite.expect[i], match[i])
// 				}
// 			}
// 		}
// 	})
// }

// func TestParseIgnoreProfiles(t *testing.T) {
// 	t.Run("read file error", func(t *testing.T) {
// 		_, err := ParseIgnoreProfiles("/nonexist", nil)
// 		if err == nil {
// 			t.Errorf("should return error, but return nil")
// 		}
// 	})

// 	t.Run("ignore file", func(t *testing.T) {
// 		dir := t.TempDir()
// 		f := filepath.Join(dir, "foo.go")
// 		lines := []string{
// 			`    //+gocover:ignore:file`,
// 			`    package foo`,
// 			`    func foo() {`,
// 			`         fmt.Println("foo")`,
// 			`    }`,
// 		}
// 		input := strings.Join(lines, "\n")
// 		ioutil.WriteFile(f, []byte(input), 0666)

// 		profile, err := ParseIgnoreProfiles(f, nil)
// 		if err != nil {
// 			t.Errorf("should return nil, but get: %s", err)
// 		}

// 		if profile.Filename != f {
// 			t.Errorf("filename should %s, but get %s", f, profile.Filename)
// 		}
// 		if profile.Type != FILE_IGNORE {
// 			t.Errorf("type should %s, but %s", FILE_IGNORE, profile.Type)
// 		}
// 	})

// 	t.Run("ignore block", func(t *testing.T) {
// 		dir := t.TempDir()
// 		f := filepath.Join(dir, "foo.go")
// 		lines := []string{
// 			`    package foo`,
// 			`    //+gocover:ignore:block`,
// 			`    func foo() {`,
// 			`         fmt.Println("foo")`,
// 			`    }`,
// 		}
// 		coverProfile := &cover.Profile{
// 			Blocks: []cover.ProfileBlock{
// 				{StartLine: 3, EndLine: 5},
// 			},
// 		}

// 		input := strings.Join(lines, "\n")
// 		ioutil.WriteFile(f, []byte(input), 0666)

// 		profile, err := ParseIgnoreProfiles(f, coverProfile)
// 		if err != nil {
// 			t.Errorf("should return nil, but get: %s", err)
// 		}

// 		if profile.Filename != f {
// 			t.Errorf("filename should %s, but get %s", f, profile.Filename)
// 		}
// 		if profile.Type != BLOCK_IGNORE {
// 			t.Errorf("type should %s, but %s", BLOCK_IGNORE, profile.Type)
// 		}
// 		for _, v := range []int{3, 4, 5} {
// 			if _, ok := profile.Lines[v]; !ok {
// 				t.Errorf("should contains %d", v)
// 			}
// 		}
// 	})
// }

// func TestParseIgnoreProfilesFromReader(t *testing.T) {

// 	t.Run("ignore file", func(t *testing.T) {
// 		lines := []string{
// 			`    //+gocover:ignore:file`,
// 			`    a := "Hello world"`,
// 			`    fmt.Println(a)`,
// 			``,
// 			`    b := "Go"`,
// 			`    fmt.Println(b)`,
// 		}
// 		input := strings.Join(lines, "\n")
// 		r := bytes.NewReader([]byte(input))

// 		profile, err := parseIgnoreProfilesFromReader(r, nil)
// 		if err != nil {
// 			t.Errorf("should not error, but %s", err)
// 		}
// 		if profile.Type != FILE_IGNORE {
// 			t.Errorf("type should %s, but %s", FILE_IGNORE, profile.Type)
// 		}
// 	})

// 	t.Run("ignore block", func(t *testing.T) {
// 		lines := []string{
// 			`    //+gocover:ignore:block`,
// 			`    a := "Hello world"`,
// 			`    fmt.Println(a)`,
// 			``,
// 			`    b := "Go"`,
// 			`    fmt.Println(b)`,
// 		}

// 		coverProfile := &cover.Profile{
// 			Blocks: []cover.ProfileBlock{
// 				{StartLine: 2, EndLine: 3},
// 				{StartLine: 5, EndLine: 6},
// 			},
// 		}

// 		ignorePattern := lines[0]
// 		input := strings.Join(lines, "\n")
// 		r := bytes.NewReader([]byte(input))

// 		profile, err := parseIgnoreProfilesFromReader(r, coverProfile)
// 		if err != nil {
// 			t.Errorf("should not error, but %s", err)
// 		}
// 		if profile.Type != BLOCK_IGNORE {
// 			t.Errorf("type should %s, but %s", BLOCK_IGNORE, profile.Type)
// 		}
// 		if len(profile.IgnoreBlocks) != 1 {
// 			t.Errorf("should have 1 ignore block, but get %d", len(profile.IgnoreBlocks))
// 		}

// 		block := profile.IgnoreBlocks[0]
// 		if block.Annotation != ignorePattern {
// 			t.Errorf("ignore pattern should be %s, but %s", ignorePattern, block.Annotation)
// 		}
// 		for idx := 0; idx < len(block.Lines); idx++ {
// 			if block.Lines[idx] != idx+2 {
// 				t.Errorf("line %d should be ignored, but %d", idx+1, block.Lines[idx])
// 			}
// 			if block.Contents[idx] != lines[idx+1] {
// 				t.Errorf("line (%s) should be ignored, but (%s)", lines[idx], block.Contents[idx])
// 			}
// 		}

// 		for idx := 0; idx < len(profile.Lines); idx++ {
// 			if _, ok := profile.Lines[idx+2]; !ok {
// 				t.Errorf("should contains %d", idx+1)
// 			}
// 		}
// 	})

// 	// actufiley, this circumstance should not happen
// 	t.Run("no cover profile block", func(t *testing.T) {
// 		lines := []string{
// 			`    //+gocover:ignore:block`,
// 			`    a := "Hello world"`,
// 			`    fmt.Println(a)`,
// 			``,
// 			`    b := "Go"`,
// 			`    fmt.Println(b)`,
// 		}

// 		coverProfile := &cover.Profile{
// 			Blocks: []cover.ProfileBlock{
// 				{StartLine: 5, EndLine: 6},
// 			},
// 		}

// 		input := strings.Join(lines, "\n")
// 		r := bytes.NewReader([]byte(input))

// 		profile, err := parseIgnoreProfilesFromReader(r, coverProfile)
// 		if err != nil {
// 			t.Errorf("should not error, but %s", err)
// 		}
// 		if profile.Type != BLOCK_IGNORE {
// 			t.Errorf("type should %s, but %s", BLOCK_IGNORE, profile.Type)
// 		}
// 		if len(profile.IgnoreBlocks) != 0 {
// 			t.Errorf("should have no ignore block, but get %d", len(profile.IgnoreBlocks))
// 		}
// 	})

// 	t.Run("ignore blocks", func(t *testing.T) {
// 		lines := []string{
// 			`    //+gocover:ignore:block`,
// 			`    a := "Hello world"`,
// 			`    fmt.Println(a)`,
// 			``,
// 			`    //+gocover:ignore:block`,
// 			`    invokeFoo(a)`,
// 			`    invokeBar(a)`,
// 		}

// 		coverProfile := &cover.Profile{
// 			Blocks: []cover.ProfileBlock{
// 				{StartLine: 2, EndLine: 3},
// 				{StartLine: 6, EndLine: 7},
// 			},
// 		}

// 		ignorePattern1 := lines[0]
// 		ignorePattern2 := lines[4]
// 		input := strings.Join(lines, "\n")
// 		r := bytes.NewReader([]byte(input))

// 		profile, err := parseIgnoreProfilesFromReader(r, coverProfile)
// 		if err != nil {
// 			t.Errorf("should not error, but %s", err)
// 		}
// 		if profile.Type != BLOCK_IGNORE {
// 			t.Errorf("type should %s, but %s", BLOCK_IGNORE, profile.Type)
// 		}
// 		if len(profile.IgnoreBlocks) != len(coverProfile.Blocks) {
// 			t.Errorf("should have %d ignore blocks, but get %d", len(coverProfile.Blocks), len(profile.IgnoreBlocks))
// 		}

// 		block := profile.IgnoreBlocks[0]
// 		if block.Annotation != ignorePattern1 {
// 			t.Errorf("ignore pattern should be %s, but %s", ignorePattern1, block.Annotation)
// 		}
// 		for idx := 0; idx < len(block.Lines); idx++ {
// 			if block.Lines[idx] != idx+2 {
// 				t.Errorf("line %d should be ignored, but %d", idx+1, block.Lines[idx])
// 			}
// 			if block.Contents[idx] != lines[idx+1] {
// 				t.Errorf("line (%s) should be ignored, but (%s)", lines[idx], block.Contents[idx])
// 			}
// 		}

// 		block = profile.IgnoreBlocks[1]
// 		if block.Annotation != ignorePattern2 {
// 			t.Errorf("ignore pattern should be %s, but %s", ignorePattern2, block.Annotation)
// 		}
// 		for idx := 0; idx < len(block.Lines); idx++ {
// 			if block.Lines[idx] != idx+6 {
// 				t.Errorf("line %d should be ignored, but %d", idx+1, block.Lines[idx])
// 			}
// 			if block.Contents[idx] != lines[idx+5] {
// 				t.Errorf("line (%s) should be ignored, but (%s)", lines[idx], block.Contents[idx])
// 			}
// 		}

// 		for _, v := range []int{2, 3, 6, 7} {
// 			if _, ok := profile.Lines[v]; !ok {
// 				t.Errorf("should contains %d", v)
// 			}
// 		}
// 	})
// }

// func TestIgnoreOnblock(t *testing.T) {
// 	lines := []string{
// 		`    //+gocover:ignore:block`,
// 		`    a := "Hello world"`,
// 		`    fmt.Println(a)`,
// 		``,
// 		`    b := "Go"`,
// 		`    fmt.Println(b)`,
// 	}
// 	input := strings.Join(lines, "\n")

// 	coverProfile := &cover.Profile{
// 		Blocks: []cover.ProfileBlock{
// 			{StartLine: 2, EndLine: 3},
// 			{StartLine: 5, EndLine: 6},
// 		},
// 	}

// 	t.Run("find cover profile", func(t *testing.T) {
// 		scanner := bufio.NewScanner(bytes.NewReader([]byte(input)))

// 		profile := &IgnoreProfile{
// 			Lines: make(map[int]bool),
// 			Type:  BLOCK_IGNORE,
// 		}

// 		scanner.Scan()
// 		ignorePattern := scanner.Text()
// 		ignoreBlockLines := ignoreOnBlock(scanner, profile, coverProfile, 1, ignorePattern)
// 		b := coverProfile.Blocks[0]
// 		if ignoreBlockLines != (b.EndLine - b.StartLine + 1) {
// 			t.Errorf("ignore block shoud have %d lines, but get: %d", b.EndLine-b.StartLine+1, ignoreBlockLines)
// 		}
// 		if len(profile.IgnoreBlocks) == 0 {
// 			t.Errorf("should have at least ignore blocks, but get 0")
// 		}

// 		block := profile.IgnoreBlocks[0]
// 		if block.Annotation != ignorePattern {
// 			t.Errorf("ignore pattern should be %s, but %s", ignorePattern, block.Annotation)
// 		}
// 		for idx := 0; idx < len(block.Lines); idx++ {
// 			if block.Lines[idx] != idx+2 {
// 				t.Errorf("line %d should be ignored, but %d", idx+2, block.Lines[idx])
// 			}
// 			if block.Contents[idx] != lines[idx+1] {
// 				t.Errorf("line (%s) should be ignored, but (%s)", lines[idx], block.Contents[idx])
// 			}
// 		}

// 		for idx := 0; idx < len(profile.Lines); idx++ {
// 			if _, ok := profile.Lines[idx+2]; !ok {
// 				t.Errorf("should contains %d", idx+1)
// 			}
// 		}
// 	})

// 	t.Run("find no cover profile", func(t *testing.T) {
// 		scanner := bufio.NewScanner(bytes.NewReader([]byte(input)))

// 		profile := &IgnoreProfile{
// 			Lines: make(map[int]bool),
// 			Type:  BLOCK_IGNORE,
// 		}

// 		scanner.Scan()
// 		ignorePattern := scanner.Text()
// 		ignoreBlockLines := ignoreOnBlock(scanner, profile, coverProfile, 3, ignorePattern)
// 		if ignoreBlockLines != 0 {
// 			t.Errorf("ignore block shoud have %d lines, but get: %d", 0, ignoreBlockLines)
// 		}
// 		if len(profile.IgnoreBlocks) != 0 {
// 			t.Errorf("should have no ignore block, but get %d", len(profile.IgnoreBlocks))
// 		}
// 	})
// }
