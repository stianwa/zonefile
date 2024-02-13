// Package zonefile provides library methods for reading and parsing a
// DNS zonefile.
//
// Almost no validation is performed, and it has no knowledge of types
// (though it expects types to be in upper case), it know about the
// classes IN, CH and HS, and that is pretty much it. A pure format
// (RFC1035 chapter 5) parser.
//
// TTLs are represented by strings so they can hold TTLs with units:
// 5M, 24H, 1D etc.
package zonefile

import (
	"fmt"
	"strings"
)

// Origin represents the $ORIGIN value and comment
type Origin struct {
	// DomainName of $ORIGIN
	DomainName string

	// Comment after $ORIGIN if any
	Comment string
}

func (o *Origin) String() string {
	return "$ORIGIN " + o.DomainName + o.Comment + "\n"
}

// TTL represents the default $TTL value and comment
type TTL struct {
	// Value of TTL, this is a string to comprehend TTLs with units
	Value string
	// Comment after $TTL if any
	Comment string
}

func (t *TTL) String() string {
	return "$TTL " + t.Value + t.Comment + "\n"
}

// Include represents the $INCLUDE filename, domain name and comment
type Include struct {
	// FileName is filename to be included. Note: The parser
	// doesn't include this file, that part is left to a higher
	// echelon code.
	FileName string

	// DomainName is optional domain name for this file
	DomainName string

	// Comment after $INCLUDE if any
	Comment string
}

func (i *Include) String() string {
	domainName := ""
	if i.DomainName != "" {
		domainName = " " + i.DomainName
	}

	return "$INCLUDE " + i.FileName + domainName + i.Comment + "\n"
}

// Comment represents a comment-only line in the zonefile
type Comment struct {
	Comment string
}

func (c *Comment) String() string {
	return c.Comment + "\n"
}

// RData represents the RData part of the RR record
type RData struct {
	// RData represents the RData of a single line. Parenthesis
	// can span over multiple lines, and this RData is for a
	// single line.
	RData string

	// Comment after RData if any
	Comment string
}

// RR represents a RR record
type RR struct {
	// DomainName represents the records domain name
	DomainName string

	// TTL represents the record TTL if specified
	TTL string

	// Class represents the record Class if specified
	Class string

	// Type represents the record Type. This is a mandatory field
	// for RR-records. Note: Any single uppercase word including
	// digits is accepted by the parser.
	Type string

	// RData can span across multiple lines when using
	// parenthesis. Each entry in the slice represents a line.
	RData []*RData
}

func (r *RR) String() string {
	first := ""
	if len(r.RData) > 0 {
		first = r.RData[0].RData + r.RData[0].Comment
	}

	s := fmt.Sprintf("%-20s %-4s %-4s %-10s %s\n", r.DomainName, r.TTL, r.Class, r.Type, first)
	if len(r.RData) > 1 {
		pad := " "
		if n := strings.LastIndex(s, "("); n > -1 {
			i := 1
			for len(s) > i+n && isSpace(s[i+n]) {
				i++
			}
			pad = strings.Repeat(" ", n+i)
		}
		for _, rd := range r.RData[1:] {
			s = s + pad + rd.RData + rd.Comment + "\n"
		}
	}

	return s
}

// Entry represents an entry in the zonefile
type Entry interface {
	String() string
}

// Parse parses zonefile data into an Entry slice. One entry per
// line. Lines containing only spaces or CR are ignored and will not
// be reproduced when printing entries.
func Parse(data []byte) ([]Entry, error) {
	var entries []Entry

	rr := &RR{}
	for lineno, origLine := range strings.Split(string(data), "\n") {
		if isEmptyLine(origLine) {
			continue
		}
		line, comment := comment(origLine)
		// An RR with type means we are not finished with the
		// previous RR since we currently are within a
		// parenthesis
		if rr.Type != "" {
			line = strings.TrimSpace(line)
			rr.RData = append(rr.RData, &RData{RData: line, Comment: comment})
			if strings.HasSuffix(line, ")") {
				entries = append(entries, rr)
				rr = &RR{}
			}
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			if comment != "" {
				entries = append(entries, &Comment{Comment: comment})
			}
			continue
		}
		switch fields[0] {
		case "$ORIGIN":
			if len(fields) != 2 {
				return nil, fmt.Errorf("bad data for $ORIGIN at line %d", lineno)
			}
			entries = append(entries, &Origin{DomainName: fields[1], Comment: comment})
		case "$INCLUDE":
			if len(fields) < 2 || len(fields) > 3 {
				return nil, fmt.Errorf("bad data for $INCLUDE at line %d", lineno)
			}
			domainName := ""
			if len(fields) == 3 {
				domainName = fields[2]
			}

			entries = append(entries, &Include{FileName: fields[1], DomainName: domainName, Comment: comment})
		case "$TTL":
			if len(fields) != 2 {
				return nil, fmt.Errorf("bad data for $TTL at line %d", lineno)
			}
			entries = append(entries, &TTL{Value: fields[1], Comment: comment})
		default:
			if !isSpace(line[0]) {
				rr.DomainName = fields[0]
				fields = fields[1:]
			}
			if len(fields) == 0 {
				return nil, fmt.Errorf("bad data for RR at line %d", lineno)
			}
			// order of record TTL and class can be mixed
			if len(fields) > 2 && isDigit(fields[0][0]) {
				rr.TTL = fields[0]
				fields = fields[1:]
			}
			if len(fields) > 2 && isClass(fields[0]) {
				rr.Class = fields[0]
				fields = fields[1:]
			}
			if len(fields) > 2 && rr.Class == "" && isClass(fields[0]) {
				rr.Class = fields[0]
				fields = fields[1:]
			}

			if len(fields) < 2 {
				return nil, fmt.Errorf("bad data RDATA for RR at line %d", lineno)
			}

			if !isType(fields[0]) {
				return nil, fmt.Errorf("bad type for RR at line %d: %s", lineno, fields[0])
			}
			rr.Type = fields[0]
			fields = fields[1:]

			rdata := &RData{RData: strings.Join(fields, " "), Comment: comment}
			rr.RData = append(rr.RData, rdata)

			if strings.Contains(rdata.RData, "(") && !strings.HasSuffix(rdata.RData, ")") {
				// unclosed parenthesis, leave rr "open" for another iteration to close
				continue
			}
			entries = append(entries, rr)
			rr = &RR{}
		}
	}

	return entries, nil
}

func comment(line string) (string, string) {
	for i, r := range line {
		if r == ';' {
			if i > 0 && line[i-1] == '\\' {
				continue
			}
			// Include all spaces before comment
			for i > 0 {
				if !isSpace(line[i-1]) {
					break
				}
				i--
			}
			return line[0:i], line[i:]
		}
	}

	return line, ""
}

func isEmptyLine(line string) bool {
	return len(strings.TrimSpace(line)) == 0
}

func isSpace(b byte) bool {
	return b == ' ' || b == '\t'
}

func isDigit(b byte) bool {
	return b >= '0' && b <= '9'
}

func isClass(s string) bool {
	return s == "IN" || s == "in" || s == "CH" || s == "ch" || s == "HS" || s == "hs"
}

func isType(s string) bool {
	if isDigit(s[0]) {
		return false
	}
	for _, r := range s {
		if !((r >= '0' && r <= '9') || (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z')) {
			return false
		}
	}

	return true
}
