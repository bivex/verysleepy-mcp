package sleepy

import (
	"archive/zip"
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// ReadSleepyProfile reads a sleepy profile file (ZIP archive) and parses its contents.
func ReadSleepyProfile(filePath string) (*ProfileData, error) {
	reader, err := zip.OpenReader(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open zip file: %w", err)
	}
	defer reader.Close()

	profileData := &ProfileData{}

	for _, file := range reader.File {
		rc, err := file.Open()
		if err != nil {
			return nil, fmt.Errorf("failed to open file %s in zip: %w", file.Name, err)
		}
		defer rc.Close()

		switch file.Name {
		case "Stats.txt":
			if err := parseStats(rc, &profileData.Stats); err != nil {
				return nil, fmt.Errorf("failed to parse Stats.txt: %w", err)
			}
		case "Symbols.txt":
			if err := parseSymbols(rc, &profileData.Symbols); err != nil {
				return nil, fmt.Errorf("failed to parse Symbols.txt: %w", err)
			}
		case "Callstacks.txt":
			callstacks, err := parseCallstacks(rc)
			if err != nil {
				return nil, fmt.Errorf("failed to parse Callstacks.txt: %w", err)
			}
			profileData.Callstacks = callstacks
		case "Threads.txt":
			if err := parseThreads(rc, &profileData.Threads); err != nil {
				return nil, fmt.Errorf("failed to parse Threads.txt: %w", err)
			}
		}
	}

	// Build symbol map for fast lookup
	profileData.buildSymbolMap()

	return profileData, nil
}

// buildSymbolMap creates an internal map for fast symbol lookup by address
func (pd *ProfileData) buildSymbolMap() {
	pd.symbolMap = make(map[uint64]*Symbol)
	for i := range pd.Symbols {
		sym := &pd.Symbols[i]
		// Parse address from hex string
		if strings.HasPrefix(sym.Address, "0x") {
			if addr, err := strconv.ParseUint(strings.TrimPrefix(sym.Address, "0x"), 16, 64); err == nil {
				pd.symbolMap[addr] = sym
			}
		}
	}
}

// ResolveCallstack converts a callstack's addresses to resolved frames with symbol information
func (pd *ProfileData) ResolveCallstack(callstack *Callstack) []ResolvedFrame {
	frames := make([]ResolvedFrame, 0, len(callstack.Addresses))

	for _, addr := range callstack.Addresses {
		frame := ResolvedFrame{
			Address: addr,
		}

		// Try to find symbol for this address
		if sym, found := pd.symbolMap[addr]; found {
			frame.Module = sym.ModuleName
			frame.Function = sym.ProcName
			frame.SourceFile = sym.FilePath
			frame.LineNumber = sym.LineNumber
		} else {
			// Symbol not found, use placeholder
			frame.Module = "?"
			frame.Function = fmt.Sprintf("[0x%X]", addr)
			frame.SourceFile = ""
			frame.LineNumber = 0
		}

		frames = append(frames, frame)
	}

	return frames
}

func parseStats(r io.Reader, stats *Stats) error {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, ": ", 2)
		if len(parts) != 2 {
			continue
		}
		key := parts[0]
		value := parts[1]

		switch key {
		case "Filename":
			stats.Filename = value
		case "Duration":
			stats.Duration = value
		case "Date":
			stats.Date = value
		case "Samples":
			numSamples, err := strconv.Atoi(value)
			if err != nil {
				return fmt.Errorf("invalid Samples value: %w", err)
			}
			stats.NumSamples = numSamples
		}
	}
	return scanner.Err()
}

func parseSymbols(r io.Reader, symbols *[]Symbol) error {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		var (
			address    string
			moduleName string
			procName   string
			filePath   string
			lineNumber int
			err        error
		)

		// Custom parsing to handle quoted strings
		fields := []string{}
		inQuote := false
		currentField := strings.Builder{}
		for _, r := range line {
			if r == '"' {
				inQuote = !inQuote
				continue
			} else if r == ' ' && !inQuote {
				fields = append(fields, currentField.String())
				currentField.Reset()
			} else {
				currentField.WriteRune(r)
			}
		}
		fields = append(fields, currentField.String()) // Add the last field

		if len(fields) < 5 {
			return fmt.Errorf("malformed line in Symbols.txt: %s", line)
		}

		address = fields[0]
		moduleName = fields[1]
		procName = fields[2]
		filePath = fields[3]
		lineNumber, err = strconv.Atoi(fields[4])
		if err != nil {
			return fmt.Errorf("invalid line number in Symbols.txt: %w (line: %s)", err, line)
		}

		*symbols = append(*symbols, Symbol{
			Address:    address,
			ModuleName: moduleName,
			ProcName:   procName,
			FilePath:   filePath,
			LineNumber: lineNumber,
		})
	}
	return scanner.Err()
}

func parseCallstacks(r io.Reader) ([]Callstack, error) {
	var callstacks []Callstack
	scanner := bufio.NewScanner(r)

	callstackID := 1 // Auto-increment ID for each callstack
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" {
			continue
		}

		// Format: "Duration Address1 Address2 ..."
		// Example: "0.001584 0x7ff7ee6f157c 0x7ff7ee6fa3e4 0x7ffffce97374 0x7ffffdd5cc91"

		parts := strings.Fields(line)

		if len(parts) < 1 {
			return nil, fmt.Errorf("malformed Callstacks.txt line (no data): %q", line)
		}

		// First part is duration
		duration, err := strconv.ParseFloat(parts[0], 64)
		if err != nil {
			return nil, fmt.Errorf("invalid duration %q: %w", parts[0], err)
		}

		// Remaining parts are addresses
		addresses := make([]uint64, 0, len(parts)-1)
		for i := 1; i < len(parts); i++ {
			addrStr := parts[i]
			if !strings.HasPrefix(addrStr, "0x") {
				return nil, fmt.Errorf("invalid address format %q (expected 0x prefix)", addrStr)
			}
			addr, err := strconv.ParseUint(strings.TrimPrefix(addrStr, "0x"), 16, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid address %q: %w", addrStr, err)
			}
			addresses = append(addresses, addr)
		}

		// Create callstack with single thread count (auto-incremented ID, duration as count)
		threadCounts := map[int]float64{
			callstackID: duration,
		}

		callstacks = append(callstacks, Callstack{
			Addresses:    addresses,
			ThreadCounts: threadCounts,
		})

		callstackID++
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading Callstacks.txt: %w", err)
	}

	return callstacks, nil
}

func parseThreads(r io.Reader, threads *[]Thread) error {
	scanner := bufio.NewScanner(r)
	var currentThread Thread
	isIDLine := true

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		if isIDLine {
			threadID, err := strconv.Atoi(line)
			if err != nil {
				return fmt.Errorf("invalid thread ID in Threads.txt: %w", err)
			}
			currentThread.ID = threadID
			isIDLine = false
		} else {
			currentThread.Name = line
			*threads = append(*threads, currentThread)
			currentThread = Thread{} // Reset for next thread
			isIDLine = true
		}
	}
	return scanner.Err()
}
