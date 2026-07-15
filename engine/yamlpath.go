package engine

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// ============================================================
// YAMLPath evaluation — read-only tree traversal
// ============================================================

// evaluatePath walks a yaml.Node tree following the YAMLPath expression.
// Returns (found, matchedPath). found is true when any node satisfies the path.
//
// Supported syntax:
//
//	$.key          — top-level key
//	$.a.b.c        — nested path
//	$.a.*          — any direct child of a
//	$.**.key       — recursive descent: key at any depth
//
// Named component instances: "$.receivers.kafka" also matches "kafka/consumer",
// "kafka/producer", etc. — any key that starts with "kafka/" is treated as a
// named instance of the same component type.
func evaluatePath(path string, root *yaml.Node) (bool, string) {
	if root == nil {
		return false, ""
	}
	path = strings.TrimPrefix(path, "$.")
	path = strings.TrimPrefix(path, "$")
	if path == "" {
		return true, "$"
	}

	segments := strings.SplitN(path, ".", 2)
	head := segments[0]
	rest := ""
	if len(segments) > 1 {
		rest = segments[1]
	}

	node := root
	if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		node = node.Content[0]
	}

	if node.Kind != yaml.MappingNode {
		return false, ""
	}

	switch head {
	case "**":
		return recursiveSearch(rest, node)

	case "*":
		for i := 1; i < len(node.Content); i += 2 {
			valueNode := node.Content[i]
			if rest == "" {
				return true, node.Content[i-1].Value
			}
			if matched, matchedPath := evaluatePath(rest, valueNode); matched {
				return true, node.Content[i-1].Value + "." + matchedPath
			}
		}
		return false, ""

	default:
		// Match the exact key OR any named instance (key/variant convention).
		for i := 0; i+1 < len(node.Content); i += 2 {
			keyNode := node.Content[i]
			if keyNode.Value == head || strings.HasPrefix(keyNode.Value, head+"/") {
				valueNode := node.Content[i+1]
				if rest == "" {
					return true, keyNode.Value
				}
				if matched, matchedPath := evaluatePath(rest, valueNode); matched {
					return true, keyNode.Value + "." + matchedPath
				}
			}
		}
		return false, ""
	}
}

// recursiveSearch searches for a path at any depth within node.
func recursiveSearch(path string, node *yaml.Node) (bool, string) {
	if node == nil {
		return false, ""
	}
	if path != "" {
		if matched, matchedPath := evaluatePath(path, node); matched {
			return true, matchedPath
		}
	}
	switch node.Kind {
	case yaml.MappingNode:
		for i := 1; i < len(node.Content); i += 2 {
			if matched, mp := recursiveSearch(path, node.Content[i]); matched {
				return true, mp
			}
		}
	case yaml.SequenceNode:
		for _, child := range node.Content {
			if matched, mp := recursiveSearch(path, child); matched {
				return true, mp
			}
		}
	case yaml.DocumentNode:
		for _, child := range node.Content {
			if matched, mp := recursiveSearch(path, child); matched {
				return true, mp
			}
		}
	default:
		// ScalarNode, AliasNode — no children to recurse into
	}
	return false, ""
}

// resolveNode returns (key, valueNode) for the first match of path in root.
func resolveNode(path string, root *yaml.Node) (string, *yaml.Node) {
	path = strings.TrimPrefix(path, "$.")
	path = strings.TrimPrefix(path, "$")
	if path == "" || root == nil {
		return "", root
	}

	node := root
	if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		node = node.Content[0]
	}

	segments := strings.SplitN(path, ".", 2)
	head := segments[0]
	rest := ""
	if len(segments) > 1 {
		rest = segments[1]
	}

	if node.Kind != yaml.MappingNode {
		return "", nil
	}

	for i := 0; i+1 < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		valueNode := node.Content[i+1]
		switch head {
		case "*":
			if rest == "" {
				return keyNode.Value, valueNode
			}
			if k, v := resolveNode(rest, valueNode); v != nil {
				return k, v
			}
		default:
			if keyNode.Value == head || strings.HasPrefix(keyNode.Value, head+"/") {
				if rest == "" {
					return keyNode.Value, valueNode
				}
				return resolveNode(rest, valueNode)
			}
		}
	}
	return "", nil
}

// ============================================================
// findAll — mutation-target traversal
// ============================================================

// pathHit records one resolved match returned by findAll.
type pathHit struct {
	parent   *yaml.Node // MappingNode containing the key-value pair
	keyIndex int        // index into parent.Content of the key node (value is keyIndex+1)
	fullPath string     // concrete path string that matched (wildcards expanded)
}

// findAll returns all (parent, keyIdx) pairs that match the YAMLPath pattern.
// Supports * wildcards and named-instance suffix matching (key/variant).
func findAll(root *yaml.Node, path string) []pathHit {
	node := unwrapDocument(root)
	if node == nil {
		return nil
	}
	return findAllNode(node, path, "")
}

