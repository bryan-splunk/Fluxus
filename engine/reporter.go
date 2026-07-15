package engine

import (
	"bytes"
	_ "embed"
	"fmt"
	"strings"
	"text/template"
	"time"
)

//go:embed templates/preassessment.tmpl
var preAssessmentTmpl string

//go:embed templates/operational.tmpl
var operationalTmpl string

func reportFuncs() template.FuncMap {
	return template.FuncMap{
		"upper": func(v interface{}) string {
			return strings.ToUpper(fmt.Sprint(v))
		},
		"join": strings.Join,
		"not":  func(b bool) bool { return !b },
		"now":  func() string { return time.Now().Format("2006-01-02 15:04") },
	}
}

// RenderPreAssessment generates the PreAssessment.md content from a DryRunResult.
// The output is structured per-file to match the web UI tab layout and the
// AI Skill's per-file README convention.
// When includeGuidance is true each change card includes the rule's Guidance prose.
func RenderPreAssessment(result *DryRunResult, targetVersion string, files []string, includeGuidance bool) (string, error) {
	reportTemplate, err := template.New("pre").Funcs(reportFuncs()).Parse(preAssessmentTmpl)
	if err != nil {
		return "", fmt.Errorf("parsing pre-assessment template: %w", err)
	}

	perFile, globalConflicts, globalCounts := BuildPerFileAssessments(result, includeGuidance)

	data := ReportData{
		TargetVersion:   targetVersion,
		Files:           files,
		GlobalCounts:    globalCounts,
		GlobalConflicts: globalConflicts,
		PerFile:         perFile,
	}

	var buffer bytes.Buffer
	if err := reportTemplate.Execute(&buffer, data); err != nil {
		return "", fmt.Errorf("rendering pre-assessment: %w", err)
	}
	return buffer.String(), nil
}

// RenderOperationalAssessment generates the OperationalAssessment.md content.
func RenderOperationalAssessment(data OperationalReportData) (string, error) {
	reportTemplate, err := template.New("ops").Funcs(reportFuncs()).Parse(operationalTmpl)
	if err != nil {
		return "", fmt.Errorf("parsing operational template: %w", err)
	}
	var buffer bytes.Buffer
	if err := reportTemplate.Execute(&buffer, data); err != nil {
		return "", fmt.Errorf("rendering operational assessment: %w", err)
	}
	return buffer.String(), nil
}
