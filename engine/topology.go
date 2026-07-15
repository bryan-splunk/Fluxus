package engine

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// PipelineRef identifies a named pipeline within a collector config file.
type PipelineRef struct {
	File       string
	Name       string // e.g. "traces/prod"
	Signal     string // "traces" | "metrics" | "logs"
	Receivers  []string
	Processors []string
	Exporters  []string
}

// ValidateTopology builds a pipeline graph across all provided config files
// and returns any connectivity issues found. It checks for:
//   - Exporters in agent configs that point to a receiver not present in gateway configs
//   - Pipelines that have exporters but no corresponding downstream receiver
//   - Processor ordering anti-patterns (memory_limiter, batch, truncate/filter order)
//   - Component definitions that are declared but not referenced by any pipeline (EG-6)
//   - Cross-file exporter→receiver dependencies that are broken (EG-7)
func ValidateTopology(state *State) []ValidationIssue {
	var issues []ValidationIssue

	pipelines := extractAllPipelines(state)
	if len(pipelines) == 0 {
		return nil
	}

	// Check for empty pipelines.
	for _, pipeline := range pipelines {
		if len(pipeline.Exporters) == 0 {
			issues = append(issues, ValidationIssue{
				Severity: "error",
				File:     pipeline.File,
				Message:  fmt.Sprintf("pipeline %q has no exporters — telemetry goes nowhere", pipeline.Name),
			})
		}
		if len(pipeline.Receivers) == 0 {
			issues = append(issues, ValidationIssue{
				Severity: "error",
				File:     pipeline.File,
				Message:  fmt.Sprintf("pipeline %q has no receivers — nothing feeds this pipeline", pipeline.Name),
			})
		}

		// Check exporters that reference OTLP/HTTP endpoints — ensure a
		// corresponding receiver exists in another config file.
		for _, exporter := range pipeline.Exporters {
			if isForwardingExporter(exporter) {
				if !hasOTLPReceiver(pipelines) {
					issues = append(issues, ValidationIssue{
						Severity: "warning",
						File:     pipeline.File,
						Message:  fmt.Sprintf("pipeline %q exports via %q but no OTLP receiver found in provided configs", pipeline.Name, exporter),
						Detail:   "If this is an agent→gateway topology, ensure the gateway config is also provided.",
					})
				}
			}
		}

		// Check processor ordering anti-patterns.
		issues = append(issues, checkProcessorOrder(pipeline)...)
	}

	// EG-6: detect component definitions declared but not used by any pipeline.
	issues = append(issues, findUnusedComponents(state, pipelines)...)

	// EG-7: detect exporters that forward to receiver types no longer present
	// across the provided configs (e.g. agent signalfx exporter pointing to a
	// gateway signalfx receiver that was removed during this upgrade).
	// Pass nil to run a general consistency scan against the current state.
	issues = append(issues, ScanCrossFileDependencies(state, nil)...)

	return issues
}

// ScanCrossFileDependencies checks whether any exporter in the provided state
// points to a known receiver type that is not present across the in-scope files.
// This is the EG-7 cross-file dependency check: after a receiver is removed from
// one config, this scan surfaces exporters in other configs that still reference
// the removed receiver's protocol/port.
//
// removedReceiverTypes is a list of component type names that were removed (e.g.
// "signalfx", "sapm", "fluentforward"). Pass nil to skip targeted removal checks
// and run a general consistency scan instead.
func ScanCrossFileDependencies(state *State, removedReceiverTypes []string) []ValidationIssue {
	var issues []ValidationIssue

	// Build a set of all receiver types currently present across all files.
	presentReceivers := make(map[string]bool)
	for _, node := range state.Files {
		doc := unwrapDocument(node)
		if doc == nil || doc.Kind != yaml.MappingNode {
			continue
		}
		receiversNode := mappingGet(doc, "receivers")
		if receiversNode == nil || receiversNode.Kind != yaml.MappingNode {
			continue
		}
		for i := 0; i+1 < len(receiversNode.Content); i += 2 {
			key := receiversNode.Content[i].Value
			// Normalise named instances: "kafka/consumer" → "kafka"
			base := strings.SplitN(key, "/", 2)[0]
			presentReceivers[base] = true
		}
	}

	// Map known exporter type names to the receiver type they forward to.
	// This covers the most common agent→gateway forwarding pairs.
	exporterToReceiver := map[string]string{
		"signalfx": "signalfx",
		"sapm":     "sapm",
		"otlp":     "otlp",
		"otlphttp": "otlp",
		"zipkin":   "zipkin",
		"jaeger":   "jaeger",
	}

	targeted := make(map[string]bool, len(removedReceiverTypes))
	for _, receiverType := range removedReceiverTypes {
		targeted[receiverType] = true
	}

	// Scan every exporter in every file.
	for filePath, node := range state.Files {
		doc := unwrapDocument(node)
		if doc == nil || doc.Kind != yaml.MappingNode {
			continue
		}
		exportersNode := mappingGet(doc, "exporters")
		if exportersNode == nil || exportersNode.Kind != yaml.MappingNode {
			continue
		}
		for i := 0; i+1 < len(exportersNode.Content); i += 2 {
			exporterKey := exportersNode.Content[i].Value
			exporterBase := strings.SplitN(exporterKey, "/", 2)[0]

			targetReceiverType, known := exporterToReceiver[exporterBase]
			if !known {
				continue
			}

			// If this exporter type forwards to a receiver that is not present
			// in any in-scope file, report it.
			if !presentReceivers[targetReceiverType] {
				// Only report if: no targeted list (general scan), or the missing
				// receiver is in the targeted removal list.
				if len(targeted) == 0 || targeted[targetReceiverType] {
					issues = append(issues, ValidationIssue{
						Severity: "warning",
						File:     filePath,
						Message: fmt.Sprintf(
							"exporter %q forwards to a %q receiver but no %q receiver is defined across in-scope files",
							exporterKey, targetReceiverType, targetReceiverType,
						),
						Detail: "This exporter may be sending to a removed or not-yet-configured receiver. " +
							"Verify the target config is included in the scope, or update this exporter.",
					})
				}
			}
		}
	}

	return issues
}

