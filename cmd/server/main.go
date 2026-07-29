package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/bryan-splunk/fluxus/engine"
)

var rulesDir = "rules"
var webDir = "web"

// Start is called by the CLI's server sub-command.
func Start(port int, rulesDirectory string) error {
	return runServer(port, rulesDirectory)
}

func runServer(port int, rulesDirectory string) error {
	rulesDir = rulesDirectory
	// Bind to loopback only — this server is intended for local/trusted use.
	address := fmt.Sprintf("127.0.0.1:%d", port)

	mux := http.NewServeMux()

	// Serve the web UI
	mux.Handle("/", http.FileServer(http.Dir(webDir)))

	// API endpoints
	mux.HandleFunc("/api/assess", handleAssess)
	mux.HandleFunc("/api/apply", handleApply)
	mux.HandleFunc("/api/rules", handleListRules)

	log.Printf("🚀 FLUXUS server listening on http://127.0.0.1:%d", port)
	return http.ListenAndServe(address, mux)
}

// assessRequest is the JSON body for POST /api/assess.
type assessRequest struct {
	Files           map[string]string `json:"files"` // filename → content
	TargetVersion   string            `json:"target_version"`
	IncludeComments bool              `json:"include_comments"`
	IncludeGuidance bool              `json:"include_guidance"`
}

// assessResponse is the JSON response for POST /api/assess.
type assessResponse struct {
	Report          string                  `json:"report"` // markdown — for download only
	PerFile         []engine.FileAssessment `json:"per_file"`
	GlobalCounts    engine.EffectCounts     `json:"global_counts"`
	Conflicts       []engine.ConflictView   `json:"conflicts"`
	ApplicableCount int                     `json:"applicable_count"`
	FutureCount     int                     `json:"future_count"`
	ConflictCount   int                     `json:"conflict_count"`
	Errors          []string                `json:"errors,omitempty"`
}

func handleAssess(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 50*1024*1024) // 50 MB cap

	var req assessRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	state, err := engine.NewState(req.Files)
	if err != nil {
		jsonError(w, "parse error: "+err.Error(), http.StatusUnprocessableEntity)
		return
	}

	rules, err := engine.LoadRulesTree(rulesDir)
	if err != nil {
		jsonError(w, "cannot load rules: "+err.Error(), http.StatusInternalServerError)
		return
	}

	result, err := engine.DryRun(state, rules, req.TargetVersion, req.IncludeComments)
	if err != nil {
		jsonError(w, "dry run failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	files := make([]string, 0, len(req.Files))
	for name := range req.Files {
		files = append(files, name)
	}

	report, err := engine.RenderPreAssessment(result, req.TargetVersion, files, req.IncludeGuidance)
	if err != nil {
		jsonError(w, "render failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	perFile, conflicts, globalCounts := engine.BuildPerFileAssessments(result, req.IncludeGuidance)

	jsonOK(w, assessResponse{
		Report:          report,
		PerFile:         perFile,
		GlobalCounts:    globalCounts,
		Conflicts:       conflicts,
		ApplicableCount: len(engine.AllEffects(result.ApplicableTicks)),
		FutureCount:     len(engine.AllEffects(result.FutureTicks)),
		ConflictCount:   len(result.Conflicts),
	})
}

// applyRequest is the JSON body for POST /api/apply.
type applyRequest struct {
	Files           map[string]string `json:"files"`
	TargetVersion   string            `json:"target_version"`
	IncludeComments bool              `json:"include_comments"`
	IncludeGuidance bool              `json:"include_guidance"`
	Select          string            `json:"select"` // "all", "p1", "p2", "p3", or comma-separated rule IDs
}

// applyResponse is the JSON response for POST /api/apply.
type applyResponse struct {
	UpdatedFiles map[string]string `json:"updated_files"`
	Report       string            `json:"report"`
	AppliedCount int               `json:"applied_count"`
	GuidedCount  int               `json:"guided_count"`
	CommentCount int               `json:"comment_count"`
	WarningCount int               `json:"warning_count"`
	Errors       []string          `json:"errors,omitempty"`
}

func handleApply(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 50*1024*1024) // 50 MB cap

	var req applyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	state, err := engine.NewState(req.Files)
	if err != nil {
		jsonError(w, "parse error: "+err.Error(), http.StatusUnprocessableEntity)
		return
	}

	rules, err := engine.LoadRulesTree(rulesDir)
	if err != nil {
		jsonError(w, "cannot load rules: "+err.Error(), http.StatusInternalServerError)
		return
	}

	approved := strings.Split(req.Select, ",")
	for i, approvalID := range approved {
		approved[i] = strings.TrimSpace(approvalID)
	}

	result, err := engine.Apply(state, rules, engine.ApplyOptions{
		TargetVersion:   req.TargetVersion,
		IncludeComments: req.IncludeComments,
		ApprovedIDs:     approved,
	})
	if err != nil {
		jsonError(w, "apply failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Topology validation
	newState, _ := engine.NewState(result.UpdatedFiles)
	var topologyIssues []engine.ValidationIssue
	if newState != nil {
		topologyIssues = engine.ValidateTopology(newState)
	}

	operationalReport, _ := engine.RenderOperationalAssessment(engine.OperationalReportData{
		TargetVersion:   req.TargetVersion,
		AppliedEffects:  result.AppliedEffects,
		GuidedEffects:   result.GuidedEffects,
		CommentEffects:  result.CommentEffects,
		UpdatedFiles:    result.UpdatedFiles,
		Warnings:        result.Warnings,
		TopologyIssues:  topologyIssues,
		IncludeGuidance: req.IncludeGuidance,
	})

	jsonOK(w, applyResponse{
		UpdatedFiles: result.UpdatedFiles,
		Report:       operationalReport,
		AppliedCount: len(result.AppliedEffects),
		GuidedCount:  len(result.GuidedEffects),
		CommentCount: len(result.CommentEffects),
		WarningCount: len(result.Warnings),
	})
}

func handleListRules(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	rules, err := engine.LoadRulesTree(rulesDir)
	if err != nil {
		jsonError(w, "cannot load rules: "+err.Error(), http.StatusInternalServerError)
		return
	}
	type ruleSummary struct {
		ID         string          `json:"id"`
		Category   engine.Category `json:"category"`
		Introduced string          `json:"introduced"`
		Title      string          `json:"title"`
	}
	out := make([]ruleSummary, len(rules))
	for i, rule := range rules {
		out[i] = ruleSummary{ID: rule.ID, Category: rule.Category, Introduced: rule.Introduced, Title: rule.Title}
	}
	jsonOK(w, out)
}

func jsonOK(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func jsonError(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	// Headers already written; discard any secondary encoder error.
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
