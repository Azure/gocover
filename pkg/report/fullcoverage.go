package report

import (
	"bytes"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Azure/gocover/pkg/annotation"
	"golang.org/x/tools/cover"
)

type FullCoverage interface {
	BuildFullCoverageTree() ([]*AllInformation, error)
}

func NewFullCoverage(
	profiles []*cover.Profile,
	modulePath string,
	repositoryPath string,
	excludes []string,
	writer io.Writer,
) (FullCoverage, error) {

	var excludesRegexps []*regexp.Regexp
	for _, ignorePattern := range excludes {
		reg, err := regexp.Compile(ignorePattern)
		if err != nil {
			return nil, fmt.Errorf("compile pattern %s: %w", ignorePattern, err)
		}
		excludesRegexps = append(excludesRegexps, reg)
	}

	return &fullCoverage{
		profiles:        profiles,
		modulePath:      modulePath,
		repositoryPath:  repositoryPath,
		excludesRegexps: excludesRegexps,
		coverageTree: &coverageTree{
			ModuleHostPath: modulePath,
			Root:           NewTreeNode(modulePath, false),
		},
		writer: writer,
	}, nil
}

var _ FullCoverage = (*fullCoverage)(nil)

type fullCoverage struct {
	profiles        []*cover.Profile
	modulePath      string
	repositoryPath  string
	excludesRegexps []*regexp.Regexp
	coverageTree    CoverageTree
	writer          io.Writer
}

func (full *fullCoverage) BuildFullCoverageTree() ([]*AllInformation, error) {
	full.ignore()
	if err := full.covered(); err != nil {
		return nil, err
	}
	return full.coverageTree.All(), nil
}

func (full *fullCoverage) ignore() {
	var filteredProfiles []*cover.Profile

	for _, p := range full.profiles {
		filter := false
		for _, reg := range full.excludesRegexps {
			if reg.MatchString(p.FileName) {
				filter = true
				break
			}
		}
		if !filter {
			filteredProfiles = append(filteredProfiles, p)
		}
	}

	full.profiles = filteredProfiles
}

func (full *fullCoverage) covered() error {

	ignoreBuf := new(bytes.Buffer)
	for _, p := range full.profiles {

		goPath := filepath.Join(full.repositoryPath, strings.TrimPrefix(p.FileName, full.modulePath))
		ignoreProfile, err := annotation.ParseIgnoreProfiles(goPath, p)
		if err != nil {
			return err
		}

		buf := new(bytes.Buffer)
		node := full.coverageTree.FindOrCreate(p.FileName)
		for _, block := range p.Blocks {
			node.TotalLines += int64(block.NumStmt)

			if ignore, ok := ignoreProfile.IgnoreBlocks[block]; !ok {
				node.TotalEffectiveLines += int64(block.NumStmt)
				if block.Count > 0 {
					node.TotalCoveredLines += int64(block.NumStmt)
				}
			} else {
				for i := 0; i < len(ignore.Contents); i++ {
					buf.WriteString(fmt.Sprintf("%d %s\n", ignore.Lines[i], ignore.Contents[i]))
				}
				buf.WriteString("\n")

			}

		}

		node.TotalIgnoredLines = node.TotalLines - node.TotalEffectiveLines
		if buf.String() != "" {
			ignoreBuf.WriteString(fmt.Sprintf("%s\n", p.FileName))
			ignoreBuf.WriteString(fmt.Sprintf("%s\n", buf.String()))
		}
	}

	full.coverageTree.CollectCoverageData()

	if ignoreBuf.String() != "" {
		fmt.Fprintf(full.writer, "%s\n", "Ignore overview:")
		fmt.Fprintf(full.writer, "%s", ignoreBuf.String())
	}
	return nil
}
