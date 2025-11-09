# Quick Start Guide

## Installation

### 1. Build (Already Done!)

The MCP server is already built: `verysleepy-mcp.exe` (6.9 MB)

If you need to rebuild:
```bash
go build -o verysleepy-mcp.exe ./cmd/server
```

### 2. Configure Claude Desktop

Add to your `claude_desktop_config.json`:

**Location**: `%APPDATA%\Claude\claude_desktop_config.json`

```json
{
  "mcpServers": {
    "verysleepy-profiler": {
      "command": "C:\\Users\\Admin\\Desktop\\Dev\\verySleepy\\verysleepy-mcp\\verysleepy-mcp.exe"
    }
  }
}
```

### 3. Restart Claude Desktop

Close and reopen Claude Desktop to load the MCP server.

## First Analysis (30 Second Workflow)

### Example with Your Test File

```
You: Analyze the profile at "C:\Program Files\PureDevSoftware\FramePro\Samples\FrameProExample\x64\Release\capture.sleepy"

Claude will:
1. load_profile
2. detect_performance_issues
3. find_hotspots
4. find_bottom_functions

Result: You'll get:
- List of critical issues
- Top 10 CPU hotspots
- Leaf functions to optimize
- Recommendations
```

## Available Tools

| Tool | Purpose | When to Use |
|------|---------|-------------|
| `load_profile` | Load .sleepy file | **Always first** |
| `detect_performance_issues` | Auto-detect problems | **Best starting point** |
| `find_hotspots` | Find expensive functions | Identify bottlenecks |
| `find_bottom_functions` | Find leaf functions | **What to actually optimize** |
| `analyze_modules` | Module breakdown | Component analysis |
| `get_statistics` | Profile stats | Overview |
| `view_callstack` | Detailed callstack | Deep dive |

## Recommended Workflow

### Quick Triage (2 minutes)
```
1. load_profile
2. detect_performance_issues
```

### Full Analysis (10 minutes)
```
1. load_profile
2. detect_performance_issues
3. find_hotspots (top 10)
4. find_bottom_functions (top 10)
5. analyze_modules
```

### Deep Dive (30 minutes)
```
1-5. Same as above
6. view_callstack (for interesting patterns)
7. Cross-reference findings
```

## Tips

### Understanding the Output

- **Hotspots**: Functions that appear in expensive call paths (may include framework code)
- **Bottom Functions**: **Actual** CPU-intensive work (what you should optimize)
- **Critical Issues**: >20% of execution time
- **High Issues**: >10% of execution time

### What to Optimize First

1. **Bottom functions** with >10% time
2. Functions in **your** code (not third-party)
3. Functions with **high sample counts** (hot loops)

### Common Patterns

- **🔴 Hot Loop**: Function in >80% of callstacks → Optimize loop body
- **🔴 Expensive Leaf**: Bottom function >20% → Direct optimization target
- **🟠 Deep Stack**: >50 frames → Check for recursion or simplify
- **🟠 System Calls**: malloc/free in hot path → Use memory pools

## Example Session

```
User: Load and analyze capture.sleepy

Claude: I'll analyze the profile...

[runs load_profile]
Profile loaded: 2.734s total, 1539 samples

[runs detect_performance_issues]
🔴 CRITICAL ISSUES:
1. [CPU Hotspot] Vector3::Normalize (35%)
2. [Hot Loop] Function appears in 92% of callstacks

[runs find_bottom_functions]
Top leaf functions:
#1: Vector3::Normalize - 0.876s (35%)
#2: sin() - 0.342s (12%)
#3: malloc() - 0.243s (10%)

RECOMMENDATION:
- Cache normalized vectors (eliminate 35% overhead)
- Pre-compute sin/cos in lookup table (12% savings)
- Use memory pool for small allocations (10% savings)

Potential speedup: 2-3x
```

## Troubleshooting

**Q: Tools not appearing in Claude?**
- Restart Claude Desktop
- Check config file path
- Verify .exe path is absolute

**Q: Profile won't load?**
- Check file path (use absolute paths)
- Verify .sleepy file isn't corrupted
- Ensure file exists

**Q: Getting "Profile not loaded" error?**
- Always run `load_profile` first
- Use exact same file path for all tools

## Advanced Usage

### Custom Analysis

Ask Claude:
- "Which functions should I optimize first?"
- "Find all functions consuming >5% of time"
- "Analyze callstacks containing function X"
- "Compare module time distribution"

### Pattern Detection

Claude can identify:
- Repeated patterns in callstacks
- Unexpected hot paths
- Memory allocation overhead
- System call bottlenecks

---

**Ready to find your bottlenecks? Ask Claude to analyze your profile!**
