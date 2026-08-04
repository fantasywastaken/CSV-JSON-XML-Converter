package main

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type config struct {
	src         string
	dst         string
	delimiter   string
	pretty      bool
	rootElement string
	rowElement  string
}

func main() {
	delim := flag.String("csv-delimiter", ",", "CSV delimiter character")
	pretty := flag.Bool("pretty", false, "Pretty print output (JSON and XML)")
	rootEl := flag.String("xml-root", "root", "XML root element name when writing XML")
	rowEl := flag.String("xml-row", "row", "XML row element name when writing XML")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: converter [flags] <input_file> <output_file>")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Convert between CSV, JSON, and XML. Formats are auto-detected")
		fmt.Fprintln(os.Stderr, "from the .csv, .json, and .xml file extensions.")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Flags:")
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() != 2 {
		flag.Usage()
		os.Exit(2)
	}

	cfg := config{
		src:         flag.Arg(0),
		dst:         flag.Arg(1),
		delimiter:   *delim,
		pretty:      *pretty,
		rootElement: *rootEl,
		rowElement:  *rowEl,
	}

	if err := convert(cfg); err != nil {
		fmt.Fprintln(os.Stderr, "converter:", err)
		os.Exit(1)
	}
	fmt.Printf("wrote %s\n", cfg.dst)
}

func convert(cfg config) error {
	srcFormat := detectFormat(cfg.src)
	dstFormat := detectFormat(cfg.dst)
	if srcFormat == "" {
		return fmt.Errorf("unable to detect input format from %s", cfg.src)
	}
	if dstFormat == "" {
		return fmt.Errorf("unable to detect output format from %s", cfg.dst)
	}

	header, rows, err := loadRows(cfg.src, srcFormat, cfg)
	if err != nil {
		return err
	}
	return writeRows(cfg.dst, dstFormat, header, rows, cfg)
}

func detectFormat(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".csv":
		return "csv"
	case ".json":
		return "json"
	case ".xml":
		return "xml"
	}
	return ""
}

func loadRows(path, format string, cfg config) ([]string, []map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()

	switch format {
	case "csv":
		return readCSV(f, cfg.delimiter)
	case "json":
		return readJSON(f)
	case "xml":
		return readXML(f)
	}
	return nil, nil, fmt.Errorf("unsupported input format: %s", format)
}

func readCSV(r io.Reader, delim string) ([]string, []map[string]string, error) {
	reader := csv.NewReader(bufio.NewReader(r))
	if len(delim) > 0 {
		reader.Comma = rune(delim[0])
	}
	reader.FieldsPerRecord = -1

	var header []string
	var rows []map[string]string
	first := true
	for {
		fields, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, nil, fmt.Errorf("csv: %w", err)
		}
		if first {
			header = append([]string{}, fields...)
			first = false
			continue
		}
		row := make(map[string]string, len(header))
		for i, v := range fields {
			if i < len(header) {
				row[header[i]] = v
			} else {
				key := fmt.Sprintf("col%d", i+1)
				row[key] = v
				header = append(header, key)
			}
		}
		rows = append(rows, row)
	}
	return header, rows, nil
}

func readJSON(r io.Reader) ([]string, []map[string]string, error) {
	var list []map[string]any
	dec := json.NewDecoder(r)
	if err := dec.Decode(&list); err != nil {
		return nil, nil, fmt.Errorf("expected top-level JSON array of objects: %w", err)
	}

	seen := make(map[string]bool)
	var header []string
	rows := make([]map[string]string, 0, len(list))
	for _, obj := range list {
		row := make(map[string]string, len(obj))
		for k, v := range obj {
			if !seen[k] {
				seen[k] = true
				header = append(header, k)
			}
			row[k] = anyToString(v)
		}
		rows = append(rows, row)
	}
	return header, rows, nil
}

func anyToString(v any) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case float64:
		return trimFloat(val)
	case bool:
		if val {
			return "true"
		}
		return "false"
	default:
		buf, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("%v", v)
		}
		return string(buf)
	}
}

