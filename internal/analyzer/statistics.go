package analyzer

import (
	"fmt"
	"math"
	"sort"

	"verysleepy-mcp/internal/sleepy"
)

// ProfileStatistics contains comprehensive statistics about the profile
type ProfileStatistics struct {
	TotalTime        float64
	TotalCallstacks  int
	TotalSymbols     int
	AverageStackDepth float64
	MaxStackDepth    int
	MinStackDepth    int
	UniqueModules    int
	UniqueFunctions  int
}

// ComputeStatistics calculates comprehensive statistics for the profile
func ComputeStatistics(profile *sleepy.ProfileData) ProfileStatistics {
	stats := ProfileStatistics{}

	// Total callstacks and symbols
	stats.TotalCallstacks = len(profile.Callstacks)
	stats.TotalSymbols = len(profile.Symbols)

	if stats.TotalCallstacks == 0 {
		return stats
	}

	// Calculate time and stack depth statistics
	totalDepth := 0
	stats.MinStackDepth = math.MaxInt32
	stats.MaxStackDepth = 0

	moduleSet := make(map[string]bool)
	functionSet := make(map[string]bool)

	for _, cs := range profile.Callstacks {
		duration := cs.GetDuration()
		stats.TotalTime += duration

		depth := len(cs.Addresses)
		totalDepth += depth

		if depth > stats.MaxStackDepth {
			stats.MaxStackDepth = depth
		}
		if depth < stats.MinStackDepth {
			stats.MinStackDepth = depth
		}

		// Track unique modules and functions
		frames := profile.ResolveCallstack(&cs)
		for _, frame := range frames {
			if frame.Module != "" && frame.Module != "?" {
				moduleSet[frame.Module] = true
			}
			funcSig := fmt.Sprintf("%s!%s", frame.Module, frame.Function)
			functionSet[funcSig] = true
		}
	}

	stats.AverageStackDepth = float64(totalDepth) / float64(stats.TotalCallstacks)
	stats.UniqueModules = len(moduleSet)
	stats.UniqueFunctions = len(functionSet)

	if stats.MinStackDepth == math.MaxInt32 {
		stats.MinStackDepth = 0
	}

	return stats
}

// FunctionCallFrequency represents how often a function appears
type FunctionCallFrequency struct {
	Function string
	Module   string
	Count    int
	Percentage float64
}

// GetFunctionCallFrequencies returns functions sorted by how often they appear in callstacks
func GetFunctionCallFrequencies(profile *sleepy.ProfileData) []FunctionCallFrequency {
	funcCount := make(map[string]*FunctionCallFrequency)
	totalStacks := len(profile.Callstacks)

	for _, cs := range profile.Callstacks {
		frames := profile.ResolveCallstack(&cs)
		seen := make(map[string]bool)

		for _, frame := range frames {
			funcSig := fmt.Sprintf("%s!%s", frame.Module, frame.Function)

			if seen[funcSig] {
				continue
			}
			seen[funcSig] = true

			if _, exists := funcCount[funcSig]; !exists {
				funcCount[funcSig] = &FunctionCallFrequency{
					Function: frame.Function,
					Module:   frame.Module,
					Count:    0,
				}
			}
			funcCount[funcSig].Count++
		}
	}

	// Convert to slice and calculate percentages
	frequencies := make([]FunctionCallFrequency, 0, len(funcCount))
	for _, freq := range funcCount {
		if totalStacks > 0 {
			freq.Percentage = (float64(freq.Count) / float64(totalStacks)) * 100.0
		}
		frequencies = append(frequencies, *freq)
	}

	// Sort by count (descending)
	sort.Slice(frequencies, func(i, j int) bool {
		return frequencies[i].Count > frequencies[j].Count
	})

	return frequencies
}

// CallstackPattern represents a common callstack pattern
type CallstackPattern struct {
	Pattern     string   // Human-readable pattern
	Frames      []string // Function signatures in the pattern
	Occurrences int
	TotalTime   float64
	Percentage  float64
}

