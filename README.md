# CSV-JSON-XML-Converter

![banner](https://i.imgur.com/PLACEHOLDER.png)

Convert data files between CSV, JSON, and XML formats with a single command and automatic format detection.

### How It Works

- Detects the input and output formats from the `.csv`, `.json`, and `.xml` file extensions.
- Reads the source file through a buffered reader so large files stream efficiently instead of loading fully into memory upfront.
- Normalizes every record into a common `[]map[string]string` structure so any pair of formats can be converted.
- Preserves column order from the first-seen keys when converting from JSON or XML back to CSV.
- Writes to the destination through a buffered writer and returns a clear message when finished.

## Setup

**Requirements**

- Go 1.21 or newer
- No external dependencies (standard library only)

**Installation**

```bash
git clone https://github.com/fantasywastaken/CSV-JSON-XML-Converter.git
cd CSV-JSON-XML-Converter
go build -o converter .
```

### Usage

CSV to JSON with pretty output:

```bash
$ converter --pretty users.csv users.json
wrote users.json
```

JSON to CSV:

```bash
$ converter data.json data.csv
wrote data.csv
```

CSV to XML using custom element names:

```bash
$ converter --pretty --xml-root=users --xml-row=user users.csv users.xml
wrote users.xml
```

Semicolon-delimited CSV to JSON:

```bash
$ converter --csv-delimiter=";" export.csv export.json
wrote export.json
```

XML to CSV:

```bash
$ converter feed.xml feed.csv
wrote feed.csv
```

### Features

- Six conversion directions between CSV, JSON, and XML.
- Automatic format detection from filename extension.
- Configurable CSV delimiter for European and tab-separated data.
- Pretty printing option for human-readable JSON and XML output.
- Custom XML root and row element names.
- Streaming read/write via `bufio` for large data sets.
- Union of keys across JSON objects so mixed schemas convert cleanly.
- Zero third-party dependencies.
