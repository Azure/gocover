package annotation

import (
	"bufio"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
)

var (
	// IgnoreRegexp finds the gocover ignore pattern.
	// Three kind ignore pattern are supported:
	//   //+gocover:ignore:all
	//   //+gocover:ignore:block
	//   //+gocover:ignore:{number}
	//
	// - `//+gocover:ignore:all`
	//   will ignore the whole file, the file won't be used to calculate coverage.
	//
	// - `//+gocover:ignore:block`
	//   will ignore a code block, the code block won't be used to calculate coverage.
	//   code block is the code that have many non-blank lines until it meet a blank line.
	//   for example:
	//       pf, err := os.Open(fileName)  -|
	//       if err != nil {                |
	// 	         return nil, err            | -> code block
	//       }                              |
	//       defer pf.Close()              -|
	//
	//       profile, err := parseIgnoreProfilesFromReader(pf) -|
	//       profile.Filename = fileName                        | -> code block
	//       return profile, err                               -|
	//
	// - `//+gocover:ignore:{number}`
	//    will ignore the specific code lines, and the code won't be used to calculate coverage.
	//    for example:
	//       //+gocover:ignore:5
	//       pf, err := os.Open(fileName)  -|
	//       if err != nil {                |
	// 	         return nil, err            | -> these 5 lines will be ignored
	//       }                              |
	//       defer pf.Close()              -|
	//
	//       profile, err := parseIgnoreProfilesFromReader(pf) -|
	//       profile.Filename = fileName                        | -> code block
	//       return profile, err                               -|
	//
	// Question: how to ignore the whole go function?
	IgnoreRegexp = regexp.MustCompile(`^\s*//\s*\+gocover:ignore:([0-9A-Za-z]+)`)
)

// IgnoreType indicates the type of the ignore profile.
// - ALL_IGNORE means the profile ignore the whole input file.
// - BLOCK_IGNORE means the profile ignore several code block of the input file.
type IgnoreType string

const (
	ALL_IGNORE   IgnoreType = "all"
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

// ParseIgnoreProfiles parses ignore profile data in the specified file and returns a ignore profile.
func ParseIgnoreProfiles(fileName string) (*IgnoreProfile, error) {
	pf, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer pf.Close()

	profile, err := parseIgnoreProfilesFromReader(pf)
	profile.Filename = fileName
	return profile, err
}

// parseIgnoreProfilesFromReader parses ignore profile data from the Reader and returns a ignore profile.
func parseIgnoreProfilesFromReader(rd io.Reader) (*IgnoreProfile, error) {
	s := bufio.NewScanner(rd)
	lineNo := 0

	profile := &IgnoreProfile{
		Lines: make(map[int]bool),
		Type:  BLOCK_IGNORE,
	}

	for s.Scan() {
		lineNo++
		line := s.Text()
		match := IgnoreRegexp.FindStringSubmatch(line)

		// not match, continue next line
		if match == nil {
			continue
		}
		if len(match) < 2 {
			continue
		}

		// match contains the result of the regexp on IgnoreRegexp
		// match = ["//+gocover:ignore:all", "all"] when input is `//+gocover:ignore:all`,
		// match = ["//+gocover:ignore:block", "block"] when input is `//+gocover:ignore:block`,
		// match = ["//+gocover:ignore:10", "10"] when input is `//+gocover:ignore:10`,
		ignoreKind := match[1]
		if ignoreKind == "all" { // set type to ALL_IGNORE and skip further processing
			profile.Type = ALL_IGNORE
			break
		} else if ignoreKind == "block" { // block
			skipLineCnt := ignoreOnBlock(s, profile, lineNo, line)
			lineNo += skipLineCnt
		} else { // number
			// parse number fail means it's not the supported pattern, continue next line
			total, err := strconv.Atoi(ignoreKind)
			if err != nil {
				continue
			}
			skipLineCnt := ignoreOnNumber(s, profile, lineNo, total, line)
			lineNo += skipLineCnt
		}
	}

	return profile, nil
}

// ignoreOnBlock process pattern that ignore on code block.
// proccessing until a blank line.
func ignoreOnBlock(scanner *bufio.Scanner, profile *IgnoreProfile, startLine int, patternText string) int {
	block := &IgnoreBlock{Annotation: patternText}
	skipLines := 0
	total := 0

	var content string
	for scanner.Scan() {
		skipLines++
		content = scanner.Text()
		if strings.TrimSpace(content) == "" {
			break
		}

		startLine++
		total++
		block.Lines = append(block.Lines, startLine)
		block.Contents = append(block.Contents, content)
		profile.Lines[startLine] = true
	}

	if total != 0 {
		profile.IgnoreBlocks = append(profile.IgnoreBlocks, block)
	}
	return skipLines
}

// ignoreOnNumber process pattern that ignore specific lines.
func ignoreOnNumber(scanner *bufio.Scanner, profile *IgnoreProfile, startLine, cnt int, patternText string) int {
	if cnt <= 0 {
		return 0
	}

	block := &IgnoreBlock{Annotation: patternText}
	skipLines := 0

	var content string
	for cnt > 0 {
		if !scanner.Scan() {
			break
		}
		cnt--
		startLine++
		skipLines++
		content = scanner.Text()

		block.Lines = append(block.Lines, startLine)
		block.Contents = append(block.Contents, content)
		profile.Lines[startLine] = true
	}

	if skipLines != 0 {
		profile.IgnoreBlocks = append(profile.IgnoreBlocks, block)
	}
	return skipLines
}
