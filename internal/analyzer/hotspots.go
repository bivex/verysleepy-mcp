package analyzer

import (
	"fmt"
	"sort"
	"strings"

	"verysleepy-mcp/internal/sleepy"
)

// Hotspot represents a performance hotspot (function that consumes significant time)
type Hotspot struct {
	Function      string
	Module        string
	SourceFile    string
	LineNumber    int
	TotalTime     float64 // Total time spent in this function
	SampleCount   int     // Number of samples hitting this function
	Percentage    float64 // Percentage of total execution time
	CallstackRefs []int   // Indices of callstacks containing this function
}

// CallChainNode represents a node in the call chain analysis
type CallChainNode struct {
	Function   string
	Module     string
	TotalTime  float64
	SampleCount int
	Children   []*CallChainNode
}

// FindHotspots identifies the top performance bottlenecks in the profile
// Returns hotspots sorted by total time (descending)
func FindHotspots(profile *sleepy.ProfileData, topN int) []Hotspot {
	// Map: function signature -> hotspot data
	hotspotMap := make(map[string]*Hotspot)

	totalProfileTime := 0.0

	// Analyze each callstack
	for csIdx, cs := range profile.Callstacks {
		duration := cs.GetDuration()
		totalProfileTime += duration

		// Resolve callstack to get functions
		frames := profile.ResolveCallstack(&cs)

		// Count each function in the callstack
		seenInThisStack := make(map[string]bool)
		for _, frame := range frames {
			funcSig := fmt.Sprintf("%s!%s", frame.Module, frame.Function)

			// Avoid double-counting in the same stack
			if seenInThisStack[funcSig] {
				continue
			}
			seenInThisStack[funcSig] = true

			if _, exists := hotspotMap[funcSig]; !exists {
				hotspotMap[funcSig] = &Hotspot{
					Function:      frame.Function,
					Module:        frame.Module,
					SourceFile:    frame.SourceFile,
					LineNumber:    frame.LineNumber,
					TotalTime:     0,
					SampleCount:   0,
					CallstackRefs: []int{},
				}
			}

			hs := hotspotMap[funcSig]
			hs.TotalTime += duration
			hs.SampleCount++
			hs.CallstackRefs = append(hs.CallstackRefs, csIdx)
		}
	}

	// Calculate percentages and convert to slice
	hotspots := make([]Hotspot, 0, len(hotspotMap))
	for _, hs := range hotspotMap {
		if totalProfileTime > 0 {
			hs.Percentage = (hs.TotalTime / totalProfileTime) * 100.0
		}
		hotspots = append(hotspots, *hs)
	}

	// Sort by total time (descending)
	sort.Slice(hotspots, func(i, j int) bool {
		return hotspots[i].TotalTime > hotspots[j].TotalTime
	})

	// Return top N
	if topN > 0 && topN < len(hotspots) {
		return hotspots[:topN]
	}
	return hotspots
}

// FindBottomFunctions identifies leaf functions (functions at the bottom of callstacks)
// These are often the actual CPU-intensive operations
func FindBottomFunctions(profile *sleepy.ProfileData, topN int) []Hotspot {
	bottomFuncMap := make(map[string]*Hotspot)
	totalProfileTime := 0.0

	for csIdx, cs := range profile.Callstacks {
		duration := cs.GetDuration()
		totalProfileTime += duration

		frames := profile.ResolveCallstack(&cs)
		if len(frames) == 0 {
			continue
		}

		// Get the bottom (first) frame
		frame := frames[0]
		funcSig := fmt.Sprintf("%s!%s", frame.Module, frame.Function)

		if _, exists := bottomFuncMap[funcSig]; !exists {
			bottomFuncMap[funcSig] = &Hotspot{
				Function:      frame.Function,
				Module:        frame.Module,
				SourceFile:    frame.SourceFile,
				LineNumber:    frame.LineNumber,
				TotalTime:     0,
				SampleCount:   0,
				CallstackRefs: []int{},
			}
		}

		hs := bottomFuncMap[funcSig]
		hs.TotalTime += duration
		hs.SampleCount++
		hs.CallstackRefs = append(hs.CallstackRefs, csIdx)
	}

	// Calculate percentages and convert to slice
	hotspots := make([]Hotspot, 0, len(bottomFuncMap))
	for _, hs := range bottomFuncMap {
		if totalProfileTime > 0 {
			hs.Percentage = (hs.TotalTime / totalProfileTime) * 100.0
		}
		hotspots = append(hotspots, *hs)
	}

	// Sort by total time (descending)
	sort.Slice(hotspots, func(i, j int) bool {
		return hotspots[i].TotalTime > hotspots[j].TotalTime
	})

	if topN > 0 && topN < len(hotspots) {
		return hotspots[:topN]
	}
	return hotspots
}

