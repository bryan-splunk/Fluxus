package engine

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// ============================================================
// executeKeyMove tests
// ============================================================

func TestExecuteKeyMove_SimpleRename(t *testing.T) {
	input := `receivers:
  hostmetrics:
    collection_interval: 60s
`
	var root yaml.Node
	if err := yaml.Unmarshal([]byte(input), &root); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	km := KeyMove{From: "$.receivers.hostmetrics", To: "$.receivers.host_metrics"}
	warnings := executeKeyMove(&root, km)
	if len(warnings) != 0 {
		t.Errorf("unexpected warnings: %v", warnings)
	}

	// After rename the old key must be gone and the new key must be present.
	doc := unwrapDocument(&root)
	if mappingGet(doc, "receivers") == nil {
		t.Fatal("receivers section missing after rename")
	}
	rcvNode := mappingGet(doc, "receivers")
	if mappingGet(rcvNode, "hostmetrics") != nil {
		t.Error("old key hostmetrics still present after rename")
	}
	if mappingGet(rcvNode, "host_metrics") == nil {
		t.Error("new key host_metrics not present after rename")
	}
}

func TestExecuteKeyMove_NamedInstanceRename(t *testing.T) {
	input := `receivers:
  hostmetrics/process:
    collection_interval: 60s
`
	var root yaml.Node
	if err := yaml.Unmarshal([]byte(input), &root); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	km := KeyMove{From: "$.receivers.hostmetrics", To: "$.receivers.host_metrics"}
	executeKeyMove(&root, km)

	rcvNode := mappingGet(unwrapDocument(&root), "receivers")
	if mappingGet(rcvNode, "hostmetrics/process") != nil {
		t.Error("old named-instance key hostmetrics/process still present")
	}
	if mappingGet(rcvNode, "host_metrics/process") == nil {
		t.Error("renamed named-instance key host_metrics/process not found")
	}
}

func TestExecuteKeyMove_CrossParentMove(t *testing.T) {
	input := `exporters:
  splunk_hec:
    batcher:
      min_size_items: 100
`
	var root yaml.Node
	if err := yaml.Unmarshal([]byte(input), &root); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	km := KeyMove{
		From: "$.exporters.splunk_hec.batcher.min_size_items",
		To:   "$.exporters.splunk_hec.sending_queue.batch.min_size_items",
	}
	warnings := executeKeyMove(&root, km)
	if len(warnings) != 0 {
		t.Errorf("unexpected warnings: %v", warnings)
	}

	doc := unwrapDocument(&root)
	expNode := mappingGet(doc, "exporters")
	hecNode := mappingGet(expNode, "splunk_hec")

	// Old path must be gone.
	batcherNode := mappingGet(hecNode, "batcher")
	if batcherNode != nil && mappingGet(batcherNode, "min_size_items") != nil {
		t.Error("old batcher.min_size_items still present")
	}

	// New path must exist with correct value.
	sqNode := mappingGet(hecNode, "sending_queue")
	if sqNode == nil {
		t.Fatal("sending_queue not created")
	}
	batchNode := mappingGet(sqNode, "batch")
	if batchNode == nil {
		t.Fatal("sending_queue.batch not created")
	}
	valNode := mappingGet(batchNode, "min_size_items")
	if valNode == nil || valNode.Value != "100" {
		t.Errorf("expected min_size_items=100, got %v", valNode)
	}
}

func TestExecuteKeyMove_Delete(t *testing.T) {
	input := `exporters:
  splunk_hec:
    deprecated_field: old_value
    keep_field: keep
`
	var root yaml.Node
	if err := yaml.Unmarshal([]byte(input), &root); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	km := KeyMove{From: "$.exporters.splunk_hec.deprecated_field"}
	warnings := executeKeyMove(&root, km)
	if len(warnings) != 0 {
		t.Errorf("unexpected warnings: %v", warnings)
	}

	hecNode := mappingGet(mappingGet(unwrapDocument(&root), "exporters"), "splunk_hec")
	if mappingGet(hecNode, "deprecated_field") != nil {
		t.Error("deleted key deprecated_field still present")
	}
	if mappingGet(hecNode, "keep_field") == nil {
		t.Error("unrelated key keep_field was removed")
	}
}

func TestExecuteKeyMove_InjectDefault(t *testing.T) {
	input := `processors:
  filter:
    error_mode: ignore
`
	var root yaml.Node
	if err := yaml.Unmarshal([]byte(input), &root); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	km := KeyMove{To: "$.processors.filter.drop_output", Default: "false"}
	warnings := executeKeyMove(&root, km)
	if len(warnings) != 0 {
		t.Errorf("unexpected warnings: %v", warnings)
	}

	filterNode := mappingGet(mappingGet(unwrapDocument(&root), "processors"), "filter")
	injected := mappingGet(filterNode, "drop_output")
	if injected == nil || injected.Value != "false" {
		t.Errorf("expected drop_output=false, got %v", injected)
	}
}