// FindCommonCallstackPatterns identifies frequently occurring callstack patterns
// depth: number of frames to consider for pattern matching (from bottom)
func FindCommonCallstackPatterns(profile *sleepy.ProfileData, depth int, topN int) []CallstackPattern {
	patterns := make(map[string]*CallstackPattern)
	totalTime := 0.0

	for _, cs := range profile.Callstacks {
		duration := cs.GetDuration()
		totalTime += duration

		frames := profile.ResolveCallstack(&cs)

		// Take bottom N frames for pattern
		patternDepth := depth
		if patternDepth > len(frames) {
			patternDepth = len(frames)
		}

		patternFrames := make([]string, patternDepth)
		for i := 0; i < patternDepth; i++ {
			frame := frames[i]
			patternFrames[i] = fmt.Sprintf("%s!%s", frame.Module, frame.Function)
		}

		patternKey := fmt.Sprintf("%v", patternFrames)

		if _, exists := patterns[patternKey]; !exists {
			patterns[patternKey] = &CallstackPattern{
				Frames:      patternFrames,
				Occurrences: 0,
				TotalTime:   0,
			}
		}

		p := patterns[patternKey]
		p.Occurrences++
		p.TotalTime += duration
	}

	// Convert to slice and calculate percentages
	patternList := make([]CallstackPattern, 0, len(patterns))
	for _, p := range patterns {
		if totalTime > 0 {
			p.Percentage = (p.TotalTime / totalTime) * 100.0
		}
		// Create human-readable pattern
		p.Pattern = fmt.Sprintf("%s", p.Frames)
		patternList = append(patternList, *p)
	}

	// Sort by total time (descending)
	sort.Slice(patternList, func(i, j int) bool {
		return patternList[i].TotalTime > patternList[j].TotalTime
	})

	if topN > 0 && topN < len(patternList) {
		return patternList[:topN]
	}
	return patternList
}

// DetectPerformanceIssues performs heuristic analysis to detect potential issues
type PerformanceIssue struct {
	Severity    string // "Critical", "High", "Medium", "Low"
	Category    string // e.g., "Deep Recursion", "Hot Loop", "Expensive Function"
	Description string
	Function    string
	Module      string
	Impact      float64 // % of total time
}

// DetectPerformanceIssues identifies potential performance problems
func DetectPerformanceIssues(profile *sleepy.ProfileData) []PerformanceIssue {
	issues := []PerformanceIssue{}
	stats := ComputeStatistics(profile)

	// Detect extremely deep callstacks (potential stack overflow or deep recursion)
	if stats.MaxStackDepth > 50 {
		issues = append(issues, PerformanceIssue{
			Severity:    "High",
			Category:    "Deep Call Stack",
			Description: fmt.Sprintf("Maximum stack depth of %d frames detected. This may indicate deep recursion or complex call chains.", stats.MaxStackDepth),
			Impact:      0,
		})
	}

	// Find functions consuming >10% of total time
	hotspots := FindHotspots(profile, 10)
	for _, hs := range hotspots {
		if hs.Percentage > 20.0 {
			issues = append(issues, PerformanceIssue{
				Severity:    "Critical",
				Category:    "CPU Hotspot",
				Description: fmt.Sprintf("Function consumes %.2f%% of total execution time", hs.Percentage),
				Function:    hs.Function,
				Module:      hs.Module,
				Impact:      hs.Percentage,
			})
		} else if hs.Percentage > 10.0 {
			issues = append(issues, PerformanceIssue{
				Severity:    "High",
				Category:    "CPU Hotspot",
				Description: fmt.Sprintf("Function consumes %.2f%% of total execution time", hs.Percentage),
				Function:    hs.Function,
				Module:      hs.Module,
				Impact:      hs.Percentage,
			})
		}
	}

	// Detect functions with very high sample counts (hot loops)
	frequencies := GetFunctionCallFrequencies(profile)
	for _, freq := range frequencies {
		if freq.Percentage > 80.0 {
			issues = append(issues, PerformanceIssue{
				Severity:    "Critical",
				Category:    "Hot Loop",
				Description: fmt.Sprintf("Function appears in %.2f%% of all callstacks - likely in a hot loop", freq.Percentage),
				Function:    freq.Function,
				Module:      freq.Module,
				Impact:      freq.Percentage,
			})
		}
	}

	// Sort by impact (descending)
	sort.Slice(issues, func(i, j int) bool {
		return issues[i].Impact > issues[j].Impact
	})

	return issues
}