// extractAllPipelines parses all config files and returns their pipeline definitions.
func extractAllPipelines(state *State) []PipelineRef {
	var pipelines []PipelineRef

	for filePath, node := range state.Files {
		doc := unwrapDocument(node)
		if doc == nil || doc.Kind != yaml.MappingNode {
			continue
		}

		serviceNode := mappingGet(doc, "service")
		if serviceNode == nil {
			continue
		}
		pipelinesNode := mappingGet(serviceNode, "pipelines")
		if pipelinesNode == nil || pipelinesNode.Kind != yaml.MappingNode {
			continue
		}

		for i := 0; i+1 < len(pipelinesNode.Content); i += 2 {
			pipelineName := pipelinesNode.Content[i].Value
			pipelineBody := pipelinesNode.Content[i+1]

			signal := strings.SplitN(pipelineName, "/", 2)[0]
			ref := PipelineRef{
				File:   filePath,
				Name:   pipelineName,
				Signal: signal,
			}

			ref.Receivers = stringSeqFromMapping(pipelineBody, "receivers")
			ref.Processors = stringSeqFromMapping(pipelineBody, "processors")
			ref.Exporters = stringSeqFromMapping(pipelineBody, "exporters")

			pipelines = append(pipelines, ref)
		}
	}
	return pipelines
}

// findUnusedComponents detects component definitions (receivers, processors,
// exporters, connectors) that are declared in the config but not referenced
// in any pipeline. These are dead entries that add confusion and consume resources.
// Corresponds to PATH-07 in DATA-PATH-KNOWLEDGE.md (EG-6).
func findUnusedComponents(state *State, pipelines []PipelineRef) []ValidationIssue {
	var issues []ValidationIssue

	// Build the set of all component keys referenced across all pipeline arrays.
	usedInPipeline := make(map[string]bool) // key: "file::componentKey"
	for _, pipeline := range pipelines {
		for _, receiver := range pipeline.Receivers {
			usedInPipeline[pipeline.File+"::"+receiver] = true
		}
		for _, processor := range pipeline.Processors {
			usedInPipeline[pipeline.File+"::"+processor] = true
		}
		for _, exporter := range pipeline.Exporters {
			usedInPipeline[pipeline.File+"::"+exporter] = true
		}
	}

	componentSections := []string{"receivers", "processors", "exporters", "connectors"}

	for filePath, node := range state.Files {
		doc := unwrapDocument(node)
		if doc == nil || doc.Kind != yaml.MappingNode {
			continue
		}

		for _, section := range componentSections {
			sectionNode := mappingGet(doc, section)
			if sectionNode == nil || sectionNode.Kind != yaml.MappingNode {
				continue
			}
			for i := 0; i+1 < len(sectionNode.Content); i += 2 {
				key := sectionNode.Content[i].Value
				if !usedInPipeline[filePath+"::"+key] {
					issues = append(issues, ValidationIssue{
						Severity: "warning",
						File:     filePath,
						Message: fmt.Sprintf(
							"%s %q is defined but not referenced in any pipeline",
							strings.TrimSuffix(section, "s"), key,
						),
						Detail: "Remove or add to a pipeline. Unused components are loaded and consume resources.",
					})
				}
			}
		}
	}
	return issues
}

