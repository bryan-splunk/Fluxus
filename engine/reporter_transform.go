package engine

import "sort"

// ============================================================
// Report view types
// ============================================================

// ReportData holds the data passed to the pre-assessment template.
type ReportData struct {
	TargetVersion   string
	Files           []string // used for How to Apply commands
	GlobalCounts    EffectCounts
	GlobalConflicts []ConflictView
	PerFile         []FileAssessment
}

// EffectCounts holds the per-priority effect totals for a scan.
// It is exported so it can be serialized as part of the /api/assess JSON response.
type EffectCounts struct {
	P1Active  int `json:"p1_active"`
	P1Comment int `json:"p1_comment"`
	P1Total   int `json:"p1_total"`
	P2Active  int `json:"p2_active"`
	P2Comment int `json:"p2_comment"`
	P2Total   int `json:"p2_total"`
	P3Active  int `json:"p3_active"`
	P3Comment int `json:"p3_comment"`
	P3Total   int `json:"p3_total"`
}

// EffectView is a flattened, JSON-serializable representation of one rule match.
// Prefix is "", "CC", or "F" matching the Skill's numbering convention (#N, #CCN, #FN).
// Guidance is only populated when the caller requests --include-guidance.
type EffectView struct {
	Num         int      `json:"num"`
	Prefix      string   `json:"prefix"`
	RuleID      string   `json:"rule_id"`
	Title       string   `json:"title"`
	Category    string   `json:"category"`
	Version     string   `json:"version"`
	Description string   `json:"description"`
	Guidance    string   `json:"guidance,omitempty"`
	Before      string   `json:"before,omitempty"`
	After       string   `json:"after,omitempty"`
	SeeAlso     []string `json:"see_also,omitempty"`
}

// FileAssessment holds the per-file pre-assessment data for a single config file.
type FileAssessment struct {
	FileName  string         `json:"file_name"`
	Counts    EffectCounts   `json:"counts"`
	Active    []EffectView   `json:"active"`    // #1, #2...
	Comments  []EffectView   `json:"comments"`  // #CC1, #CC2...
	Future    []EffectView   `json:"future"`    // #F1, #F2...
	Conflicts []ConflictView `json:"conflicts"` // conflicts affecting this file only
}

// ConflictView is a JSON-serializable representation of a detected rule conflict.
type ConflictView struct {
	Rule1ID  string `json:"rule1_id"`
	Rule2ID  string `json:"rule2_id"`
	Tick1    string `json:"tick1"`
	Tick2    string `json:"tick2"`
	FileName string `json:"file_name"`
	Message  string `json:"message"`
}

// OperationalReportData holds data for the operational assessment template.
type OperationalReportData struct {
	TargetVersion  string
	AppliedEffects []Effect
	GuidedEffects  []Effect
	// CommentEffects are rule matches against commented-out config that were
	// processed on the comment path (renames + injected upgrade-note annotations).
	CommentEffects  []Effect
	UpdatedFiles    map[string]string
	Warnings        []string
	TopologyIssues  []ValidationIssue
	IncludeGuidance bool
}

// ============================================================
// Transform functions
// ============================================================

// countEffects tallies P1/P2/P3 active and comment counts from a slice of effects.
func countEffects(effects []Effect) EffectCounts {
	var counts EffectCounts
	for _, effect := range effects {
		switch effect.Rule.Category {
		case CategoryP1:
			if effect.IsComment {
				counts.P1Comment++
			} else {
				counts.P1Active++
			}
		case CategoryP2:
			if effect.IsComment {
				counts.P2Comment++
			} else {
				counts.P2Active++
			}
		case CategoryP3:
			if effect.IsComment {
				counts.P3Comment++
			} else {
				counts.P3Active++
			}
		}
	}
	counts.P1Total = counts.P1Active + counts.P1Comment
	counts.P2Total = counts.P2Active + counts.P2Comment
	counts.P3Total = counts.P3Active + counts.P3Comment
	return counts
}

