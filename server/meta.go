package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// Meta is what the scanner needs from a media file.
type Meta struct {
	Mime        string
	Type        string // image | video
	Width       int
	Height      int
	Orientation int
	DurationMs  int
	TakenLocal  string // "YYYY-MM-DD HH:MM:SS" wall clock, "" if unknown
	TzOffsetMin *int
	Make        string
	Model       string
	Lens        string
	Lat, Lon    *float64
}

// exifTool keeps one exiftool process alive (-stay_open) instead of paying Perl start-up per file.
type exifTool struct {
	mu  sync.Mutex
	cmd *exec.Cmd
	in  io.WriteCloser
	out *bufio.Reader
}

var exifTags = []string{
	"-MIMEType", "-ImageWidth", "-ImageHeight", "-Orientation", "-Duration",
	"-DateTimeOriginal", "-OffsetTimeOriginal", "-CreationDate", "-CreateDate",
	"-Make", "-Model", "-LensModel", "-Composite:GPSLatitude", "-Composite:GPSLongitude",
}

func startExifTool() (*exifTool, error) {
	cmd := exec.Command("exiftool", "-stay_open", "True", "-@", "-")
	in, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	out, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("exiftool: %w (install libimage-exiftool-perl)", err)
	}
	return &exifTool{cmd: cmd, in: in, out: bufio.NewReaderSize(out, 64<<10)}, nil
}

func (e *exifTool) Close() {
	e.mu.Lock()
	defer e.mu.Unlock()
	io.WriteString(e.in, "-stay_open\nFalse\n")
	e.in.Close()
	e.cmd.Wait()
}

// Read returns metadata for each path, in order. A file exiftool can't read gets a zero Meta.
func (e *exifTool) Read(paths []string) ([]Meta, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	var b strings.Builder
	// -n: numbers not prose; QuickTimeUTC: video CreateDate is UTC, convert to the server's local time.
	b.WriteString("-json\n-n\n-fast\n-charset\nfilename=utf8\n-api\nQuickTimeUTC=1\n-api\nLargeFileSupport=1\n")
	for _, t := range exifTags {
		b.WriteString(t + "\n")
	}
	for _, p := range paths {
		if strings.ContainsAny(p, "\n\r") {
			p = "" // can't be passed through the argfile; exiftool reports it missing
		}
		b.WriteString(p + "\n")
	}
	b.WriteString("-execute\n")
	if _, err := io.WriteString(e.in, b.String()); err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	for {
		line, err := e.out.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("exiftool output: %w", err)
		}
		if strings.TrimSpace(line) == "{ready}" {
			break
		}
		buf.WriteString(line)
	}
	var raw []map[string]any
	if buf.Len() > 0 {
		if err := json.Unmarshal(buf.Bytes(), &raw); err != nil {
			return nil, fmt.Errorf("exiftool json: %w", err)
		}
	}
	bySource := make(map[string]map[string]any, len(raw))
	for _, r := range raw {
		if s, ok := r["SourceFile"].(string); ok {
			bySource[s] = r
		}
	}
	res := make([]Meta, len(paths))
	for i, p := range paths {
		if r, ok := bySource[p]; ok {
			res[i] = parseMeta(r)
		}
	}
	return res, nil
}

func parseMeta(r map[string]any) Meta {
	m := Meta{
		Mime:        str(r["MIMEType"]),
		Width:       num(r["ImageWidth"]),
		Height:      num(r["ImageHeight"]),
		Orientation: num(r["Orientation"]),
		Make:        strings.TrimSpace(str(r["Make"])),
		Model:       strings.TrimSpace(str(r["Model"])),
		Lens:        strings.TrimSpace(str(r["LensModel"])),
	}
	if strings.HasPrefix(m.Mime, "video/") {
		m.Type = "video"
	} else {
		m.Type = "image"
	}
	if d, ok := r["Duration"].(float64); ok {
		m.DurationMs = int(d * 1000)
	}
	if m.Orientation >= 5 && m.Orientation <= 8 { // rotated 90/270: display size is swapped
		m.Width, m.Height = m.Height, m.Width
	}
	// Photos: DateTimeOriginal is local wall clock. Videos: CreationDate carries an offset; CreateDate is
	// UTC, already converted to server-local by QuickTimeUTC.
	for _, k := range []string{"DateTimeOriginal", "CreationDate", "CreateDate"} {
		if local, off, ok := parseExifDate(str(r[k])); ok {
			m.TakenLocal, m.TzOffsetMin = local, off
			break
		}
	}
	if m.TzOffsetMin == nil {
		if o, ok := parseOffset(str(r["OffsetTimeOriginal"])); ok {
			m.TzOffsetMin = &o
		}
	}
	lat, latOk := r["GPSLatitude"].(float64)
	lon, lonOk := r["GPSLongitude"].(float64)
	if latOk && lonOk && !(lat == 0 && lon == 0) {
		m.Lat, m.Lon = &lat, &lon
	}
	return m
}

// parseExifDate turns "2024:05:01 10:00:00[.123][+02:00]" into "2024-05-01 10:00:00" and an optional offset.
func parseExifDate(s string) (string, *int, bool) {
	if len(s) < 19 {
		return "", nil, false
	}
	t, err := time.Parse("2006:01:02 15:04:05", s[:19])
	if err != nil || t.Year() < 1900 { // "0000:00:00 00:00:00" is common junk
		return "", nil, false
	}
	rest := strings.TrimLeft(s[19:], ".0123456789")
	if o, ok := parseOffset(rest); ok {
		return t.Format("2006-01-02 15:04:05"), &o, true
	}
	return t.Format("2006-01-02 15:04:05"), nil, true
}

func parseOffset(s string) (int, bool) {
	s = strings.TrimSpace(s)
	if s == "Z" {
		return 0, true
	}
	if len(s) != 6 || (s[0] != '+' && s[0] != '-') || s[3] != ':' {
		return 0, false
	}
	t, err := time.Parse("15:04", s[1:])
	if err != nil {
		return 0, false
	}
	o := t.Hour()*60 + t.Minute()
	if s[0] == '-' {
		o = -o
	}
	return o, true
}

func str(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case float64:
		return fmt.Sprint(x)
	}
	return ""
}

func num(v any) int {
	if f, ok := v.(float64); ok {
		return int(f)
	}
	return 0
}
