package engine

import "testing"

// ============================================================
// Test helpers
// ============================================================

func makeRule(id string, category Category, guidance string) *Rule {
	return &Rule{
		ID:          id,
		Category:    category,
		Title:       "Test title for " + id,
		Description: "Test description for " + id,
		Guidance:    guidance,
		Migration: Migration{
			Before: "before: old",
			After:  "after: new",
		},
		SeeAlso: []string{"https://example.com"},
	}
}

func makeEffect(rule *Rule, filePath string, isComment bool, tick string) Effect {
	return Effect{
		Rule:        rule,
		FiredAtTick: tick,
		IsComment:   isComment,
		FilePath:    filePath,
	}
}

// makeDryRunResult wraps raw effect slices into the DryRunResult structure
// expected by BuildPerFileAssessments.
func makeDryRunResult(applicable []Effect, future []Effect, conflicts []Conflict) *DryRunResult {
	var applicableTicks []TickResult
	if len(applicable) > 0 {
		applicableTicks = []TickResult{{Version: "0.10", Effects: applicable}}
	}
	var futureTicks []TickResult
	if len(future) > 0 {
		futureTicks = []TickResult{{Version: "0.20", Effects: future}}
	}
	return &DryRunResult{
		ApplicableTicks: applicableTicks,
		FutureTicks:     futureTicks,
		Conflicts:       conflicts,
	}
}

// ============================================================
// countEffects tests
// ============================================================

func TestCountEffects_EmptyInput(t *testing.T) {
	counts := countEffects(nil)
	if counts.P1Active != 0 || counts.P1Comment != 0 || counts.P1Total != 0 {
		t.Errorf("expected all-zero counts for nil input, got %+v", counts)
	}
}

func TestCountEffects_P1P2P3(t *testing.T) {
	effects := []Effect{
		makeEffect(makeRule("P1-01", CategoryP1, ""), "a.yaml", false, "0.10"), // P1 active
		makeEffect(makeRule("P1-02", CategoryP1, ""), "a.yaml", true, "0.10"),  // P1 comment
		makeEffect(makeRule("P2-01", CategoryP2, ""), "a.yaml", false, "0.11"), // P2 active
		makeEffect(makeRule("P3-01", CategoryP3, ""), "a.yaml", true, "0.12"),  // P3 comment
	}
	c := countEffects(effects)

	if c.P1Active != 1 {
		t.Errorf("P1Active: want 1, got %d", c.P1Active)
	}
	if c.P1Comment != 1 {
		t.Errorf("P1Comment: want 1, got %d", c.P1Comment)
	}
	if c.P1Total != 2 {
		t.Errorf("P1Total: want 2, got %d", c.P1Total)
	}
	if c.P2Active != 1 {
		t.Errorf("P2Active: want 1, got %d", c.P2Active)
	}
	if c.P2Comment != 0 {
		t.Errorf("P2Comment: want 0, got %d", c.P2Comment)
	}
	if c.P2Total != 1 {
		t.Errorf("P2Total: want 1, got %d", c.P2Total)
	}
	if c.P3Active != 0 {
		t.Errorf("P3Active: want 0, got %d", c.P3Active)
	}
	if c.P3Comment != 1 {
		t.Errorf("P3Comment: want 1, got %d", c.P3Comment)
	}
	if c.P3Total != 1 {
		t.Errorf("P3Total: want 1, got %d", c.P3Total)
	}
}

// ============================================================
// effectToView tests
// ============================================================