func findAllNode(node *yaml.Node, remaining, soFar string) []pathHit {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	if remaining == "" {
		return nil
	}

	segments := strings.SplitN(remaining, ".", 2)
	head := segments[0]
	rest := ""
	if len(segments) > 1 {
		rest = segments[1]
	}

	var hits []pathHit
	switch head {
	case "*":
		for i := 0; i+1 < len(node.Content); i += 2 {
			keyValue := node.Content[i].Value
			current := soFar + keyValue
			if rest == "" {
				hits = append(hits, pathHit{parent: node, keyIndex: i, fullPath: current})
			} else {
				hits = append(hits, findAllNode(node.Content[i+1], rest, current+".")...)
			}
		}
	default:
		for i := 0; i+1 < len(node.Content); i += 2 {
			keyValue := node.Content[i].Value
			if keyValue == head || strings.HasPrefix(keyValue, head+"/") {
				current := soFar + keyValue
				if rest == "" {
					hits = append(hits, pathHit{parent: node, keyIndex: i, fullPath: current})
				} else {
					hits = append(hits, findAllNode(node.Content[i+1], rest, current+".")...)
				}
			}
		}
	}
	return hits
}

// ============================================================
// Tree mutation helpers
// ============================================================

// setAtPath creates or overwrites the node at the given concrete path,
// creating any missing intermediate mapping nodes.
func setAtPath(root *yaml.Node, path string, value *yaml.Node) error {
	segments := strings.Split(path, ".")
	node := unwrapDocument(root)
	if node == nil {
		return fmt.Errorf("nil root")
	}

	for _, segment := range segments[:len(segments)-1] {
		if node.Kind != yaml.MappingNode {
			return fmt.Errorf("expected mapping at %q", segment)
		}
		child := mappingGet(node, segment)
		if child == nil {
			newMap := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
			mappingSetExact(node, segment, newMap)
			node = newMap
		} else {
			node = child
		}
	}

	if node.Kind != yaml.MappingNode {
		return fmt.Errorf("cannot set key in non-mapping node")
	}
	mappingSetExact(node, segments[len(segments)-1], value)
	return nil
}

// ============================================================
// Node utilities — unwrap, get, set, construct
// ============================================================

// unwrapDocument returns the first content node when root is a DocumentNode.
func unwrapDocument(root *yaml.Node) *yaml.Node {
	if root != nil && root.Kind == yaml.DocumentNode && len(root.Content) > 0 {
		return root.Content[0]
	}
	return root
}

// mappingGet returns the value node for key (exact string match).
// Returns nil when node is nil, not a MappingNode, or the key is absent.
func mappingGet(node *yaml.Node, key string) *yaml.Node {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i+1]
		}
	}
	return nil
}

// mappingSetExact sets or creates key=value in a MappingNode (exact key match).
func mappingSetExact(m *yaml.Node, key string, value *yaml.Node) {
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			m.Content[i+1] = value
			return
		}
	}
	m.Content = append(m.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key},
		value,
	)
}

// scalarNode creates a plain string scalar yaml.Node.
func scalarNode(value string) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value}
}

// defaultToNode converts a KeyMove Default string to a yaml.Node.
// When the string contains newlines it is treated as a YAML block and parsed
// into a mapping/sequence sub-tree so complex component definitions can be
// injected (EG-3). Plain single-line strings are returned as a scalar node.
func defaultToNode(s string) *yaml.Node {
	if !strings.Contains(s, "\n") {
		return scalarNode(s)
	}
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(s), &doc); err != nil || len(doc.Content) == 0 {
		return scalarNode(s) // fallback to scalar if YAML is not valid
	}
	return doc.Content[0] // unwrap the DocumentNode wrapper
}

// cloneNode performs a shallow copy of a scalar or mapping node sufficient for
// moving a value from one path to another without aliasing.
func cloneNode(node *yaml.Node) *yaml.Node {
	if node == nil {
		return nil
	}
	cloned := *node
	if len(node.Content) > 0 {
		cloned.Content = make([]*yaml.Node, len(node.Content))
		copy(cloned.Content, node.Content)
	}
	return &cloned
}

// ============================================================
// Path string helpers
// ============================================================

// normalizePath strips the leading "$." or "$" from a YAMLPath string.
func normalizePath(path string) string {
	path = strings.TrimPrefix(path, "$.")
	path = strings.TrimPrefix(path, "$")
	return path
}

// isSameParentRename reports whether fromPath and toPath differ only in the
// final key segment (same parent mapping). E.g.:
//
//	receivers.hostmetrics  → receivers.host_metrics   → true  (simple rename)
//	exporters.splunk_hec.batcher.x → exporters.splunk_hec.queue.y → false (cross-parent move)
func isSameParentRename(fromPath, toPath string) bool {
	fromIndex := strings.LastIndex(fromPath, ".")
	toIndex := strings.LastIndex(toPath, ".")
	if fromIndex < 0 || toIndex < 0 {
		// Top-level key — parent is the root mapping, same parent.
		return fromIndex == toIndex
	}
	return fromPath[:fromIndex] == toPath[:toIndex]
}

// splitLastSegment splits a dotted path into (leafKey, parentPath).
// e.g. "processors.filter.error_mode" → ("error_mode", "processors.filter")
// e.g. "processors.filter" → ("filter", "processors")
// e.g. "filter" → ("filter", "")
func splitLastSegment(path string) (leaf, parent string) {
	i := strings.LastIndex(path, ".")
	if i < 0 {
		return path, ""
	}
	return path[i+1:], path[:i]
}
