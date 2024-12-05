# File Scanner

A high-performance Go utility for concurrent pattern matching across multiple repositories.

## Features

- Concurrent file scanning across multiple repositories
- Support for wildcard patterns in search queries
- Extension-based file filtering
- CSV report generation with match positions
- Memory-efficient processing using Go routines
- Detailed match position reporting

## Requirements

- Go 1.16 or higher
- Read/Write permissions in the execution directory

## Installation

```bash
# Clone the repository
git clone https://github.com/kur0byte/file-scanner.git
cd file-scanner

# Build the application
go build -o scanner
```

## Usage

### Basic Usage
```bash
./scanner -queriesFile queries.json -output results.csv
```

### Command Line Arguments
- `-queriesFile`: Path to JSON file containing search patterns
- `-output`: Path for the output CSV file

### Directory Structure
```
file-scanner/
├── main.go           # Main application code
├── queries.json      # Search patterns configuration
├── scanner          # Compiled binary
└── repos/           # Directory containing repositories to scan
```

### Query Configuration

Create a `queries.json` file with your search patterns:

```json
{
  "queries": [
    {
      "query": "pattern*to*search",
      "extensions": [".js", ".ts"]
    },
    {
      "query": "another*pattern",
      "extensions": [".go", ".py"]
    }
  ]
}
```

#### Pattern Support
- `*`: Matches any sequence of characters
- `?`: Matches any single character

### Output Format

The script generates a CSV file with the following columns:
- `file_path`: Path to the file containing the match
- `line_number`: Line number where the match was found
- `line_content`: Content of the matching line
- `repository_name`: Name of the repository
- `pattern`: Original search pattern that matched
- `start_index`: Starting position of the match in the line
- `end_index`: Ending position of the match in the line

## Performance

### Optimizations
- Concurrent file processing using goroutines
- Memory-efficient streaming for file reading
- Buffered CSV writing
- Efficient pattern matching using compiled regex
- Early extension filtering

### Memory Usage
- Default buffer size: 10MB per file
- Channel buffer: 10,000 results

## Examples

### Simple Pattern Search
```json
{
  "queries": [
    {
      "query": "TODO:",
      "extensions": [".go"]
    }
  ]
}
```

### Multiple Extensions
```json
{
  "queries": [
    {
      "query": "api_key*",
      "extensions": [".js", ".ts", ".go"]
    }
  ]
}
```

## Limitations

- Maximum file size: 10MB
- Extensions are case-sensitive
- Results channel buffer limited to 10,000 items

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Troubleshooting

### Common Issues

1. Permission Denied
```bash
chmod +x scanner
```

2. Memory Issues
- Reduce the number of concurrent goroutines
- Process fewer repositories at once
- Increase system memory

3. Performance Issues
- Check disk I/O capacity
- Monitor CPU usage
- Verify memory availability

## Acknowledgments

- Built with Go's standard library
- Inspired by grep and similar text search utilities
- Optimized for large-scale repository scanning