func TestExecuteKeyMove_SequenceReplace(t *testing.T) {
	input := `service:
  pipelines:
    metrics:
      receivers:
        - hostmetrics
        - prometheus
`
	var root yaml.Node
	if err := yaml.Unmarshal([]byte(input), &root); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	km := KeyMove{
		SequencePath: "$.service.pipelines.*.receivers",
		OldValue:     "hostmetrics",
		NewValue:     "host_metrics",
	}
	executeKeyMove(&root, km)

	doc := unwrapDocument(&root)
	pipelinesNode := mappingGet(mappingGet(doc, "service"), "pipelines")
	metricsNode := mappingGet(pipelinesNode, "metrics")
	rcvSeq := mappingGet(metricsNode, "receivers")
	if rcvSeq == nil {
		t.Fatal("receivers sequence missing")
	}
	found := false
	for _, item := range rcvSeq.Content {
		if item.Value == "host_metrics" {
			found = true
		}
		if item.Value == "hostmetrics" {
			t.Error("old sequence value hostmetrics still present")
		}
	}
	if !found {
		t.Error("renamed value host_metrics not in sequence")
	}
}

// ============================================================
// deleteFromParent tests
// ============================================================

func TestDeleteFromParent_PreservesNeighbourComments(t *testing.T) {
	// Build a mapping manually: [keyA, valA, keyB (with HeadComment), valB]
	// Then delete keyA/valA and verify that keyB's HeadComment survives.
	mapping := &yaml.Node{Kind: yaml.MappingNode}
	keyA := &yaml.Node{Kind: yaml.ScalarNode, Value: "a"}
	valA := &yaml.Node{Kind: yaml.ScalarNode, Value: "1"}
	keyB := &yaml.Node{Kind: yaml.ScalarNode, Value: "b", HeadComment: "# comment on b"}
	valB := &yaml.Node{Kind: yaml.ScalarNode, Value: "2"}
	mapping.Content = []*yaml.Node{keyA, valA, keyB, valB}

	deleteFromParent(mapping, 0) // delete a

	if len(mapping.Content) != 2 {
		t.Fatalf("expected 2 nodes remaining, got %d", len(mapping.Content))
	}
	if mapping.Content[0].Value != "b" {
		t.Errorf("expected key b, got %s", mapping.Content[0].Value)
	}
	if mapping.Content[0].HeadComment != "# comment on b" {
		t.Errorf("HeadComment lost: %q", mapping.Content[0].HeadComment)
	}
}

func TestDeleteFromParent_OrphanedCommentRehomed(t *testing.T) {
	// Predecessor value carries a FootComment (orphaned template block).
	// Deleting keyB should re-home keyB's HeadComment onto keyC.
	mapping := &yaml.Node{Kind: yaml.MappingNode}
	keyA := &yaml.Node{Kind: yaml.ScalarNode, Value: "a"}
	valA := &yaml.Node{Kind: yaml.ScalarNode, Value: "1", FootComment: "# orphaned above b"}
	keyB := &yaml.Node{Kind: yaml.ScalarNode, Value: "b", HeadComment: "# own comment of b"}
	valB := &yaml.Node{Kind: yaml.ScalarNode, Value: "2"}
	keyC := &yaml.Node{Kind: yaml.ScalarNode, Value: "c"}
	valC := &yaml.Node{Kind: yaml.ScalarNode, Value: "3"}
	mapping.Content = []*yaml.Node{keyA, valA, keyB, valB, keyC, valC}

	deleteFromParent(mapping, 2) // delete b

	if len(mapping.Content) != 4 {
		t.Fatalf("expected 4 nodes remaining, got %d", len(mapping.Content))
	}
	// valA's FootComment should have been moved.
	if valA.FootComment != "" {
		t.Error("predecessor FootComment not drained")
	}
	// keyC should carry both orphaned texts.
	if !strings.Contains(mapping.Content[2].HeadComment, "orphaned above b") {
		t.Errorf("orphaned FootComment not re-homed onto keyC: %q", mapping.Content[2].HeadComment)
	}
	if !strings.Contains(mapping.Content[2].HeadComment, "own comment of b") {
		t.Errorf("keyB's HeadComment not re-homed onto keyC: %q", mapping.Content[2].HeadComment)
	}
}

func TestDeleteFromParent_DeleteLastPair(t *testing.T) {
	// When the last pair is deleted, orphaned comments fall back to
	// the predecessor value's FootComment for later fixSectionCommentDrift.
	mapping := &yaml.Node{Kind: yaml.MappingNode}
	keyA := &yaml.Node{Kind: yaml.ScalarNode, Value: "a"}
	valA := &yaml.Node{Kind: yaml.ScalarNode, Value: "1"}
	keyB := &yaml.Node{Kind: yaml.ScalarNode, Value: "b", HeadComment: "# b head"}
	valB := &yaml.Node{Kind: yaml.ScalarNode, Value: "2"}
	mapping.Content = []*yaml.Node{keyA, valA, keyB, valB}

	deleteFromParent(mapping, 2) // delete b (the last pair)

	if len(mapping.Content) != 2 {
		t.Fatalf("expected 2 nodes remaining, got %d", len(mapping.Content))
	}
	// b's head comment should have been moved to the predecessor value's FootComment.
	if !strings.Contains(valA.FootComment, "b head") {
		t.Errorf("last-pair head comment not promoted to predecessor FootComment: %q", valA.FootComment)
	}
}

