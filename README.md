# Very Sleepy Profiler MCP Server

Professional MCP (Model Context Protocol) server for analyzing Very Sleepy CPU profiles (.sleepy files). Built with 40 years of senior engineering wisdom.

## 🎯 Purpose

This MCP server enables AI assistants (like Claude) to perform deep performance analysis of CPU profiles, quickly identifying bottlenecks and performance issues without manual inspection.

## 🏗️ Architecture

```
verysleepy-mcp/
├── cmd/
│   └── server/          # MCP server entry point
│       └── main.go
├── internal/
│   ├── sleepy/          # Core profile parsing (no external dependencies)
│   │   ├── types.go     # Data structures
│   │   └── parser.go    # .sleepy file parser
│   └── analyzer/        # Performance analysis algorithms
│       ├── hotspots.go  # Hotspot detection
│       └── statistics.go # Statistical analysis
└── tools/               # MCP tool implementations
    ├── load_profile.go
    ├── find_hotspots.go
    ├── find_bottom_functions.go
    ├── analyze_modules.go
    ├── detect_issues.go
    ├── get_statistics.go
    └── view_callstack.go
```

### Design Principles

1. **Separation of Concerns**: Core parsing logic isolated from analysis logic
2. **No External Dependencies for Core**: Parser only uses stdlib
3. **Caching**: Loaded profiles cached for efficiency
4. **Performance**: O(n) algorithms where possible, O(n log n) for sorting
5. **Testability**: Each component independently testable

## 🛠️ Available Tools

### 1. `load_profile`
**Purpose**: Load and validate a .sleepy profile file

**Parameters**:
- `file_path` (string): Absolute path to .sleepy file

**Output**: Profile metadata (duration, samples, callstacks, etc.)

**Use Case**: Always call this first before using other tools

---

### 2. `find_hotspots` 🔥
**Purpose**: Identify functions consuming the most CPU time

**Parameters**:
- `file_path` (string): Path to loaded profile
- `top_n` (number): Number of hotspots to return (default: 10)

**Output**: Ranked list of functions with:
- Total time consumed
- Percentage of execution time
- Sample count
- Source file location

**Use Case**: **Start here!** This is your primary tool for finding what to optimize.

---

### 3. `find_bottom_functions` 🎯
**Purpose**: Find leaf functions (where actual CPU work happens)

**Parameters**:
- `file_path` (string): Path to loaded profile
- `top_n` (number): Number of functions to return (default: 10)

**Output**: Ranked list of leaf functions

**Use Case**: After identifying hotspots, use this to find the actual CPU-intensive operations to optimize. These are the functions at the bottom of callstacks doing real work.

---

### 4. `analyze_modules` 📦
**Purpose**: Break down time consumption by module/library

**Parameters**:
- `file_path` (string): Path to loaded profile

**Output**: Module-level time breakdown with percentages and visual bars

**Use Case**: Identify which components or third-party libraries are consuming resources. Useful for architectural decisions.

---

### 5. `detect_performance_issues` ⚠️
**Purpose**: Automated heuristic-based issue detection

**Parameters**:
- `file_path` (string): Path to loaded profile

**Output**: Categorized list of issues (Critical, High, Medium, Low) with:
- Issue type (CPU Hotspot, Hot Loop, Deep Call Stack, etc.)
- Impact percentage
- Affected functions

**Use Case**: Quick triage - run this to get a summary of all problems. Great starting point for analysis.

**Detection Heuristics**:
- Functions consuming >20% time → Critical
- Functions consuming >10% time → High
- Stack depth >50 frames → Deep recursion warning
- Functions in >80% of callstacks → Hot loop

---

### 6. `get_statistics` 📊
**Purpose**: Get comprehensive profile statistics

**Parameters**:
- `file_path` (string): Path to loaded profile

**Output**:
- Total execution time
- Callstack counts and depth statistics
- Unique module/function counts

**Use Case**: Get overview of profile characteristics. Useful for understanding profile scope.

---

### 7. `view_callstack` 📞
**Purpose**: View detailed callstack with resolved symbols

**Parameters**:
- `file_path` (string): Path to loaded profile
- `callstack_index` (number): Callstack index (1-based)

**Output**: Complete callstack from leaf to root with:
- Function names
- Module names
- Source locations
- Memory addresses

**Use Case**: Deep dive into specific execution paths. Useful when you know which callstack to investigate.

## 🚀 Quick Start

### Build

```bash
cd verysleepy-mcp
go build -o verysleepy-mcp.exe ./cmd/server
```

### Configure with Claude Desktop

