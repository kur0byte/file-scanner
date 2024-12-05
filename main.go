// Package main implements a high-performance file scanner for searching patterns
// across multiple code repositories.
//
// Features:
//   - Concurrent file processing
//   - Wildcard pattern matching
//   - CSV report generation
//   - Support for multiple file extensions
//   - Character position tracking for matches
//
// Usage:
//
//	file-scanner -queriesFile queries.json -output results.csv
//
// The queries.json file should contain search patterns and file extensions:
//
//	{
//	  "queries": [
//	    {
//	      "query": "*pattern*",
//	      "extensions": [".js", ".ts"]
//	    }
//	  ]
//	}
package main

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"time"
)

// Query represents a search pattern and its associated file extensions.
type Query struct {
	Pattern    string   `json:"query"`      // The search pattern, supports wildcards (* and ?)
	Extensions []string `json:"extensions"` // List of file extensions to search in
}

// QueriesFile represents the structure of the JSON input file containing search queries.
type QueriesFile struct {
	Queries []Query `json:"queries"` // Array of search queries
}

// MatchInfo contains information about a pattern match in a line of text.
type MatchInfo struct {
	Pattern    string // Original search pattern that matched
	StartIndex int    // Starting character position of the match
	EndIndex   int    // Ending character position of the match
}

// SearchResult represents a single match found during the search process.
type SearchResult struct {
	FilePath   string    // Full path to the file containing the match
	LineNumber int       // Line number where the match was found
	LineText   string    // Content of the line containing the match
	Repository string    // Name of the repository containing the file
	MatchInfo  MatchInfo // Information about the match
}

// ResultChannel is a channel type for passing search results between goroutines.
type ResultChannel chan SearchResult

// findMatchPositions searches for all matches of a pattern in a line of text
// and returns their positions.
func findMatchPositions(line string, pattern *regexp.Regexp, originalPattern string) []MatchInfo {
	matches := pattern.FindAllStringSubmatchIndex(line, -1)
	var results []MatchInfo

	for _, match := range matches {
		if len(match) >= 2 {
			results = append(results, MatchInfo{
				Pattern:    originalPattern,
				StartIndex: match[0],
				EndIndex:   match[1],
			})
		}
	}

	return results
}

// compilePatterns converts the query patterns into regular expressions and creates
// a map of valid file extensions. It returns the compiled patterns, original patterns,
// and a map of extensions.
func compilePatterns(queries []Query) ([]*regexp.Regexp, []string, map[string]bool) {
	patterns := make([]*regexp.Regexp, 0, len(queries))
	originalPatterns := make([]string, 0, len(queries))
	extensions := make(map[string]bool)

	for _, q := range queries {
		// Convert wildcard pattern to regex
		pattern := regexp.QuoteMeta(q.Pattern)
		pattern = regexp.MustCompile(`\\\*`).ReplaceAllString(pattern, ".*")
		pattern = regexp.MustCompile(`\\\?`).ReplaceAllString(pattern, ".")

		re := regexp.MustCompile(pattern)
		patterns = append(patterns, re)
		originalPatterns = append(originalPatterns, q.Pattern)

		// Build extensions map
		for _, ext := range q.Extensions {
			extensions[ext] = true
		}
	}

	return patterns, originalPatterns, extensions
}

// scanFile reads a file line by line and searches for pattern matches.
// Matches are sent through the resultChan channel.
func scanFile(filePath string, patterns []*regexp.Regexp, originalPatterns []string, repository string, resultChan ResultChannel) {
	file, err := os.Open(filePath)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 1024*1024), 10*1024*1024) // 10MB buffer

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		for i, pattern := range patterns {
			matches := findMatchPositions(line, pattern, originalPatterns[i])
			for _, match := range matches {
				resultChan <- SearchResult{
					FilePath:   filePath,
					LineNumber: lineNum,
					LineText:   line,
					Repository: repository,
					MatchInfo:  match,
				}
			}
		}
	}
}

// walkRepository traverses a repository directory and processes each file that
// matches the specified extensions. It uses goroutines for concurrent processing.
func walkRepository(
	repoPath string,
	repository string,
	patterns []*regexp.Regexp,
	originalPatterns []string,
	extensions map[string]bool,
	resultChan ResultChannel,
	wg *sync.WaitGroup,
) {
	filepath.Walk(repoPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if !info.IsDir() && extensions[filepath.Ext(path)] {
			wg.Add(1)
			go func() {
				defer wg.Done()
				scanFile(path, patterns, originalPatterns, repository, resultChan)
			}()
		}
		return nil
	})
}

// writeResults writes the search results to a CSV file in the specified format.
func writeResults(results []SearchResult, outputFile string) error {
	file, err := os.Create(outputFile)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(bufio.NewWriter(file))
	defer writer.Flush()

	// Write header
	writer.Write([]string{
		"file_path",
		"line_number",
		"line_content",
		"repository_name",
		"pattern",
		"start_index",
		"end_index",
	})

	// Write results
	for _, result := range results {
		err := writer.Write([]string{
			result.FilePath,
			fmt.Sprintf("%d", result.LineNumber),
			result.LineText,
			result.Repository,
			result.MatchInfo.Pattern,
			fmt.Sprintf("%d", result.MatchInfo.StartIndex),
			fmt.Sprintf("%d", result.MatchInfo.EndIndex),
		})
		if err != nil {
			return err
		}
	}

	return nil
}

// main is the entry point of the application. It handles command-line arguments,
// initializes the search process, and coordinates the result collection and output.
func main() {
	// Parse command-line flags
	queriesFile := flag.String("queriesFile", "", "Path to queries JSON file")
	outputFile := flag.String("output", "", "Path to output CSV file")
	flag.Parse()

	if *queriesFile == "" || *outputFile == "" {
		fmt.Println("Both queriesFile and output flags are required")
		os.Exit(1)
	}

	startTime := time.Now()

	// Read and parse queries file
	data, err := os.ReadFile(*queriesFile)
	if err != nil {
		fmt.Printf("Error reading queries file: %v\n", err)
		os.Exit(1)
	}

	var queries QueriesFile
	if err := json.Unmarshal(data, &queries); err != nil {
		fmt.Printf("Error parsing queries file: %v\n", err)
		os.Exit(1)
	}

	patterns, originalPatterns, extensions := compilePatterns(queries.Queries)

	// Get repositories
	reposDir := filepath.Join(".", "repos")
	repositories, err := os.ReadDir(reposDir)
	if err != nil {
		fmt.Printf("Error reading repos directory: %v\n", err)
		os.Exit(1)
	}

	// Create channels for results and synchronization
	resultChan := make(ResultChannel, 10000) // Buffered channel
	var wg sync.WaitGroup

	// Launch repository scanning goroutines
	go func() {
		for _, repo := range repositories {
			if repo.IsDir() {
				repoPath := filepath.Join(reposDir, repo.Name())
				fmt.Printf("Scanning repository: %s\n", repo.Name())
				walkRepository(repoPath, repo.Name(), patterns, originalPatterns, extensions, resultChan, &wg)
			}
		}
		wg.Wait()
		close(resultChan)
	}()

	// Collect results
	var results []SearchResult
	for result := range resultChan {
		results = append(results, result)
	}

	// Write results to CSV
	if err := writeResults(results, *outputFile); err != nil {
		fmt.Printf("Error writing results: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Execution completed in %v\n", time.Since(startTime))
	fmt.Printf("Total results: %d\n", len(results))
}
