package annotation

import (
	"bufio"
	"bytes"
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"
)

func TestIgnoreRegexp(t *testing.T) {
	t.Run("validate IgnoreRegexp", func(t *testing.T) {
		var testSuites = []struct {
			input  string
			expect []string
		}{
			{input: "//+gocover:ignore:all", expect: []string{"//+gocover:ignore:all", "all"}},
			{input: "// +gocover:ignore:all", expect: []string{"// +gocover:ignore:all", "all"}},
			{input: "    //+gocover:ignore:all", expect: []string{"    //+gocover:ignore:all", "all"}},
			{input: "//+gocover:ignore:block", expect: []string{"//+gocover:ignore:block", "block"}},
			{input: "// +gocover:ignore:block", expect: []string{"// +gocover:ignore:block", "block"}},
			{input: "    //+gocover:ignore:block", expect: []string{"    //+gocover:ignore:block", "block"}},
			{input: "  //  //+gocover:ignore:blcok", expect: []string{}},
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
		_, err := ParseIgnoreProfiles("/nonexist")
		if err == nil {
			t.Errorf("should return error, but return nil")
		}
	})

	t.Run("ignore all", func(t *testing.T) {
		dir := t.TempDir()
		f := filepath.Join(dir, "foo.go")
		lines := []string{
			`    //+gocover:ignore:all`,
			`    package foo`,
			`    func foo() {`,
			`         fmt.Println("foo")`,
			`    }`,
		}
		input := strings.Join(lines, "\n")
		ioutil.WriteFile(f, []byte(input), 0666)

		profile, err := ParseIgnoreProfiles(f)
		if err != nil {
			t.Errorf("should return nil, but get: %s", err)
		}

		if profile.Filename != f {
			t.Errorf("filename should %s, but get %s", f, profile.Filename)
		}
		if profile.Type != ALL_IGNORE {
			t.Errorf("type should %s, but %s", ALL_IGNORE, profile.Type)
		}
	})

	t.Run("ignore block", func(t *testing.T) {
		dir := t.TempDir()
		f := filepath.Join(dir, "foo.go")
		lines := []string{
			`    //+gocover:ignore:block`,
			`    package foo`,
			`    func foo() {`,
			`         fmt.Println("foo")`,
			`    }`,
		}
		input := strings.Join(lines, "\n")
		ioutil.WriteFile(f, []byte(input), 0666)

		profile, err := ParseIgnoreProfiles(f)
		if err != nil {
			t.Errorf("should return nil, but get: %s", err)
		}

		if profile.Filename != f {
			t.Errorf("filename should %s, but get %s", f, profile.Filename)
		}
		if profile.Type != BLOCK_IGNORE {
			t.Errorf("type should %s, but %s", BLOCK_IGNORE, profile.Type)
		}
		for idx := 0; idx < 4; idx++ {
			if _, ok := profile.Lines[idx+2]; !ok {
				t.Errorf("should contains %d", idx+2)
			}
		}
	})
}

func TestParseIgnoreProfilesFromReader(t *testing.T) {

	t.Run("ignore all", func(t *testing.T) {
		lines := []string{
			`    //+gocover:ignore:all`,
			`    a := "Hello world"`,
			`    fmt.Println(a)`,
			``,
			`    b := "Go"`,
			`    fmt.Println(b)`,
		}
		input := strings.Join(lines, "\n")
		r := bytes.NewReader([]byte(input))

		profile, err := parseIgnoreProfilesFromReader(r)
		if err != nil {
			t.Errorf("should not error, but %s", err)
		}
		if profile.Type != ALL_IGNORE {
			t.Errorf("type should %s, but %s", ALL_IGNORE, profile.Type)
		}
	})

	t.Run("ignore block", func(t *testing.T) {
		ignorePattern := `    //+gocover:ignore:block`
		lines := []string{
			ignorePattern,
			`    a := "Hello world"`,
			`    fmt.Println(a)`,
			``,
			`    b := "Go"`,
			`    fmt.Println(b)`,
		}
		input := strings.Join(lines, "\n")
		r := bytes.NewReader([]byte(input))

		profile, err := parseIgnoreProfilesFromReader(r)
		if err != nil {
			t.Errorf("should not error, but %s", err)
		}
		if profile.Type != BLOCK_IGNORE {
			t.Errorf("type should %s, but %s", BLOCK_IGNORE, profile.Type)
		}
		if len(profile.IgnoreBlocks) != 1 {
			t.Errorf("should have 1 ignore block, but get %d", len(profile.IgnoreBlocks))
		}

		block := profile.IgnoreBlocks[0]
		if block.Annotation != ignorePattern {
			t.Errorf("ignore pattern should be %s, but %s", ignorePattern, block.Annotation)
		}
		for idx := 0; idx < 2; idx++ {
			if block.Lines[idx] != idx+2 {
				t.Errorf("line %d should be ignored, but %d", idx+1, block.Lines[idx])
			}
			if block.Contents[idx] != lines[idx+1] {
				t.Errorf("line (%s) should be ignored, but (%s)", lines[idx], block.Contents[idx])
			}
		}
	})

	t.Run("ignore range", func(t *testing.T) {
		ignorePattern := `    //+gocover:ignore:4`
		lines := []string{
			ignorePattern,
			`    a := "Hello world"`,
			`    fmt.Println(a)`,
			``,
			`    b := "Go"`,
			`    fmt.Println(b)`,
		}
		input := strings.Join(lines, "\n")
		r := bytes.NewReader([]byte(input))

		profile, err := parseIgnoreProfilesFromReader(r)
		if err != nil {
			t.Errorf("should not error, but %s", err)
		}
		if profile.Type != BLOCK_IGNORE {
			t.Errorf("type should %s, but %s", BLOCK_IGNORE, profile.Type)
		}
		if len(profile.IgnoreBlocks) != 1 {
			t.Errorf("should have 1 ignore block, but get %d", len(profile.IgnoreBlocks))
		}

		block := profile.IgnoreBlocks[0]
		if block.Annotation != ignorePattern {
			t.Errorf("ignore pattern should be %s, but %s", ignorePattern, block.Annotation)
		}
		for idx := 0; idx < 4; idx++ {
			if block.Lines[idx] != idx+2 {
				t.Errorf("line %d should be ignored, but %d", idx+1, block.Lines[idx])
			}
			if block.Contents[idx] != lines[idx+1] {
				t.Errorf("line (%s) should be ignored, but (%s)", lines[idx], block.Contents[idx])
			}
		}
	})

	t.Run("ignore block and range", func(t *testing.T) {
		ignorePattern1 := `    //+gocover:ignore:block`
		ignorePattern2 := `    //+gocover:ignore:1`
		lines := []string{
			ignorePattern1,
			`    a := "Hello world"`,
			`    fmt.Println(a)`,
			``,
			ignorePattern2,
			`    invokeFoo(a)`,
			`    invokeBar(a)`,
		}
		input := strings.Join(lines, "\n")
		r := bytes.NewReader([]byte(input))

		profile, err := parseIgnoreProfilesFromReader(r)
		if err != nil {
			t.Errorf("should not error, but %s", err)
		}
		if profile.Type != BLOCK_IGNORE {
			t.Errorf("type should %s, but %s", BLOCK_IGNORE, profile.Type)
		}
		if len(profile.IgnoreBlocks) != 2 {
			t.Errorf("should have 2 ignore blocks, but get %d", len(profile.IgnoreBlocks))
		}

		block := profile.IgnoreBlocks[0]
		if block.Annotation != ignorePattern1 {
			t.Errorf("ignore pattern should be %s, but %s", ignorePattern1, block.Annotation)
		}
		for idx := 0; idx < 2; idx++ {
			if block.Lines[idx] != idx+2 {
				t.Errorf("line %d should be ignored, but %d", idx+1, block.Lines[idx])
			}
			if block.Contents[idx] != lines[idx+1] {
				t.Errorf("line (%s) should be ignored, but (%s)", lines[idx], block.Contents[idx])
			}
		}

		block = profile.IgnoreBlocks[1]
		if block.Annotation != ignorePattern2 {
			t.Errorf("ignore pattern should be %s, but %s", ignorePattern2, block.Annotation)
		}
		for idx := 0; idx < 1; idx++ {
			if block.Lines[idx] != idx+6 {
				t.Errorf("line %d should be ignored, but %d", idx+1, block.Lines[idx])
			}
			if block.Contents[idx] != lines[idx+5] {
				t.Errorf("line (%s) should be ignored, but (%s)", lines[idx], block.Contents[idx])
			}
		}
	})

	t.Run("ignore range and block", func(t *testing.T) {
		ignorePattern1 := `    //+gocover:ignore:2`
		ignorePattern2 := `    //+gocover:ignore:block`
		lines := []string{
			ignorePattern1,
			`    a := "Hello world"`,
			`    fmt.Println(a)`,
			``,
			ignorePattern2,
			`    invokeFoo(a)`,
			`    invokeBar(a)`,
		}
		input := strings.Join(lines, "\n")
		r := bytes.NewReader([]byte(input))

		profile, err := parseIgnoreProfilesFromReader(r)
		if err != nil {
			t.Errorf("should not error, but %s", err)
		}
		if profile.Type != BLOCK_IGNORE {
			t.Errorf("type should %s, but %s", BLOCK_IGNORE, profile.Type)
		}
		if len(profile.IgnoreBlocks) != 2 {
			t.Errorf("should have 2 ignore blocks, but get %d", len(profile.IgnoreBlocks))
		}

		block := profile.IgnoreBlocks[0]
		if block.Annotation != ignorePattern1 {
			t.Errorf("ignore pattern should be %s, but %s", ignorePattern1, block.Annotation)
		}
		for idx := 0; idx < 2; idx++ {
			if block.Lines[idx] != idx+2 {
				t.Errorf("line %d should be ignored, but %d", idx+1, block.Lines[idx])
			}
			if block.Contents[idx] != lines[idx+1] {
				t.Errorf("line (%s) should be ignored, but (%s)", lines[idx], block.Contents[idx])
			}
		}

		block = profile.IgnoreBlocks[1]
		if block.Annotation != ignorePattern2 {
			t.Errorf("ignore pattern should be %s, but %s", ignorePattern2, block.Annotation)
		}
		for idx := 0; idx < 2; idx++ {
			if block.Lines[idx] != idx+6 {
				t.Errorf("line %d should be ignored, but %d", idx+1, block.Lines[idx])
			}
			if block.Contents[idx] != lines[idx+5] {
				t.Errorf("line (%s) should be ignored, but (%s)", lines[idx], block.Contents[idx])
			}
		}
	})
}

func TestIgnoreOnblock(t *testing.T) {
	lines := []string{
		`    a := "Hello world"`,
		`    fmt.Println(a)`,
		``,
		`    b := "Go"`,
		`    fmt.Println(b)`,
	}
	input := strings.Join(lines, "\n")

	t.Run("input is not empty", func(t *testing.T) {
		scanner := bufio.NewScanner(bytes.NewReader([]byte(input)))

		profile := &IgnoreProfile{
			Lines: make(map[int]bool),
			Type:  BLOCK_IGNORE,
		}

		ignorePattern := "//+gocover:ignore:block"

		skipLines := ignoreOnBlock(scanner, profile, 0, ignorePattern)
		if skipLines != 3 {
			t.Errorf("should skip 3 lines, but get: %d", skipLines)
		}
		if len(profile.IgnoreBlocks) == 0 {
			t.Errorf("should have at least ignore blocks, but get 0")
		}

		block := profile.IgnoreBlocks[0]
		if block.Annotation != ignorePattern {
			t.Errorf("ignore pattern should be %s, but %s", ignorePattern, block.Annotation)
		}
		for idx := 0; idx < 2; idx++ {
			if block.Lines[idx] != idx+1 {
				t.Errorf("line %d should be ignored, but %d", idx+1, block.Lines[idx])
			}
			if block.Contents[idx] != lines[idx] {
				t.Errorf("line (%s) should be ignored, but (%s)", lines[idx], block.Contents[idx])
			}
		}

		for idx := 0; idx < 2; idx++ {
			if _, ok := profile.Lines[idx+1]; !ok {
				t.Errorf("should contains %d", idx+1)
			}
		}
	})

	t.Run("input is empty", func(t *testing.T) {
		scanner := bufio.NewScanner(bytes.NewReader([]byte(`  `)))

		profile := &IgnoreProfile{
			Lines: make(map[int]bool),
			Type:  BLOCK_IGNORE,
		}

		ignorePattern := "//+gocover:ignore:block"

		skipLines := ignoreOnBlock(scanner, profile, 0, ignorePattern)
		if skipLines != 1 {
			t.Errorf("should skip 1 line, but get: %d", skipLines)
		}
		if len(profile.IgnoreBlocks) != 0 {
			t.Errorf("should have no ignore blocks, but get %d", len(profile.IgnoreBlocks))
		}
	})
}

func TestIgnoreOnNumber(t *testing.T) {
	// input has 5 lines
	lines := []string{
		`    a := "Hello world"`,
		`    fmt.Println(a)`,
		``,
		`    b := "Go"`,
		`    fmt.Println(b)`,
	}
	input := strings.Join(lines, "\n")

	t.Run("when require skip number is less than total lines of input", func(t *testing.T) {
		scanner := bufio.NewScanner(bytes.NewReader([]byte(input)))

		profile := &IgnoreProfile{
			Lines: make(map[int]bool),
			Type:  BLOCK_IGNORE,
		}

		ignoreLines := 4
		ignorePattern := "//+gocover:ignore:4"
		skipLines := ignoreOnNumber(scanner, profile, 0, ignoreLines, ignorePattern)

		if skipLines != ignoreLines {
			t.Errorf("should ignore %d lines, but get: %d", ignoreLines, skipLines)
		}
		if len(profile.IgnoreBlocks) == 0 {
			t.Errorf("should have at least ignore blocks, but get 0")
		}

		block := profile.IgnoreBlocks[0]
		if block.Annotation != ignorePattern {
			t.Errorf("ignore pattern should be %s, but %s", ignorePattern, block.Annotation)
		}
		for idx := 0; idx < ignoreLines; idx++ {
			if block.Lines[idx] != idx+1 {
				t.Errorf("line %d should be ignored, but %d", idx+1, block.Lines[idx])
			}
			if block.Contents[idx] != lines[idx] {
				t.Errorf("line (%s) should be ignored, but (%s)", lines[idx], block.Contents[idx])
			}
		}

		for idx := 0; idx < 4; idx++ {
			if _, ok := profile.Lines[idx+1]; !ok {
				t.Errorf("should contains %d", idx+1)
			}
		}
	})

	t.Run("when require number is greater than total lines of input", func(t *testing.T) {
		scanner := bufio.NewScanner(bytes.NewReader([]byte(input)))

		profile := &IgnoreProfile{
			Lines: make(map[int]bool),
			Type:  BLOCK_IGNORE,
		}

		ignoreLines := 6
		ignorePattern := "//+gocover:ignore:6"
		skipLines := ignoreOnNumber(scanner, profile, 0, ignoreLines, ignorePattern)

		if skipLines != len(lines) {
			t.Errorf("should ignore %d lines, but get: %d", len(lines), skipLines)
		}
		if len(profile.IgnoreBlocks) == 0 {
			t.Errorf("should have at least ignore blocks, but get 0")
		}

		block := profile.IgnoreBlocks[0]
		if block.Annotation != ignorePattern {
			t.Errorf("ignore pattern should be %s, but %s", ignorePattern, block.Annotation)
		}
		for idx := 0; idx < len(lines); idx++ {
			if block.Lines[idx] != idx+1 {
				t.Errorf("line %d should be ignored, but %d", idx+1, block.Lines[idx])
			}
			if block.Contents[idx] != lines[idx] {
				t.Errorf("line (%s) should be ignored, but (%s)", lines[idx], block.Contents[idx])
			}
		}

		for idx := 0; idx < len(lines); idx++ {
			if _, ok := profile.Lines[idx+1]; !ok {
				t.Errorf("should contains %d", idx+1)
			}
		}
	})

	t.Run("when require number is 0", func(t *testing.T) {
		scanner := bufio.NewScanner(bytes.NewReader([]byte(input)))

		profile := &IgnoreProfile{
			Lines: make(map[int]bool),
			Type:  BLOCK_IGNORE,
		}

		ignoreLines := 0
		ignorePattern := "//+gocover:ignore:0"
		skipLines := ignoreOnNumber(scanner, profile, 0, ignoreLines, ignorePattern)
		if skipLines != 0 {
			t.Errorf("no line should be ignored, but get: %d", skipLines)
		}
		if len(profile.IgnoreBlocks) != 0 {
			t.Errorf("should have no ignore blocks, but get %d", len(profile.IgnoreBlocks))
		}
	})

	t.Run("input is empty", func(t *testing.T) {
		scanner := bufio.NewScanner(bytes.NewReader([]byte(``)))

		profile := &IgnoreProfile{
			Lines: make(map[int]bool),
			Type:  BLOCK_IGNORE,
		}

		ignoreLines := 4
		ignorePattern := "//+gocover:ignore:4"
		skipLines := ignoreOnNumber(scanner, profile, 0, ignoreLines, ignorePattern)

		if skipLines != 0 {
			t.Errorf("no line should be ignored, but get: %d", skipLines)
		}
		if len(profile.IgnoreBlocks) != 0 {
			t.Errorf("should have no ignore blocks, but get %d", len(profile.IgnoreBlocks))
		}
	})
}