func trimFloat(f float64) string {
	s := fmt.Sprintf("%g", f)
	return s
}

func readXML(r io.Reader) ([]string, []map[string]string, error) {
	dec := xml.NewDecoder(bufio.NewReader(r))
	var rows []map[string]string
	var current map[string]string
	var currentKey string
	var depth int
	var header []string
	seen := make(map[string]bool)

	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, nil, fmt.Errorf("xml: %w", err)
		}
		switch t := tok.(type) {
		case xml.StartElement:
			depth++
			switch depth {
			case 2:
				current = make(map[string]string)
				for _, attr := range t.Attr {
					current[attr.Name.Local] = attr.Value
					if !seen[attr.Name.Local] {
						seen[attr.Name.Local] = true
						header = append(header, attr.Name.Local)
					}
				}
			case 3:
				currentKey = t.Name.Local
			}
		case xml.EndElement:
			if depth == 2 && current != nil {
				rows = append(rows, current)
				current = nil
			}
			if depth == 3 {
				currentKey = ""
			}
			depth--
		case xml.CharData:
			if depth == 3 && currentKey != "" && current != nil {
				val := strings.TrimSpace(string(t))
				if val != "" {
					current[currentKey] = val
					if !seen[currentKey] {
						seen[currentKey] = true
						header = append(header, currentKey)
					}
				}
			}
		}
	}
	return header, rows, nil
}

func writeRows(path, format string, header []string, rows []map[string]string, cfg config) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	defer w.Flush()

	switch format {
	case "csv":
		return writeCSV(w, header, rows, cfg.delimiter)
	case "json":
		return writeJSON(w, header, rows, cfg.pretty)
	case "xml":
		return writeXML(w, header, rows, cfg.rootElement, cfg.rowElement, cfg.pretty)
	}
	return fmt.Errorf("unsupported output format: %s", format)
}

func writeCSV(w io.Writer, header []string, rows []map[string]string, delim string) error {
	cw := csv.NewWriter(w)
	if len(delim) > 0 {
		cw.Comma = rune(delim[0])
	}
	if err := cw.Write(header); err != nil {
		return err
	}
	for _, row := range rows {
		fields := make([]string, len(header))
		for i, h := range header {
			fields[i] = row[h]
		}
		if err := cw.Write(fields); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

func writeJSON(w io.Writer, header []string, rows []map[string]string, pretty bool) error {
	out := make([]map[string]string, 0, len(rows))
	for _, r := range rows {
		obj := make(map[string]string, len(header))
		for _, h := range header {
			obj[h] = r[h]
		}
		out = append(out, obj)
	}
	enc := json.NewEncoder(w)
	if pretty {
		enc.SetIndent("", "  ")
	}
	return enc.Encode(out)
}

func writeXML(w io.Writer, header []string, rows []map[string]string, root, row string, pretty bool) error {
	if root == "" {
		root = "root"
	}
	if row == "" {
		row = "row"
	}
	nl, indent, inner := "", "", ""
	if pretty {
		nl, indent, inner = "\n", "  ", "    "
	}
	if _, err := fmt.Fprintf(w, "<?xml version=\"1.0\" encoding=\"UTF-8\"?>%s", nl); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "<%s>%s", root, nl); err != nil {
		return err
	}
	for _, r := range rows {
		if _, err := fmt.Fprintf(w, "%s<%s>%s", indent, row, nl); err != nil {
			return err
		}
		for _, h := range header {
			esc := escapeXML(r[h])
			if _, err := fmt.Fprintf(w, "%s<%s>%s</%s>%s", inner, h, esc, h, nl); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintf(w, "%s</%s>%s", indent, row, nl); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintf(w, "</%s>%s", root, nl)
	return err
}

func escapeXML(s string) string {
	var b strings.Builder
	if err := xml.EscapeText(&b, []byte(s)); err != nil {
		return s
	}
	return b.String()
}