// ============================================================
// fixSectionCommentDrift tests
// ============================================================

func TestFixSectionCommentDrift_PromotesFootComment(t *testing.T) {
	// Build a minimal document where the last leaf of a component has a
	// FootComment that should be promoted to the next sibling key.
	input := `receivers:
  hostmetrics:
    collection_interval: 60s
  prometheus:
    config: {}
`
	var root yaml.Node
	if err := yaml.Unmarshal([]byte(input), &root); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// Manually plant a FootComment on a deep leaf to simulate the bug.
	rcv := mappingGet(unwrapDocument(&root), "receivers")
	hm := mappingGet(rcv, "hostmetrics")
	leaf := mappingGet(hm, "collection_interval")
	if leaf == nil {
		t.Fatal("could not locate leaf node for test setup")
	}
	leaf.FootComment = "# section separator"

	fixSectionCommentDrift(&root)

	// After repair the FootComment should have moved to the prometheus key's HeadComment.
	for i := 0; i+1 < len(rcv.Content); i += 2 {
		if rcv.Content[i].Value == "prometheus" {
			if !strings.Contains(rcv.Content[i].HeadComment, "section separator") {
				t.Errorf("FootComment not promoted to sibling HeadComment: %q", rcv.Content[i].HeadComment)
			}
			return
		}
	}
	t.Fatal("prometheus key not found after fixSectionCommentDrift")
}

// ============================================================
// collapseCommentBlankLinesInString tests
// ============================================================

func TestCollapseCommentBlankLines_SameIndent(t *testing.T) {
	input := "# line 1\n\n# line 2\n\n# line 3"
	got := collapseCommentBlankLinesInString(input)
	if strings.Contains(got, "\n\n") {
		t.Errorf("blank lines not collapsed between same-indent comment lines:\n%s", got)
	}
}

func TestCollapseCommentBlankLines_DifferentIndent(t *testing.T) {
	// A blank line between differently-indented comment lines should be kept.
	input := "# section\n\n  # field comment"
	got := collapseCommentBlankLinesInString(input)
	if !strings.Contains(got, "\n\n") {
		t.Errorf("blank line between different-indent comments should be preserved:\n%s", got)
	}
}

func TestCollapseCommentBlankLines_NonCommentLines(t *testing.T) {
	// Blank lines between non-comment lines must not be touched.
	input := "key: value\n\nanother: value"
	got := collapseCommentBlankLinesInString(input)
	if got != input {
		t.Errorf("non-comment blank line was modified:\ngot: %s", got)
	}
}

// ============================================================
// resolveWildcards tests
// ============================================================

func TestResolveWildcards_NamedInstance(t *testing.T) {
	got := resolveWildcards(
		"exporters.splunk_hec/prod.batcher.min_size_items",
		"exporters.splunk_hec.batcher.min_size_items",
		"exporters.splunk_hec.sending_queue.batch.min_size_items",
	)
	want := "exporters.splunk_hec/prod.sending_queue.batch.min_size_items"
	if got != want {
		t.Errorf("resolveWildcards named-instance: got %q, want %q", got, want)
	}
}

func TestResolveWildcards_ExplicitWildcard(t *testing.T) {
	got := resolveWildcards(
		"exporters.otlp.batcher.min_size_items",
		"exporters.*.batcher.min_size_items",
		"exporters.*.sending_queue.batch.min_size_items",
	)
	want := "exporters.otlp.sending_queue.batch.min_size_items"
	if got != want {
		t.Errorf("resolveWildcards wildcard: got %q, want %q", got, want)
	}
}

// ============================================================
// applyKeyMoves round-trip smoke test
// ============================================================

func TestApplyKeyMoves_RoundTrip(t *testing.T) {
	input := `receivers:
  hostmetrics:
    collection_interval: 60s
exporters:
  splunk_hec:
    token: abc123
`
	rule := &Rule{
		ID: "TEST-P1-01",
		Migration: Migration{
			KeyMoves: []KeyMove{
				{From: "$.receivers.hostmetrics", To: "$.receivers.host_metrics"},
			},
		},
	}
	effect := Effect{Rule: rule}

	result, warnings, err := applyKeyMoves(input, effect)
	if err != nil {
		t.Fatalf("applyKeyMoves error: %v", err)
	}
	if len(warnings) != 0 {
		t.Errorf("unexpected warnings: %v", warnings)
	}
	if strings.Contains(result, "hostmetrics:") {
		t.Error("old key hostmetrics still present in output")
	}
	if !strings.Contains(result, "host_metrics:") {
		t.Error("new key host_metrics not found in output")
	}
	// Ensure unrelated sections survive.
	if !strings.Contains(result, "splunk_hec:") {
		t.Error("unrelated exporters section missing from output")
	}
}
