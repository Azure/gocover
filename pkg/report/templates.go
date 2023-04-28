package report

// htmlCoverageReport is the templates contents for html style coverage report.
var htmlCoverageReport = "" +
	`<!DOCTYPE html>
<html lang="en">

<head>
    <meta charset="utf-8">
    <title>
    {{ if IsFullCoverageReport .StatisticsType }}
        Full Coverage
    {{ end }}

    {{ if IsDiffCoverageReport .StatisticsType }}
        Diff Coverage
    {{ end }}
    </title>
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

        a {
            text-decoration: none;
        }
        a:hover {
            text-decoration: underline;
        }
        a:active {
            color: black;
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
                <b>Coverage</b>: {{ .TotalCoverageWithoutIgnore }}%
            </li>
            <li>
                <b>Coverage (with ignorance)</b>: {{ .TotalCoveragePercent }}%
            </li>
        </ul>

        <p>
            <b>Coverage </b> = Covered / Total <br />
            <b>Coverage (with ignorance) </b> = (Covered - CoveredButIngored) / Effective <br />
            <b>Total</b> = Effective + Ignored
        </p>

        <table border="1">
            <thead>
                <tr>
                    <th>Source File</th>
                    {{ if IsFullCoverageReport .StatisticsType }}
                        <th>Full Coverage (with ignorance) (%)</th>
                        <th>Full Coverage (%)</th>
                    {{ end }}
                    {{ if IsDiffCoverageReport .StatisticsType }}
                        <th>Diff Coverage (with ignorance) (%)</th>
                        <th>Diff Coverage (%)</th>
                    {{ end }}
                    <th>Covered Lines</th>
                    <th>Ignored Lines</th>
                    <th>Covered But Ignored Lines</th>
                    <th>Effective Lines</th>
                    <th>Total Lines</th>
                </tr>
            </thead>
            <tbody>
                {{ range .CoverageProfile }}
                <tr>
                    <td><a href="#{{.FileName}}">{{ .FileName }}</a></td>
                    <td>{{ PercentCovered .TotalEffectiveLines .CoveredLines .CoveredButIgnoredLines }}</td>
                    <td>{{ PercentCovered .TotalLines .CoveredLines 0 }}</td>
                    <td>{{ .CoveredLines }}</td>
                    <td>{{ .TotalIgnoredLines }}</td>
                    <td>{{ .CoveredButIgnoredLines }}</td>
                    <td>{{ .TotalEffectiveLines }}</td>
                    <td>{{ .TotalLines }}</td>
                </tr>
                {{ end }}
            </tbody>
        </table>

        {{ range .CoverageProfile }}
            <div class="src-snippet">
                {{ if lt (PercentCovered .TotalEffectiveLines .CoveredLines .CoveredButIgnoredLines) 100.0 }}
                <div class="src-name" id="{{.FileName}}">{{ .FileName }}</div>
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

    {{ if .ExcludeFiles }}
        <h3>Exclude Files</h3>
        <ul>
        {{ range .ExcludeFiles }}
            <li>{{ . }}</li>
        {{ end }}
        </ul>
    {{ end }}

</body>

</html>
`