func TestEffectToView_Fields(t *testing.T) {
	rule := makeRule("TEST-01", CategoryP1, "some guidance")
	e := makeEffect(rule, "cfg.yaml", false, "0.129")
	ev := effectToView(e, 5, "", false)

	if ev.Num != 5 {
		t.Errorf("Num: want 5, got %d", ev.Num)
	}
	if ev.RuleID != "TEST-01" {
		t.Errorf("RuleID: want TEST-01, got %q", ev.RuleID)
	}
	if ev.Title != "Test title for TEST-01" {
		t.Errorf("Title: want 'Test title for TEST-01', got %q", ev.Title)
	}
	if ev.Category != "p1" {
		t.Errorf("Category: want p1, got %q", ev.Category)
	}
	if ev.Version != "0.129" {
		t.Errorf("Version: want 0.129, got %q", ev.Version)
	}
	if ev.Before != "before: old" {
		t.Errorf("Before: want 'before: old', got %q", ev.Before)
	}
	if ev.After != "after: new" {
		t.Errorf("After: want 'after: new', got %q", ev.After)
	}
	if len(ev.SeeAlso) != 1 || ev.SeeAlso[0] != "https://example.com" {
		t.Errorf("SeeAlso: want [https://example.com], got %v", ev.SeeAlso)
	}
}

func TestEffectToView_GuidanceGated(t *testing.T) {
	rule := makeRule("TEST-02", CategoryP2, "important guidance text")
	e := makeEffect(rule, "cfg.yaml", false, "0.130")

	// Without flag: Guidance must be empty.
	evNo := effectToView(e, 1, "", false)
	if evNo.Guidance != "" {
		t.Errorf("expected empty Guidance when includeGuidance=false, got %q", evNo.Guidance)
	}

	// With flag: Guidance must be populated.
	evYes := effectToView(e, 1, "", true)
	if evYes.Guidance != "important guidance text" {
		t.Errorf("expected guidance text when includeGuidance=true, got %q", evYes.Guidance)
	}
}

// ============================================================
// BuildPerFileAssessments tests
// ============================================================

func TestBuildPerFileAssessments_Grouping(t *testing.T) {
	rA := makeRule("P1-01", CategoryP1, "")
	rB := makeRule("P2-01", CategoryP2, "")
	applicable := []Effect{
		makeEffect(rA, "alpha.yaml", false, "0.10"),
		makeEffect(rB, "beta.yaml", false, "0.10"),
	}
	result := makeDryRunResult(applicable, nil, nil)

	assessments, _, _ := BuildPerFileAssessments(result, false)

	if len(assessments) != 2 {
		t.Fatalf("expected 2 file assessments, got %d", len(assessments))
	}
	// Files are returned sorted alphabetically.
	if assessments[0].FileName != "alpha.yaml" {
		t.Errorf("expected alpha.yaml first, got %q", assessments[0].FileName)
	}
	if len(assessments[0].Active) != 1 || assessments[0].Active[0].RuleID != "P1-01" {
		t.Errorf("alpha.yaml should have exactly P1-01 as its active effect")
	}
	if len(assessments[1].Active) != 1 || assessments[1].Active[0].RuleID != "P2-01" {
		t.Errorf("beta.yaml should have exactly P2-01 as its active effect")
	}
}

func TestBuildPerFileAssessments_Numbering(t *testing.T) {
	// Two active + one comment (applicable) and one future — all for the same file.
	r1 := makeRule("P1-01", CategoryP1, "")
	r2 := makeRule("P2-01", CategoryP2, "")
	r3 := makeRule("P2-02", CategoryP2, "")
	r4 := makeRule("P3-01", CategoryP3, "")
	applicable := []Effect{
		makeEffect(r1, "cfg.yaml", false, "0.10"), // active #1
		makeEffect(r2, "cfg.yaml", false, "0.10"), // active #2
		makeEffect(r3, "cfg.yaml", true, "0.10"),  // comment #CC1
	}
	future := []Effect{
		makeEffect(r4, "cfg.yaml", false, "0.20"), // future #F1
	}
	result := makeDryRunResult(applicable, future, nil)

	assessments, _, _ := BuildPerFileAssessments(result, false)

	if len(assessments) != 1 {
		t.Fatalf("expected 1 file assessment, got %d", len(assessments))
	}
	fa := assessments[0]

	// Active changes: Num starts at 1, Prefix is empty.
	if len(fa.Active) != 2 {
		t.Fatalf("expected 2 active effects, got %d", len(fa.Active))
	}
	for i, ev := range fa.Active {
		if ev.Num != i+1 {
			t.Errorf("active[%d].Num: want %d, got %d", i, i+1, ev.Num)
		}
		if ev.Prefix != "" {
			t.Errorf("active[%d].Prefix: want empty, got %q", i, ev.Prefix)
		}
	}

	// Comment changes: Num starts at 1, Prefix is "CC".
	if len(fa.Comments) != 1 {
		t.Fatalf("expected 1 comment effect, got %d", len(fa.Comments))
	}
	if fa.Comments[0].Num != 1 || fa.Comments[0].Prefix != "CC" {
		t.Errorf("comment numbering wrong: num=%d prefix=%q", fa.Comments[0].Num, fa.Comments[0].Prefix)
	}

	// Future changes: Num starts at 1, Prefix is "F".
	if len(fa.Future) != 1 {
		t.Fatalf("expected 1 future effect, got %d", len(fa.Future))
	}
	if fa.Future[0].Num != 1 || fa.Future[0].Prefix != "F" {
		t.Errorf("future numbering wrong: num=%d prefix=%q", fa.Future[0].Num, fa.Future[0].Prefix)
	}
}