// checkProcessorOrder flags known bad processor ordering patterns.
func checkProcessorOrder(pipeline PipelineRef) []ValidationIssue {
	var issues []ValidationIssue

	memLimiterIndex := indexOf(pipeline.Processors, func(s string) bool {
		return strings.Contains(s, "memory_limiter")
	})
	batchIndex := indexOf(pipeline.Processors, func(s string) bool {
		return strings.Contains(s, "batch")
	})
	filterIndex := indexOf(pipeline.Processors, func(s string) bool {
		return strings.Contains(s, "filter")
	})
	truncateIndex := indexOf(pipeline.Processors, func(s string) bool {
		return strings.Contains(s, "truncat")
	})
	resourceDetectionIndex := indexOf(pipeline.Processors, func(s string) bool {
		return s == "resource_detection" || s == "resourcedetection" ||
			strings.HasPrefix(s, "resource_detection/") || strings.HasPrefix(s, "resourcedetection/")
	})

	// Anti-pattern: memory_limiter not first.
	if memLimiterIndex > 0 {
		issues = append(issues, ValidationIssue{
			Severity: "warning",
			File:     pipeline.File,
			Message:  fmt.Sprintf("pipeline %q: memory_limiter is not the first processor", pipeline.Name),
			Detail:   "memory_limiter should be first so it can back-pressure before downstream processors consume memory.",
		})
	}

	// Anti-pattern: batch before memory_limiter.
	if batchIndex >= 0 && memLimiterIndex >= 0 && batchIndex < memLimiterIndex {
		issues = append(issues, ValidationIssue{
			Severity: "warning",
			File:     pipeline.File,
			Message:  fmt.Sprintf("pipeline %q: batch processor appears before memory_limiter", pipeline.Name),
			Detail:   "batch should come after memory_limiter to avoid buffering data before the limiter can act.",
		})
	}

	// Anti-pattern: batch before resource_detection (resource attrs won't be batched with correct attrs).
	// PROC-order anti-pattern from DATA-PATH-KNOWLEDGE.md.
	if batchIndex >= 0 && resourceDetectionIndex >= 0 && batchIndex < resourceDetectionIndex {
		issues = append(issues, ValidationIssue{
			Severity: "warning",
			File:     pipeline.File,
			Message:  fmt.Sprintf("pipeline %q: batch processor appears before resource_detection", pipeline.Name),
			Detail:   "resource_detection should run before batch so resource attributes are attached before telemetry is batched.",
		})
	}

	// Anti-pattern: truncation before filtering wastes CPU.
	if truncateIndex >= 0 && filterIndex >= 0 && truncateIndex < filterIndex {
		issues = append(issues, ValidationIssue{
			Severity: "warning",
			File:     pipeline.File,
			Message:  fmt.Sprintf("pipeline %q: truncation processor appears before filter processor", pipeline.Name),
			Detail:   "Filtering before truncation is more efficient — dropped spans avoid truncation work.",
		})
	}

	return issues
}

// stringSeqFromMapping returns the string values of a sequence child of node.
func stringSeqFromMapping(node *yaml.Node, key string) []string {
	seq := mappingGet(node, key)
	if seq == nil || seq.Kind != yaml.SequenceNode {
		return nil
	}
	var results []string
	for _, item := range seq.Content {
		results = append(results, item.Value)
	}
	return results
}

// isForwardingExporter returns true for exporters that forward to another collector.
func isForwardingExporter(name string) bool {
	name = strings.ToLower(name)
	return strings.HasPrefix(name, "otlp") ||
		strings.HasPrefix(name, "otlphttp") ||
		strings.HasPrefix(name, "sapm")
}

// hasOTLPReceiver returns true if any pipeline in any file has an OTLP receiver.
func hasOTLPReceiver(pipelines []PipelineRef) bool {
	for _, pipeline := range pipelines {
		for _, receiver := range pipeline.Receivers {
			if strings.HasPrefix(strings.ToLower(receiver), "otlp") {
				return true
			}
		}
	}
	return false
}

// indexOf returns the first index where predicate returns true, or -1.
func indexOf(slice []string, predicate func(string) bool) int {
	for i, s := range slice {
		if predicate(s) {
			return i
		}
	}
	return -1
}
