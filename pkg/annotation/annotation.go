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
	// when it's BLOCK_IGNORE, IgnoreBlocks contains the concrete ignore data.
	Type         IgnoreType
	Filename     string
	IgnoreBlocks map[cover.ProfileBlock]*IgnoreBlock
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
	s.Split(bufio.ScanLines)
	var fileLines []string
	for s.Scan() {
		fileLines = append(fileLines, s.Text())
	}

	profile := &IgnoreProfile{
		Type:         BLOCK_IGNORE,
		IgnoreBlocks: make(map[cover.ProfileBlock]*IgnoreBlock),
	}

	totalLines := len(fileLines)
	i := 0
	for i < totalLines {
		match := IgnoreRegexp.FindStringSubmatch(fileLines[i])
		// not match, continue next line
		if match == nil {
			i++
			continue
		}

		// match contains the result of the regexp on IgnoreRegexp
		ignoreKind := match[1]
		if ignoreKind == "file" { // set type to FILE_IGNORE and skip further processing
			profile.Type = FILE_IGNORE
			profile.IgnoreBlocks = nil
			break
		} else if ignoreKind == "block" { // block
			// ignoreOnBlock returns the endline of cover profile block
			// as index of fileLines starts from 0, the endline is actually the next index that waiting handling.
			i = ignoreOnBlock(fileLines, profile, coverProfile, i+1, fileLines[i])
		} else {
			//+gocover:ignore:block
			i++
		}
	}

	return profile, nil
}

// ignoreOnBlock finds the cover profile block that contains the ignore pattern text
// and returns the line number of the end line of cover profile block
func ignoreOnBlock(fileLines []string, profile *IgnoreProfile, coverProfile *cover.Profile, patternLineNumber int, patternText string) int {
	var profileBlock *cover.ProfileBlock

	// gocover ignore patterns are placed in block like following,
	// so the line number of it >= start line of code block and <= end line of code block
	// {  //+gocover:ignore:xxx
	//    //+gocover:ignore:xxx
	// }
	for _, b := range coverProfile.Blocks {
		if b.StartLine <= patternLineNumber && patternLineNumber < b.EndLine {
			profileBlock = &b
			break
		}
	}

	if profileBlock == nil {
		return patternLineNumber + 1
	}

	if _, ok := profile.IgnoreBlocks[*profileBlock]; !ok {
		ignoreBlock := &IgnoreBlock{Annotation: patternText}

		// Record the ignore code profile contents
		for i := profileBlock.StartLine; i <= profileBlock.EndLine; i++ {
			// as the source file of the scanner is same with cover profile,
			// so this method call always true.
			ignoreBlock.Lines = append(ignoreBlock.Lines, i)
			ignoreBlock.Contents = append(ignoreBlock.Contents, fileLines[i-1])
		}
		profile.IgnoreBlocks[*profileBlock] = ignoreBlock
	}

	return profileBlock.EndLine - 1
}
