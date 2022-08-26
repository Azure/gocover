package report

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/sirupsen/logrus"
)

// ReportGenerator represents the feature that generate coverage report.
type ReportGenerator interface {
	GenerateReport(statistics *Statistics) error
}

// htmlReportGenerator implements a html style report generator.
type htmlReportGenerator struct {
	// lexer for parsing go code
	lexer chroma.Lexer
	// style for go code snippets
	style *chroma.Style
	// outputPath report path
	outputPath string
	// reportName report name
	reportName string
	// logger
	logger logrus.FieldLogger
}

var _ ReportGenerator = (*htmlReportGenerator)(nil)

const (
	// CodeLanguage represents the language style for report.
	CodeLanguage = "go"
	// codeHighlightColor background color for those uncovered code lines.
	codeHighlightColor = "bg:#ffcccc"
)

// NewReportGenerator creates a html report generator to generate html coverage report.
// We will use https://pygments.org/docs/styles to style the output,
// and use // https://github.com/alecthomas/chroma to help to generate code snippets.
func NewReportGenerator(
	codeStyle string,
	outputPath string,
	reportName string,
	logger logrus.FieldLogger,
) ReportGenerator {
	style := styles.Get(codeStyle)
	if style == nil {
		style = styles.Fallback
	}

	builder := style.Builder().Add(chroma.LineHighlight, codeHighlightColor)
	if s, err := builder.Build(); err == nil {
		style = s
	}

	lexer := lexers.Get(CodeLanguage)
	if lexer == nil {
		lexer = lexers.Fallback
	}

	return &htmlReportGenerator{
		lexer:      lexer,
		style:      style,
		outputPath: outputPath,
		reportName: reportName,
		logger:     logger,
	}
}

// GenerateReport process the diff coverage profile statistics and generate the final html report.
func (g *htmlReportGenerator) GenerateReport(statistics *Statistics) error {

	err := g.processCodeSnippets(statistics)
	if err != nil {
		return fmt.Errorf("process code snippets: %w", err)
	}

	reportFile := filepath.Join(g.outputPath, finalName(g.reportName))
	f, err := os.Create(reportFile)
	if err != nil {
		return fmt.Errorf("create report file: %w", err)
	}

	err = htmlCoverageReportTemplate.Execute(f, statistics)
	if err != nil {
		return fmt.Errorf("write report: %w", err)
	}

	g.logger.Debugf("generate html report to %s", reportFile)
	return nil
}

// processCodeSnippets process the violation sections and generate the corresponding go code snippets
// which shows the concrete code lines that violates the test coverage.
func (g *htmlReportGenerator) processCodeSnippets(statistics *Statistics) error {

	// each file has a coverage profile, and each coverage profile may have zero to many violation sections.
	for _, profile := range statistics.CoverageProfile {
		if profile.CoveredLines == profile.TotalLines {
			continue
		}

		// transform each violation sections to corresponding code snippets.
		for _, section := range profile.ViolationSections {
			iter, err := g.lexer.Tokenise(nil, strings.Join(section.Contents, "\n"))
			if err != nil {
				return fmt.Errorf("tokenise failed: %w", err)
			}

			var hlLines [][2]int
			for _, line := range section.ViolationLines {
				hlLines = append(hlLines, [2]int{line, line})
			}

			formatter := html.New(
				html.WithLineNumbers(true),
				html.LineNumbersInTable(true),
				html.BaseLineNumber(section.StartLine),
				html.LinkableLineNumbers(true, ""),
				html.HighlightLines(hlLines),
			)

			var buf bytes.Buffer
			err = formatter.Format(&buf, g.style, iter)
			if err != nil {
				return fmt.Errorf("format code snippet: %s", err)
			}

			profile.CodeSnippet = append(profile.CodeSnippet, template.HTML(buf.String()))
		}

	}

	return nil
}

func finalName(reportName string) string {
	return fmt.Sprintf("%s.html", reportName)
}

// htmlCoverageReportTemplate is the render engine for html coverage report.
var htmlCoverageReportTemplate = template.Must(
	template.New("htmlReportTemplate").
		Funcs(template.FuncMap{"IntsJoin": intsJoin}).
		Funcs(template.FuncMap{"NormalizeLines": normalizeLines}).
		Funcs(template.FuncMap{"PercentCovered": percentCovered}).
		Funcs(template.FuncMap{"IsFullCoverageReport": isFullCoverageReport}).
		Funcs(template.FuncMap{"IsDiffCoverageReport": isDiffCoverageReport}).
		Parse(htmlCoverageReport),
)

// intsJoin returns string that a int slice join with ,
func intsJoin(inputs []int) string {
	var s []string
	for _, i := range inputs {
		s = append(s, fmt.Sprintf("%d", i))
	}
	return strings.Join(s, ",")
}

// nmormalizeLines pluralize the noun if number is greater than one.
func normalizeLines(lines int) string {
	if lines < 2 {
		return fmt.Sprintf("%d line", lines)
	} else {
		return fmt.Sprintf("%d lines", lines)
	}
}

func percentCovered(total, covered int) float64 {
	var c float64
	// total is zero, no need to calculate
	if total == 0 {
		c = 100 // Avoid zero denominator.
	} else {
		c = float64(covered) / float64(total) * 100
	}
	percent, _ := strconv.ParseFloat(fmt.Sprintf("%.2f", c), 64)
	return percent
}

func isFullCoverageReport(statisticsType StatisticsType) bool {
	return statisticsType == FullStatisticsType
}

func isDiffCoverageReport(statisticsType StatisticsType) bool {
	return statisticsType == DiffStatisticsType
}
