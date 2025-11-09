package sleepy

// Stats represents the data from Stats.txt
type Stats struct {
	Filename   string
	Duration   string
	Date       string
	NumSamples int
}

// Symbol represents a single entry from Symbols.txt
type Symbol struct {
	Address    string
	ModuleName string
	ProcName   string
	FilePath   string
	LineNumber int
}

// Callstack represents a single call stack entry from Callstacks.txt
type Callstack struct {
	Addresses    []uint64
	ThreadCounts map[int]float64 // Map of ThreadID to Count
}

// Thread represents a single entry from Threads.txt
type Thread struct {
	ID   int
	Name string
}

// ProfileData holds all the parsed data from a sleepy profile file
type ProfileData struct {
	Stats      Stats
	Symbols    []Symbol
	Callstacks []Callstack
	Threads    []Thread
	symbolMap  map[uint64]*Symbol // Internal map for fast symbol lookup
}

// ResolvedFrame represents a single frame in a callstack with resolved symbol information
type ResolvedFrame struct {
	Address    uint64
	Module     string
	Function   string
	SourceFile string
	LineNumber int
}

// GetDuration returns the duration for a callstack
func (cs *Callstack) GetDuration() float64 {
	for _, d := range cs.ThreadCounts {
		return d // Return first (and usually only) duration
	}
	return 0.0
}
