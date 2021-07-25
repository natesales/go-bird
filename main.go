package bird

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"regexp"
	"strings"
	"time"
)

// isNumeric checks if a byte is character for number
func isNumeric(b byte) bool {
	return b >= byte('0') && b <= byte('9')
}

// trimDupSpace trims duplicate whitespace
func trimDupSpace(s string) string {
	headTailWhitespace := regexp.MustCompile(`^[\s\p{Zs}]+|[\s\p{Zs}]+$`)
	innerWhitespace := regexp.MustCompile(`[\s\p{Zs}]{2,}`)
	return innerWhitespace.ReplaceAllString(headTailWhitespace.ReplaceAllString(s, ""), " ")
}

// Daemon stores a BIRD socket connection
type Daemon struct {
	Conn net.Conn
}

// Protocol stores a BIRD protocol
type Protocol struct {
	Name  string
	Proto string
	Table string
	State string
	Since time.Time
	Info  string
}

// New returns a new Daemon
func New(socket string) (*Daemon, error) {
	conn, err := net.Dial("unix", socket)
	if err != nil {
		return nil, err
	}
	return &Daemon{Conn: conn}, nil
}

// Close closes the BIRD socket connection
func (d *Daemon) Close() error {
	return d.Conn.Close()
}

// Read a line from bird socket, removing preceding status number, output it. Returns if there are more lines.
func (d *Daemon) Read(w io.Writer) bool {
	// Read from socket byte by byte, until reaching newline character
	c := make([]byte, 1024, 1024)
	pos := 0
	for {
		if pos >= 1024 {
			break
		}
		_, err := d.Conn.Read(c[pos : pos+1])
		if err != nil {
			panic(err)
		}
		if c[pos] == byte('\n') {
			break
		}
		pos++
	}

	c = c[:pos+1]

	// Remove preceding status numbers
	if pos > 4 && isNumeric(c[0]) && isNumeric(c[1]) && isNumeric(c[2]) && isNumeric(c[3]) {
		// There is a status number at beginning, remove it (first 5 bytes)
		if w != nil && pos > 6 {
			pos = 5
			if _, err := w.Write(c[pos:]); err != nil {
				panic(err)
			}
		}
		return c[0] != byte('0') && c[0] != byte('8') && c[0] != byte('9')
	} else {
		if w != nil {
			if _, err := w.Write(c[1:]); err != nil {
				panic(err)
			}
		}
		return true
	}
}

// ReadString reads the full BIRD response as a string
func (d *Daemon) ReadString() (string, error) {
	var buf bytes.Buffer
	for d.Read(&buf) {
	}
	if r := recover(); r != nil {
		return "", fmt.Errorf("%s", r)
	}
	return buf.String(), nil
}

// Write a command to BIRD
func (d *Daemon) Write(command string) {
	d.Conn.Write([]byte(strings.TrimRight(command, "\n") + "\n"))
}

// Protocols gets a slice of parsed protocols
func (d *Daemon) Protocols() ([]Protocol, error) {
	d.Write("show protocols")
	protocolsString, err := d.ReadString()
	if err != nil {
		return nil, err
	}

	var protocols []Protocol
	for _, line := range strings.Split(strings.TrimSuffix(protocolsString, "\n"), "\n") {
		line = trimDupSpace(line)
		// Skip header
		if !(strings.Contains(line, "Name Proto Table") || strings.Contains(line, "ready.")) {
			parts := strings.Split(line, " ")
			timeVal, err := time.Parse("2006-01-02 15:04:05", parts[4]+" "+parts[5])
			if err != nil {
				return nil, err
			}
			protocols = append(protocols, Protocol{
				Name:  parts[0],
				Proto: parts[1],
				Table: parts[2],
				State: parts[3],
				Since: timeVal,
				Info:  strings.Join(parts[6:], " "),
			})
		}
	}
	return protocols, nil
}
