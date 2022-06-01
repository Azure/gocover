package report

// htmlCoverageReport is the templates contents for html style coverage report.
var htmlCoverageReport = `
<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="utf-8">
    <title>Diff Coverage</title>
    <style type="text/css">
      .src-snippet { margin-top: 2em; }
      .src-name { font-weight: bold;  }
      .snippets {
        border-top: 1px solid #bdbdbd;
        border-bottom: 1px solid #bdbdbd;
      }
    </style>
  </head>
  <body>
    <h1>Diff Coverage</h1>
    <p>Diff: {{ .ComparedBranch }}...HEAD</p>

    {{ if .CoverageProfile }}
      <ul>
        <li>
          <b>Total</b>: {{ NormalizeLines .TotalLines }} 
        </li>
        <li>
          <b>Missing</b>: {{ NormalizeLines .TotalViolationLines }}
        </li>
        <li>
          <b>Coverage</b>: {{ .TotalCoveragePercent }}%
        </li>
      </ul>
        <table border="1">
          <thead>
            <tr>
              <th>Source File</th>
              <th>Diff Coverage (%)</th>
              <th>Missing Lines</th>
            </tr>
          </thead>
          <tbody>
          {{ range .CoverageProfile }}
            <tr>
              <td>{{ .FileName }}</td>
              <td>{{ PercentCovered .TotalLines .CoveredLines }}</td>
              <td>{{ IntsJoin .TotalViolationLines }}</td>
            </tr>
          {{ end }}
          </tbody>
        </table>

        {{ range .CoverageProfile }}
          <div class="src-snippet">
          {{ if lt (PercentCovered .TotalLines .CoveredLines) 100.0 }}
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
