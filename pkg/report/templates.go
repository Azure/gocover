package report

// htmlCoverageReport is the templates contents for html style coverage report.
var htmlCoverageReport = "" +
	`<!DOCTYPE html>
<html lang="en">

<head>
    <meta charset="utf-8">
    <title>Diff Coverage</title>
    <style type="text/css">
        .src-snippet {
            margin-top: 2em;
        }

        .src-name {
            font-weight: bold;
        }

        .snippets {
            border-top: 1px solid #bdbdbd;
            border-bottom: 1px solid #bdbdbd;
        }
    </style>
</head>

<body>
    {{ if IsFullCoverageReport .StatisticsType }}
        <h1>Full Coverage</h1>
    {{ end }}

    {{ if IsDiffCoverageReport .StatisticsType }}
        <h1>Diff Coverage</h1>
        <p>Diff: {{ .ComparedBranch }}...HEAD</p>
    {{ end }}

    {{ if .CoverageProfile }}
    <ul>
        <li>
            <b>Total</b>: {{ NormalizeLines .TotalLines }}
        </li>
        <li>
            <b>Effective</b>: {{ NormalizeLines .TotalEffectiveLines }}
        </li>
        <li>
            <b>Covered</b>: {{ NormalizeLines .TotalCoveredLines }}
        </li>
        <li>
            <b>Ignored</b>: {{ NormalizeLines .TotalIgnoredLines }}
        </li>
        <li>
            <b>Coverage</b>: {{ .TotalCoveragePercent }}%
        </li>
    </ul>
    <table border="1">
        <thead>
            <tr>
                <th>Source File</th>
                {{ if IsFullCoverageReport .StatisticsType }}
                    <th>Full Coverage (%)</th>
                {{ end }}
                {{ if IsDiffCoverageReport .StatisticsType }}
                    <th>Diff Coverage (%)</th>
                {{ end }}
                <th>Covered Lines</th>
                <th>Ignored Lines</th>
                <th>Effective Lines</th>
                <th>Total Lines</th>
            </tr>
        </thead>
        <tbody>
            {{ range .CoverageProfile }}
            <tr>
                <td>{{ .FileName }}</td>
                <td>{{ PercentCovered .TotalEffectiveLines .CoveredLines }}</td>
                <td>{{ .CoveredLines }}</td>
                <td>{{ .TotalIgnoredLines }}</td>
                <td>{{ .TotalEffectiveLines }}</td>
                <td>{{ .TotalLines }}</td>
            </tr>
            {{ end }}
        </tbody>
    </table>

    {{ range .CoverageProfile }}
    <div class="src-snippet">
        {{ if lt (PercentCovered .TotalEffectiveLines .CoveredLines) 100.0 }}
        <div class="src-name">{{ .FileName }}</div>
        <div class="snippets">
            {{range .CodeSnippet}}
            {{ . }}
            {{ end }}
        </div>
        {{ end }}
    </div>
    {{ end }}

    {{ else }}
    <p>No lines with coverage information in this diff.</p>
    {{ end }}

</body>

</html>
`
