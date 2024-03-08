package report

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/sirupsen/logrus"
)

type MDGen struct {
	// lexer for parsing go code
	lexer chroma.Lexer
	// style for go code snippets
	style *chroma.Style
	// outputPath report path
	outputPath string
	// reportName report name
	reportName string
	// logger
	logger  logrus.FieldLogger
	githash string
}

var _ ReportGenerator = (*MDGen)(nil)

// NewReportGenerator creates a html report generator to generate html coverage report.
// We will use https://pygments.org/docs/styles to style the output,
// and use // https://github.com/alecthomas/chroma to help to generate code snippets.
func NewMDReportGenerator(
	codeStyle string,
	outputPath string,
	reportName string,
	logger logrus.FieldLogger,
	gitHash string,
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

	return &MDGen{
		lexer:      lexer,
		style:      style,
		outputPath: outputPath,
		reportName: reportName,
		logger:     logger,
		githash:    gitHash,
	}
}

func (md *MDGen) GenerateReport(statistics *Statistics) error {

	report := []string{}

	violatedFiles := 0
	for _, profile := range statistics.CoverageProfile {
		if profile.CoveredLines == profile.TotalLines {
			continue
		}
		violatedFiles++
		report = append(report, "<details>\n")
		coveragePercent := (profile.CoveredLines / profile.TotalEffectiveLines) * 100
		circle := ":red_circle:"
		if coveragePercent > 50 {
			circle = ":orange_circle:"
		}
		if coveragePercent > 75 {
			circle = ":yellow_circle:"
		}
		report = append(report, fmt.Sprintf("<summary>%s %d%% %s</summary>\n", strings.Join(strings.Split(profile.FileName, "/")[3:], "/"), coveragePercent, circle))
		for _, section := range profile.ViolationSections {
			previousLineNumber := section.ViolationLines[0]
			uncoveredStart := previousLineNumber
			uncoveredEnd := previousLineNumber
			for _, lineNumber := range section.ViolationLines {
				if lineNumber-previousLineNumber > 1 {
					report = append(report, md.GenerateFileLinks(profile.FileName, uncoveredStart, uncoveredEnd))
					uncoveredStart = lineNumber
				}
				previousLineNumber = lineNumber
				uncoveredEnd = lineNumber
			}
			report = append(report, md.GenerateFileLinks(profile.FileName, uncoveredStart, uncoveredEnd))
		}
		report = append(report, "\n</details>\n")
	}

	reportFile := filepath.Join(md.outputPath, finalName(md.reportName))
	f, err := os.Create(reportFile)
	if err != nil {
		return fmt.Errorf("create report file: %w", err)
	}
	if violatedFiles == 0 {
		f.WriteString("#### :+1: Congrats! All the new codes are covered with tests! :green_circle:")
		f.Close()

		return nil
	}

	output := strings.Join(append([]string{"#### Missing coverage for file(s) below:\n"}, report...), "\n")
	if len(report) > violatedFiles+6 {
		output = strings.Join(strings.Split(output, "#L"), "?plain=1#L")
	}
	f.WriteString(output)
	f.Close()

	return nil
}

func (md *MDGen) GenerateFileLinks(filename string, startLineNum, endLineNum int) string {
	fn := strings.Split(filename, "/")
	link := fmt.Sprintf("https://%s/blob/%s/%s", strings.Join(fn[0:3], "/"), md.githash, strings.Join(fn[3:], "/"))
	snippet := fmt.Sprintf("#L%d", startLineNum)
	if startLineNum != endLineNum {
		snippet = fmt.Sprintf("%s-L%d", snippet, endLineNum)
	}

	return link + snippet
}
