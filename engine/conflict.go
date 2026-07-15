package engine

import "fmt"

// DetectConflicts scans a slice of TickResults for cases where two effects
// at different ticks both match the same config key in the same file. This is
// the "narrowphase" equivalent in the State/Effect pattern — it surfaces
// potential ordering or compatibility issues before the user approves changes.
func DetectConflicts(ticks []TickResult) []Conflict {
	type seen struct {
		effect Effect
	}
	// key: "filepath::yamlpath" → first effect that touched it
	touched := make(map[string]seen)
	var conflicts []Conflict

	for _, tick := range ticks {
		for _, effect := range tick.Effects {
			key := fmt.Sprintf("%s::%s", effect.FilePath, effect.MatchedPath)
			if prior, exists := touched[key]; exists {
				if prior.effect.Rule.ID != effect.Rule.ID {
					conflicts = append(conflicts, Conflict{
						Effect1: prior.effect,
						Effect2: effect,
						Key:     key,
						Message: fmt.Sprintf(
							"Rules %s (tick %s) and %s (tick %s) both modify %q in %s. "+
								"Review migration order carefully.",
							prior.effect.Rule.ID, prior.effect.FiredAtTick,
							effect.Rule.ID, effect.FiredAtTick,
							effect.MatchedPath, effect.FilePath,
						),
					})
				}
			} else {
				touched[key] = seen{effect: effect}
			}
		}
	}
	return conflicts
}
