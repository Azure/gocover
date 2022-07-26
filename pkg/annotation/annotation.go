package annotation

import (
	"bufio"
	"io"
	"os"
	"regexp"

	"golang.org/x/tools/cover"
)

var (
	// IgnoreRegexp the regexp for the gocover ignore pattern.
	// Two kinds of ignore pattern are supported:
	// - block
	// - file
	//
	// This regexp matches the lines that
	// starts with any characters, then follows `//`, and `+gocover:ignore:` then following either `file` or `block`.
	IgnoreRegexp = regexp.MustCompile(`.*//\+gocover:ignore:(file|block)`)
)

// IgnoreType indicates the type of the ignore profile.
// - FILE_IGNORE means the profile ignore the whole input file.
// - BLOCK_IGNORE means the profile ignore several code block of the input file.
type IgnoreType string

const (
	FILE_IGNORE  IgnoreType = "file"
	BLOCK_IGNORE IgnoreType = "block"
)

// IgnoreProfile represents the ignore profiling data for a specific file.
type IgnoreProfile struct {
	// type of the ignore profile.
	// when it's BLOCK_IGNORE, Lines includes all the code line number that should be ignored,
	// IgnoreBlocks contains the concrete ignore data.
	Type         IgnoreType
	Filename     string
	Lines        map[int]bool
	IgnoreBlocks []*IgnoreBlock
}

// IgnoreBlock represents a single block of ignore profiling data.
type IgnoreBlock struct {
	Annotation string   // concrete ignore pattern
	Contents   []string // ignore contents
	Lines      []int    // corresponding code line number of the ignore contents
}

// ParseIgnoreProfiles parses ignore profile data in the specified file with the help of go unit test cover profile,
// and returns a ignore profile. The ProfileBlock in the cover profile is already sorted.
func ParseIgnoreProfiles(fileName string, coverProfile *cover.Profile) (*IgnoreProfile, error) {
	pf, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer pf.Close()

	profile, err := parseIgnoreProfilesFromReader(pf, coverProfile)
	profile.Filename = fileName
	return profile, err
}

// parseIgnoreProfilesFromReader parses ignore profile data from the Reader and returns a ignore profile.
func parseIgnoreProfilesFromReader(rd io.Reader, coverProfile *cover.Profile) (*IgnoreProfile, error) {
	s := bufio.NewScanner(rd)
	lineNumber := 0

	profile := &IgnoreProfile{
		Lines: make(map[int]bool),
		Type:  BLOCK_IGNORE,
	}

	for s.Scan() {
		lineNumber++
		line := s.Text()
		match := IgnoreRegexp.FindStringSubmatch(line)

		// not match, continue next line
		if match == nil {
			continue
		}

		// match contains the result of the regexp on IgnoreRegexp
		ignoreKind := match[1]
		if ignoreKind == "file" { // set type to FILE_IGNORE and skip further processing
			profile.Type = FILE_IGNORE
			break
		} else if ignoreKind == "block" { // block
			ignoreBlockLineCnt := ignoreOnBlock(s, profile, coverProfile, lineNumber, line)
			lineNumber += ignoreBlockLineCnt
		} else {
			//+gocover:ignore:block
			// actually, here won't happen
		}
	}

	return profile, nil
}

// gocover ignore patterns are placed in block like following,
// so the line number of it >= start line of code block and less than end line of code block
// {  //+gocover:ignore:xxx
//    //+gocover:ignore:xxx
// }
func ignoreOnBlock(scanner *bufio.Scanner, profile *IgnoreProfile, coverProfile *cover.Profile, patternLineNumber int, patternText string) int {
	var profileBlock *cover.ProfileBlock
	// startLine := patternLineNumber + 1
	// `startLine` is the line number after the annotation line.
	// Use the `startLine` to find the Profile Block.
	// Because the two profile blocks may have the same value on startline and endline,
	// which means that the finding process uses the condition the `startLine` equals to the start line of the block
	// and less or equal to the end line of the block to find the suitable block.
	for _, b := range coverProfile.Blocks {
		if b.StartLine <= patternLineNumber && patternLineNumber < b.EndLine {
			profileBlock = &b
			break
		}
	}

	m := make(map[*cover.ProfileBlock]bool)

	if profileBlock == nil {
		return 0
	}

	ignoreBlock := &IgnoreBlock{Annotation: patternText}

	// Record the ignore code profile contents and its corresponding line number.
	var content string
	for i := profileBlock.StartLine; i <= profileBlock.EndLine; i++ {
		// as the source file of the scanner is same with cover profile,
		// so this method call always true.
		scanner.Scan()
		content = scanner.Text()

		ignoreBlock.Lines = append(ignoreBlock.Lines, i)
		ignoreBlock.Contents = append(ignoreBlock.Contents, content)
		profile.Lines[i] = true
	}

	profile.IgnoreBlocks = append(profile.IgnoreBlocks, ignoreBlock)
	return profileBlock.EndLine - profileBlock.StartLine + 1
}