Add to your Claude desktop config (`claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "verysleepy-profiler": {
      "command": "C:\\Users\\Admin\\Desktop\\Dev\\verySleepy\\verysleepy-mcp\\verysleepy-mcp.exe"
    }
  }
}
```

### Usage Example

1. Load a profile:
```
load_profile with file_path: "C:\path\to\profile.sleepy"
```

2. Detect issues automatically:
```
detect_performance_issues with file_path: "C:\path\to\profile.sleepy"
```

3. Find top hotspots:
```
find_hotspots with file_path: "C:\path\to\profile.sleepy", top_n: 10
```

4. Analyze module distribution:
```
analyze_modules with file_path: "C:\path\to\profile.sleepy"
```

5. Find where actual work happens:
```
find_bottom_functions with file_path: "C:\path\to\profile.sleepy", top_n: 10
```

## 📖 Analysis Workflow (Senior Engineer Approach)

### Step 1: Load and Triage (5 minutes)
```
1. load_profile
2. detect_performance_issues
3. get_statistics
```

This gives you immediate overview and identifies critical problems.

### Step 2: Identify Hotspots (10 minutes)
```
4. find_hotspots (top 10)
5. analyze_modules
```

Understand where time is spent and which components are involved.

### Step 3: Drill Down (20 minutes)
```
6. find_bottom_functions (top 10)
7. view_callstack (for interesting callstacks)
```

Find the actual functions to optimize and understand execution flow.

### Step 4: Analyze Patterns
```
- Cross-reference hotspots with bottom functions
- Check if hot functions are in your code or third-party libs
- Look for unexpected patterns (deep recursion, hot loops)
```

## 🧠 Performance Analysis Best Practices

### Understanding the Difference

- **Hotspots** (`find_hotspots`): Any function that appears in expensive callstacks. May include framework functions, entry points, etc.

- **Bottom Functions** (`find_bottom_functions`): The actual CPU-intensive leaf functions. These are what you usually want to optimize.

**Example**:
```
Hotspot: main() - 80% of time
  ↓
  ↓ (calls)
  ↓
Bottom Function: expensiveCalculation() - 80% of time
```

Both show 80%, but you want to optimize `expensiveCalculation()`, not `main()`.

### Common Patterns

1. **Hot Loop**: Same function in >80% of callstacks
   - **Fix**: Optimize the loop body or reduce iterations

2. **Expensive Library Call**: Third-party function at bottom of stack
   - **Fix**: Cache results, use faster alternative, or reduce calls

3. **Deep Recursion**: Stack depth >50
   - **Fix**: Convert to iteration or add memoization

4. **System Call Overhead**: Many small system/API calls
   - **Fix**: Batch operations

## 🔍 Example Analysis Session

```
=== Profile: game.sleepy ===

1. load_profile
   → 2.5s total, 1539 samples, 250 functions

2. detect_performance_issues
   → Critical: Physics::Update (35% of time)
   → High: Renderer::DrawSprites (18% of time)
   → Hot Loop: Vector3::Normalize (appears in 92% of callstacks)

3. find_hotspots (top 5)
   #1: Physics::Update (35%)
   #2: Renderer::DrawSprites (18%)
   #3: GameObject::Update (12%)
   #4: Vector3::Normalize (10%)
   #5: MemoryManager::Alloc (8%)

4. find_bottom_functions (top 5)
   #1: Vector3::Normalize (28%)  ← Real bottleneck!
   #2: sin() from msvcrt (12%)
   #3: malloc() (10%)
   #4: RenderSprite() (8%)
   #5: CollisionCheck() (6%)

5. Conclusion:
   - Vector3::Normalize is the real problem (called in hot loop)
   - Physics is expensive due to Vector3 operations and sin() calls
   - Rendering has memory allocation overhead (malloc in hot path)

6. Action Items:
   - Cache normalized vectors instead of recomputing
   - Pre-compute sin/cos in lookup table
   - Use memory pool instead of malloc in render path
```

## 📁 File Format Support

Supports Very Sleepy `.sleepy` files (ZIP archives containing):
- `Stats.txt` - Profile metadata
- `Symbols.txt` - Symbol table (address → function mapping)
- `Callstacks.txt` - Captured callstacks
- `Threads.txt` - Thread information (optional)
- `IPCounts.txt` - Instruction pointer counts (optional)

## 🤝 Contributing

This server follows professional software engineering practices:
- Clean architecture with clear separation
- Comprehensive error handling
- Performance-optimized algorithms
- Well-documented code
- Senior-level code quality

## 📝 License

Created for Very Sleepy Profiler analysis automation.

---

**Built with 40 years of wisdom: Measure, analyze, optimize. Never guess.**