// BuildPerFileAssessments processes a DryRunResult into per-file structured data
// for the web front-end pre-assessment tab UI. Files are returned sorted
// alphabetically. Numbering mirrors the Skill's convention: active changes
// start at #1, commented-out at #CC1, future at #F1 — restarting per file.
// When includeGuidance is true each EffectView will carry the rule's Guidance field.
func BuildPerFileAssessments(result *DryRunResult, includeGuidance bool) ([]FileAssessment, []ConflictView, EffectCounts) {
	applicable := AllEffects(result.ApplicableTicks)
	SortEffects(applicable)
	future := AllEffects(result.FutureTicks)
	SortEffects(future)

	type fileData struct {
		active   []Effect
		comments []Effect
		future   []Effect
	}

	byFile := make(map[string]*fileData)
	ensureFile := func(name string) *fileData {
		if byFile[name] == nil {
			byFile[name] = &fileData{}
		}
		return byFile[name]
	}

	for _, e := range applicable {
		fd := ensureFile(e.FilePath)
		if e.IsComment {
			fd.comments = append(fd.comments, e)
		} else {
			fd.active = append(fd.active, e)
		}
	}
	for _, e := range future {
		ensureFile(e.FilePath).future = append(ensureFile(e.FilePath).future, e)
	}

	names := make([]string, 0, len(byFile))
	for name := range byFile {
		names = append(names, name)
	}
	sort.Strings(names)

	var globalCounts EffectCounts
	assessments := make([]FileAssessment, 0, len(names))
	for _, name := range names {
		fd := byFile[name]
		fa := FileAssessment{FileName: name}

		for i, e := range fd.active {
			fa.Active = append(fa.Active, effectToView(e, i+1, "", includeGuidance))
		}
		for i, e := range fd.comments {
			fa.Comments = append(fa.Comments, effectToView(e, i+1, "CC", includeGuidance))
		}
		for i, e := range fd.future {
			fa.Future = append(fa.Future, effectToView(e, i+1, "F", includeGuidance))
		}

		// Per-file counts cover applicable effects only (future is informational).
		applicable := append(fd.active, fd.comments...)
		fa.Counts = countEffects(applicable)

		globalCounts.P1Active += fa.Counts.P1Active
		globalCounts.P1Comment += fa.Counts.P1Comment
		globalCounts.P1Total += fa.Counts.P1Total
		globalCounts.P2Active += fa.Counts.P2Active
		globalCounts.P2Comment += fa.Counts.P2Comment
		globalCounts.P2Total += fa.Counts.P2Total
		globalCounts.P3Active += fa.Counts.P3Active
		globalCounts.P3Comment += fa.Counts.P3Comment
		globalCounts.P3Total += fa.Counts.P3Total

		assessments = append(assessments, fa)
	}

	// Build the global conflict list and index per-file conflicts.
	conflictsByFile := make(map[string][]ConflictView)
	globalConflicts := make([]ConflictView, 0, len(result.Conflicts))
	for _, c := range result.Conflicts {
		cv := ConflictView{
			Rule1ID:  c.Effect1.Rule.ID,
			Rule2ID:  c.Effect2.Rule.ID,
			Tick1:    c.Effect1.FiredAtTick,
			Tick2:    c.Effect2.FiredAtTick,
			FileName: c.Effect1.FilePath,
			Message:  c.Message,
		}
		globalConflicts = append(globalConflicts, cv)
		conflictsByFile[cv.FileName] = append(conflictsByFile[cv.FileName], cv)
	}

	// Attach per-file conflicts to each FileAssessment.
	for i := range assessments {
		assessments[i].Conflicts = conflictsByFile[assessments[i].FileName]
	}

	return assessments, globalConflicts, globalCounts
}

// effectToView flattens an Effect into an EffectView for JSON serialization.
// When includeGuidance is true the rule's Guidance prose is included in the view.
func effectToView(e Effect, num int, prefix string, includeGuidance bool) EffectView {
	ev := EffectView{
		Num:         num,
		Prefix:      prefix,
		RuleID:      e.Rule.ID,
		Title:       e.Rule.Title,
		Category:    string(e.Rule.Category),
		Version:     e.FiredAtTick,
		Description: e.Rule.Description,
		Before:      e.Rule.Migration.Before,
		After:       e.Rule.Migration.After,
		SeeAlso:     e.Rule.SeeAlso,
	}
	if includeGuidance {
		ev.Guidance = e.Rule.Guidance
	}
	return ev
}
