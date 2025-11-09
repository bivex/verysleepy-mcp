package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"verysleepy-mcp/internal/analyzer"
	"verysleepy-mcp/internal/sleepy"
)

// Profile cache
var profileCache = make(map[string]*sleepy.ProfileData)

func main() {
	// Create MCP server
	s := server.NewMCPServer(
		"verysleepy-profiler",
		"1.0.0",
		server.WithLogging(),
	)

	// Tool 1: Load Profile
	loadProfileTool := mcp.NewTool("load_profile",
		mcp.WithDescription("Load a Very Sleepy .sleepy profile file for analysis"),
		mcp.WithString("file_path",
			mcp.Required(),
			mcp.Description("Absolute path to the .sleepy profile file"),
		),
	)

	s.AddTool(loadProfileTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		filePath, err := request.RequireString("file_path")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		profile, err := sleepy.ReadSleepyProfile(filePath)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to load profile: %v", err)), nil
		}

		profileCache[filePath] = profile

		result := fmt.Sprintf(`Profile loaded successfully!

File: %s
Duration: %s
Date: %s
Samples: %d
Callstacks: %d
Symbols: %d
Threads: %d

Use other tools to analyze this profile.
`,
			filePath,
			profile.Stats.Duration,
			profile.Stats.Date,
			profile.Stats.NumSamples,
			len(profile.Callstacks),
			len(profile.Symbols),
			len(profile.Threads),
		)

		return mcp.NewToolResultText(result), nil
	})

	// Tool 2: Find Hotspots
	findHotspotsTool := mcp.NewTool("find_hotspots",
		mcp.WithDescription("Find the top CPU hotspots (functions consuming the most time) in the profile. This is the most important tool for identifying performance bottlenecks."),
		mcp.WithString("file_path",
			mcp.Required(),
			mcp.Description("Path to the loaded .sleepy profile file"),
		),
		mcp.WithNumber("top_n",
			mcp.Description("Number of top hotspots to return (default: 10)"),
		),
	)

	s.AddTool(findHotspotsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		filePath, err := request.RequireString("file_path")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		topN := 10
		if n := request.GetFloat("top_n", 10.0); n != 10.0 {
			topN = int(n)
		}

		profile, ok := profileCache[filePath]
		if !ok {
			return mcp.NewToolResultError("Profile not loaded. Use load_profile tool first"), nil
		}

		hotspots := analyzer.FindHotspots(profile, topN)

		var sb strings.Builder
		sb.WriteString("🔥 TOP CPU HOTSPOTS (Functions Consuming Most Time)\n")
		sb.WriteString("═══════════════════════════════════════════════════\n\n")

		if len(hotspots) == 0 {
			sb.WriteString("No hotspots found.\n")
		} else {
			for i, hs := range hotspots {
				sb.WriteString(analyzer.FormatHotspot(hs, i+1))
				sb.WriteString("\n")
			}
		}

		return mcp.NewToolResultText(sb.String()), nil
	})

	// Tool 3: Find Bottom Functions
	findBottomFunctionsTool := mcp.NewTool("find_bottom_functions",
		mcp.WithDescription("Find leaf functions (functions at the bottom of callstacks - where actual CPU work happens). These are often the real performance bottlenecks to optimize."),
		mcp.WithString("file_path",
			mcp.Required(),
			mcp.Description("Path to the loaded .sleepy profile file"),
		),
		mcp.WithNumber("top_n",
			mcp.Description("Number of top functions to return (default: 10)"),
		),
	)

	s.AddTool(findBottomFunctionsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		filePath, err := request.RequireString("file_path")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		topN := 10
		if n := request.GetFloat("top_n", 10.0); n != 10.0 {
			topN = int(n)
		}

		profile, ok := profileCache[filePath]
		if !ok {
			return mcp.NewToolResultError("Profile not loaded. Use load_profile tool first"), nil
		}

		bottomFuncs := analyzer.FindBottomFunctions(profile, topN)

		var sb strings.Builder
		sb.WriteString("🎯 LEAF FUNCTIONS (Where Actual CPU Work Happens)\n")
		sb.WriteString("═══════════════════════════════════════════════════\n\n")
		sb.WriteString("These are the functions at the bottom of callstacks - the actual CPU-intensive operations.\n")
		sb.WriteString("Optimizing these will have direct performance impact.\n\n")

		if len(bottomFuncs) == 0 {
			sb.WriteString("No leaf functions found.\n")
		} else {
			for i, hs := range bottomFuncs {
				sb.WriteString(fmt.Sprintf("#%d: %s!%s\n", i+1, hs.Module, hs.Function))
				sb.WriteString(fmt.Sprintf("    Time: %.6f seconds (%.2f%%)\n", hs.TotalTime, hs.Percentage))
				sb.WriteString(fmt.Sprintf("    Samples: %d\n", hs.SampleCount))
				if hs.SourceFile != "" && hs.SourceFile != "[unknown]" {
					sb.WriteString(fmt.Sprintf("    Source: %s:%d\n", hs.SourceFile, hs.LineNumber))
				}
				sb.WriteString("\n")
			}
		}

		return mcp.NewToolResultText(sb.String()), nil
	})

	// Tool 4: Analyze Modules
	analyzeModulesTool := mcp.NewTool("analyze_modules",
		mcp.WithDescription("Analyze time spent in each module/library. Useful for identifying which components or libraries are consuming resources."),
		mcp.WithString("file_path",
			mcp.Required(),
			mcp.Description("Path to the loaded .sleepy profile file"),
		),
	)

	s.AddTool(analyzeModulesTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		filePath, err := request.RequireString("file_path")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		profile, ok := profileCache[filePath]
		if !ok {
			return mcp.NewToolResultError("Profile not loaded. Use load_profile tool first"), nil
		}

		moduleHotspots := analyzer.FindModuleHotspots(profile)

		totalTime := 0.0
		for _, time := range moduleHotspots {
			totalTime += time
		}

		type ModuleTime struct {
			Module     string
			Time       float64
			Percentage float64
		}

		modules := make([]ModuleTime, 0, len(moduleHotspots))
		for module, time := range moduleHotspots {
			pct := 0.0
			if totalTime > 0 {
				pct = (time / totalTime) * 100.0
			}
			modules = append(modules, ModuleTime{
				Module:     module,
				Time:       time,
				Percentage: pct,
			})
		}

		// Sort by time descending
		for i := 0; i < len(modules); i++ {
			for j := i + 1; j < len(modules); j++ {
				if modules[j].Time > modules[i].Time {
					modules[i], modules[j] = modules[j], modules[i]
				}
			}
		}

		var sb strings.Builder
		sb.WriteString("📦 MODULE TIME ANALYSIS\n")
		sb.WriteString("═══════════════════════════════════════════════════\n\n")

		for i, m := range modules {
			sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, m.Module))
			sb.WriteString(fmt.Sprintf("   Time: %.6f seconds (%.2f%%)\n", m.Time, m.Percentage))

			barLength := int(m.Percentage / 2)
			if barLength > 50 {
				barLength = 50
			}
			sb.WriteString("   ")
			sb.WriteString(strings.Repeat("█", barLength))
			sb.WriteString("\n\n")
		}

		return mcp.NewToolResultText(sb.String()), nil
	})

	// Tool 5: Detect Performance Issues
	detectIssuesTool := mcp.NewTool("detect_performance_issues",
		mcp.WithDescription("Automatically detect potential performance issues using heuristics. This is a great starting point for performance analysis."),
		mcp.WithString("file_path",
			mcp.Required(),
			mcp.Description("Path to the loaded .sleepy profile file"),
		),
	)

	s.AddTool(detectIssuesTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		filePath, err := request.RequireString("file_path")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		profile, ok := profileCache[filePath]
		if !ok {
			return mcp.NewToolResultError("Profile not loaded. Use load_profile tool first"), nil
		}

		issues := analyzer.DetectPerformanceIssues(profile)

		var sb strings.Builder
		sb.WriteString("⚠️  AUTOMATED PERFORMANCE ISSUE DETECTION\n")
		sb.WriteString("═══════════════════════════════════════════════════\n\n")

		if len(issues) == 0 {
			sb.WriteString("✅ No significant performance issues detected!\n")
		} else {
			critical := []analyzer.PerformanceIssue{}
			high := []analyzer.PerformanceIssue{}
			medium := []analyzer.PerformanceIssue{}
			low := []analyzer.PerformanceIssue{}

			for _, issue := range issues {
				switch issue.Severity {
				case "Critical":
					critical = append(critical, issue)
				case "High":
					high = append(high, issue)
				case "Medium":
					medium = append(medium, issue)
				case "Low":
					low = append(low, issue)
				}
			}

			if len(critical) > 0 {
				sb.WriteString("🔴 CRITICAL ISSUES:\n\n")
				for i, issue := range critical {
					sb.WriteString(fmt.Sprintf("%d. [%s] %s\n", i+1, issue.Category, issue.Description))
					if issue.Function != "" {
						sb.WriteString(fmt.Sprintf("   Function: %s!%s\n", issue.Module, issue.Function))
					}
					if issue.Impact > 0 {
						sb.WriteString(fmt.Sprintf("   Impact: %.2f%% of total time\n", issue.Impact))
					}
					sb.WriteString("\n")
				}
			}

			if len(high) > 0 {
				sb.WriteString("🟠 HIGH PRIORITY ISSUES:\n\n")
				for i, issue := range high {
					sb.WriteString(fmt.Sprintf("%d. [%s] %s\n", i+1, issue.Category, issue.Description))
					if issue.Function != "" {
						sb.WriteString(fmt.Sprintf("   Function: %s!%s\n", issue.Module, issue.Function))
					}
					if issue.Impact > 0 {
						sb.WriteString(fmt.Sprintf("   Impact: %.2f%% of total time\n", issue.Impact))
					}
					sb.WriteString("\n")
				}
			}

			sb.WriteString("\n📊 SUMMARY:\n")
			sb.WriteString(fmt.Sprintf("   Critical: %d\n", len(critical)))
			sb.WriteString(fmt.Sprintf("   High: %d\n", len(high)))
			sb.WriteString(fmt.Sprintf("   Medium: %d\n", len(medium)))
			sb.WriteString(fmt.Sprintf("   Low: %d\n", len(low)))
		}

		return mcp.NewToolResultText(sb.String()), nil
	})

	// Tool 6: Get Statistics
	getStatisticsTool := mcp.NewTool("get_statistics",
		mcp.WithDescription("Get comprehensive statistics about the profile including total time, callstack depths, unique functions/modules, etc."),
		mcp.WithString("file_path",
			mcp.Required(),
			mcp.Description("Path to the loaded .sleepy profile file"),
		),
	)

	s.AddTool(getStatisticsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		filePath, err := request.RequireString("file_path")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		profile, ok := profileCache[filePath]
		if !ok {
			return mcp.NewToolResultError("Profile not loaded. Use load_profile tool first"), nil
		}

		stats := analyzer.ComputeStatistics(profile)

		var sb strings.Builder
		sb.WriteString("📊 PROFILE STATISTICS\n")
		sb.WriteString("═══════════════════════════════════════════════════\n\n")

		sb.WriteString(fmt.Sprintf("Total Execution Time: %.6f seconds\n", stats.TotalTime))
		sb.WriteString(fmt.Sprintf("Total Callstacks: %d\n", stats.TotalCallstacks))
		sb.WriteString(fmt.Sprintf("Total Symbols: %d\n\n", stats.TotalSymbols))

		sb.WriteString("Call Stack Depth Statistics:\n")
		sb.WriteString(fmt.Sprintf("  Average: %.2f frames\n", stats.AverageStackDepth))
		sb.WriteString(fmt.Sprintf("  Maximum: %d frames\n", stats.MaxStackDepth))
		sb.WriteString(fmt.Sprintf("  Minimum: %d frames\n\n", stats.MinStackDepth))

		sb.WriteString("Unique Elements:\n")
		sb.WriteString(fmt.Sprintf("  Modules: %d\n", stats.UniqueModules))
		sb.WriteString(fmt.Sprintf("  Functions: %d\n", stats.UniqueFunctions))

		return mcp.NewToolResultText(sb.String()), nil
	})

	// Tool 7: View Callstack
	viewCallstackTool := mcp.NewTool("view_callstack",
		mcp.WithDescription("View a specific callstack with resolved function names and source locations. Useful for understanding execution flow."),
		mcp.WithString("file_path",
			mcp.Required(),
			mcp.Description("Path to the loaded .sleepy profile file"),
		),
		mcp.WithNumber("callstack_index",
			mcp.Required(),
			mcp.Description("Index of the callstack to view (1-based)"),
		),
	)

	s.AddTool(viewCallstackTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		filePath, err := request.RequireString("file_path")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		csIdx, err := request.RequireFloat("callstack_index")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		profile, ok := profileCache[filePath]
		if !ok {
			return mcp.NewToolResultError("Profile not loaded. Use load_profile tool first"), nil
		}

		index := int(csIdx) - 1

		if index < 0 || index >= len(profile.Callstacks) {
			return mcp.NewToolResultError(fmt.Sprintf("Invalid callstack index. Valid range: 1-%d", len(profile.Callstacks))), nil
		}

		cs := profile.Callstacks[index]
		duration := cs.GetDuration()
		frames := profile.ResolveCallstack(&cs)

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("📞 CALLSTACK #%d\n", int(csIdx)))
		sb.WriteString("═══════════════════════════════════════════════════\n\n")
		sb.WriteString(fmt.Sprintf("Duration: %.6f seconds\n", duration))
		sb.WriteString(fmt.Sprintf("Stack Depth: %d frames\n\n", len(frames)))

		sb.WriteString("Call Stack (bottom to top):\n\n")

		for i, frame := range frames {
			sb.WriteString(fmt.Sprintf("%d. ", i))

			if frame.Module != "" && frame.Module != "?" {
				sb.WriteString(fmt.Sprintf("%s!%s\n", frame.Module, frame.Function))
			} else {
				sb.WriteString(fmt.Sprintf("%s\n", frame.Function))
			}

			if frame.SourceFile != "" && frame.SourceFile != "[unknown]" {
				sb.WriteString(fmt.Sprintf("   %s:%d\n", frame.SourceFile, frame.LineNumber))
			}

			sb.WriteString(fmt.Sprintf("   [0x%X]\n\n", frame.Address))
		}

		return mcp.NewToolResultText(sb.String()), nil
	})

	// Start the server
	if err := server.ServeStdio(s); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