func TestBuildPerFileAssessments_GlobalCounts(t *testing.T) {
	rA := makeRule("P1-01", CategoryP1, "")
	rB := makeRule("P1-02", CategoryP1, "")
	rC := makeRule("P2-01", CategoryP2, "")
	applicable := []Effect{
		makeEffect(rA, "alpha.yaml", false, "0.10"),
		makeEffect(rB, "beta.yaml", false, "0.10"),
		makeEffect(rC, "beta.yaml", false, "0.10"),
	}
	result := makeDryRunResult(applicable, nil, nil)

	assessments, _, globalCounts := BuildPerFileAssessments(result, false)

	// Individual file counts.
	if len(assessments[0].Active) != 1 {
		t.Errorf("alpha.yaml: want 1 active, got %d", len(assessments[0].Active))
	}
	if len(assessments[1].Active) != 2 {
		t.Errorf("beta.yaml: want 2 active, got %d", len(assessments[1].Active))
	}

	// Global counts must equal the sum across all files.
	if globalCounts.P1Active != 2 {
		t.Errorf("global P1Active: want 2, got %d", globalCounts.P1Active)
	}
	if globalCounts.P2Active != 1 {
		t.Errorf("global P2Active: want 1, got %d", globalCounts.P2Active)
	}
	if globalCounts.P1Total != 2 {
		t.Errorf("global P1Total: want 2, got %d", globalCounts.P1Total)
	}
}

func TestBuildPerFileAssessments_Conflicts(t *testing.T) {
	rA := makeRule("P1-01", CategoryP1, "")
	rB := makeRule("P1-02", CategoryP1, "")
	e1 := makeEffect(rA, "cfg.yaml", false, "0.10")
	e2 := makeEffect(rB, "cfg.yaml", false, "0.11")
	conflicts := []Conflict{
		{
			Effect1: e1,
			Effect2: e2,
			Key:     "$.receivers.hostmetrics",
			Message: "both rules modify the same key",
		},
	}
	result := makeDryRunResult([]Effect{e1, e2}, nil, conflicts)

	assessments, globalConflicts, _ := BuildPerFileAssessments(result, false)

	// Global conflict list must have exactly one entry with correct rule IDs.
	if len(globalConflicts) != 1 {
		t.Fatalf("expected 1 global conflict, got %d", len(globalConflicts))
	}
	gc := globalConflicts[0]
	if gc.Rule1ID != "P1-01" {
		t.Errorf("conflict Rule1ID: want P1-01, got %q", gc.Rule1ID)
	}
	if gc.Rule2ID != "P1-02" {
		t.Errorf("conflict Rule2ID: want P1-02, got %q", gc.Rule2ID)
	}

	// The same conflict must be attached to the cfg.yaml FileAssessment.
	if len(assessments) != 1 {
		t.Fatalf("expected 1 file assessment, got %d", len(assessments))
	}
	if len(assessments[0].Conflicts) != 1 {
		t.Errorf("expected 1 per-file conflict on cfg.yaml, got %d", len(assessments[0].Conflicts))
	}
}