// FindModuleHotspots groups hotspots by module
func FindModuleHotspots(profile *sleepy.ProfileData) map[string]float64 {
	moduleTime := make(map[string]float64)

	for _, cs := range profile.Callstacks {
		duration := cs.GetDuration()
		frames := profile.ResolveCallstack(&cs)

		seenModules := make(map[string]bool)
		for _, frame := range frames {
			module := frame.Module
			if module == "" || module == "?" {
				module = "[unknown]"
			}

			if seenModules[module] {
				continue
			}
			seenModules[module] = true

			moduleTime[module] += duration
		}
	}

	return moduleTime
}

// AnalyzeCallChains builds a call tree showing which functions call which
// depth: how deep to analyze (0 = unlimited)
func AnalyzeCallChains(profile *sleepy.ProfileData, depth int) map[string]*CallChainNode {
	rootFunctions := make(map[string]*CallChainNode)

	for _, cs := range profile.Callstacks {
		duration := cs.GetDuration()
		frames := profile.ResolveCallstack(&cs)

		if len(frames) == 0 {
			continue
		}

		// Start from the bottom (leaf) and build up
		var currentNode *CallChainNode

		maxDepth := len(frames)
		if depth > 0 && depth < maxDepth {
			maxDepth = depth
		}

		for i := 0; i < maxDepth; i++ {
			frame := frames[i]
			funcSig := fmt.Sprintf("%s!%s", frame.Module, frame.Function)

			if i == 0 {
				// Root level
				if _, exists := rootFunctions[funcSig]; !exists {
					rootFunctions[funcSig] = &CallChainNode{
						Function:    frame.Function,
						Module:      frame.Module,
						TotalTime:   0,
						SampleCount: 0,
						Children:    []*CallChainNode{},
					}
				}
				currentNode = rootFunctions[funcSig]
				currentNode.TotalTime += duration
				currentNode.SampleCount++
			} else {
				// Find or create child
				found := false
				for _, child := range currentNode.Children {
					childSig := fmt.Sprintf("%s!%s", child.Module, child.Function)
					if childSig == funcSig {
						child.TotalTime += duration
						child.SampleCount++
						currentNode = child
						found = true
						break
					}
				}

				if !found {
					newChild := &CallChainNode{
						Function:    frame.Function,
						Module:      frame.Module,
						TotalTime:   duration,
						SampleCount: 1,
						Children:    []*CallChainNode{},
					}
					currentNode.Children = append(currentNode.Children, newChild)
					currentNode = newChild
				}
			}
		}
	}

	return rootFunctions
}

// FormatHotspot returns a human-readable string representation of a hotspot
func FormatHotspot(hs Hotspot, rank int) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("#%d: %s!%s\n", rank, hs.Module, hs.Function))
	sb.WriteString(fmt.Sprintf("    Time: %.6f seconds (%.2f%%)\n", hs.TotalTime, hs.Percentage))
	sb.WriteString(fmt.Sprintf("    Samples: %d\n", hs.SampleCount))

	if hs.SourceFile != "" && hs.SourceFile != "[unknown]" {
		sb.WriteString(fmt.Sprintf("    Source: %s:%d\n", hs.SourceFile, hs.LineNumber))
	}

	return sb.String()
}
