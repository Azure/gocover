package annotation

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strings"

	"golang.org/x/tools/cover"
)

var (
	// IgnoreRegexp the regexp for the gocover ignore pattern.
	// Two kinds of ignore pattern are supported:
	// - block
	// - file
	//
	// This regexp matches the lines that
	// starts with any characters, then follows `//+gocover:ignore:` and following either `file` or `block`,
	// then comments about the intention.
	IgnoreRegexp = regexp.MustCompile(`.*//\s*\+gocover:ignore:(file|block)(\s*)(.*)`)

	ErrCommentsRequired      = errors.New("comments required")
	ErrWrongAnnotationFormat = errors.New("wrong ignore annotation format")
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
	Comments     string // comments about file ignore
	Annotation   string // concrete ignore pattern
}

// IgnoreBlock represents a single block of ignore profiling data.
type IgnoreBlock struct {
	Annotation           string   // concrete ignore pattern
	AnnotationLineNumber int      // line number the ignore pattern locates at
	Contents             []string // ignore contents
	Lines                []int    // corresponding code line number of the ignore contents
	Comments             string   // comments about block ignore
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
	if err != nil {
		return nil, fmt.Errorf("%w in %s", err, fileName)
	}
	profile.Filename = fileName
	return profile, nil
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

	sort.Sort(blocksByStart(coverProfile.Blocks))

	totalLines := len(fileLines)
	i := 0
	for i < totalLines {
		ignoreKind, comments, err := parseIgnoreAnnotation(fileLines[i], i+1)
		if err != nil {
			return nil, err
		}

		// not match, continue next line
		if ignoreKind == "" {
			i++
			continue
		}

		if ignoreKind == "file" { // set type to FILE_IGNORE and skip further processing
			profile.Type = FILE_IGNORE
			profile.Annotation = fileLines[i]
			profile.Comments = comments
			profile.IgnoreBlocks = nil
			break
		} else if ignoreKind == "block" { // block
			// ignoreOnBlock returns the endline of cover profile block
			// as index of fileLines starts from 0, the endline is actually the next index that waiting handling.
			i = ignoreOnBlock(fileLines, profile, coverProfile, i+1, fileLines[i], comments)
		} else {
			//+gocover:ignore:block won't reach here
			i++
		}
	}

	return profile, nil
}

// ignoreOnBlock finds the cover profile block that contains the ignore pattern text
// and returns the line number of the end line of cover profile block.
func ignoreOnBlock(fileLines []string, profile *IgnoreProfile, coverProfile *cover.Profile, patternLineNumber int, patternText string, comments string) int {
	if len(coverProfile.Blocks) == 0 {
		return patternLineNumber + 1
	}

	idx := sort.Search(len(coverProfile.Blocks), func(i int) bool {
		return coverProfile.Blocks[i].StartLine > patternLineNumber
	})

	switch idx {
	case 0: // find no profile block, use the first one
	default:
		// Check the validate of the previous one, if annotation is among the previous one, use previous one,
		// otherwise, keep current one because it meets the following case
		// {
		//    {
		//	      fmt.Println(1)
		//    }
		//
		//    // annotation here, should return following profile block
		//    fmt.Println(2)
		// }
		if coverProfile.Blocks[idx-1].StartLine <= patternLineNumber && coverProfile.Blocks[idx-1].EndLine > patternLineNumber {
			idx--
			break
		}
		if idx == len(coverProfile.Blocks) {
			return patternLineNumber + 1
		}
	}

	profileBlock := &coverProfile.Blocks[idx]

	if _, ok := profile.IgnoreBlocks[*profileBlock]; !ok {
		ignoreBlock := &IgnoreBlock{Annotation: patternText, Comments: comments, AnnotationLineNumber: patternLineNumber}

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

func parseIgnoreAnnotation(line string, lineNumber int) (string, string, error) {
	match := IgnoreRegexp.FindStringSubmatch(line)
	// not match, continue next line
	if match == nil {
		return "", "", nil
	}

	kind := match[1]
	separator := match[2]
	comments := match[3]

	trimmedComments := strings.TrimSpace(comments)
	if trimmedComments == "" {
		return "", "", fmt.Errorf("%w for annotation '%s' at line %d", ErrCommentsRequired, line, lineNumber)
	}

	if separator == "" {
		return "", "", fmt.Errorf(
			"%w for annotation '%s' at line %d, use at least one space to seperate annotation and comments",
			ErrWrongAnnotationFormat, line, lineNumber,
		)
	}

	return kind, trimmedComments, nil
}

type blocksByStart []cover.ProfileBlock

func (b blocksByStart) Len() int      { return len(b) }
func (b blocksByStart) Swap(i, j int) { b[i], b[j] = b[j], b[i] }
func (b blocksByStart) Less(i, j int) bool {
	bi, bj := b[i], b[j]
	return bi.StartLine < bj.StartLine || bi.StartLine == bj.StartLine && bi.StartCol < bj.StartCol
}
