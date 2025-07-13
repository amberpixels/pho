package jsonl

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"strings"
)

// JSONCommentsCleaner is an io.Reader implementation that removes /* */ comments from the input stream.
// It comes very handy to be a pre-reader to json.Decoder() so we can easily parse .jsonl that include comments.
type JSONCommentsCleaner struct {
	reader           *bufio.Reader
	insideComment    bool
	jsonNestingLevel int
}

var _ io.Reader = &JSONCommentsCleaner{}

// NewJSONCommentsCleaner creates a new JSONCommentsCleaner with the given io.Reader as the input source.
func NewJSONCommentsCleaner(r io.Reader) *JSONCommentsCleaner {
	return &JSONCommentsCleaner{reader: bufio.NewReader(r)}
}

// Read reads data from the underlying input source and removes comments.
func (cr *JSONCommentsCleaner) Read(p []byte) (int, error) {
	var buf bytes.Buffer
	for {
		line, err := cr.reader.ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return 0, err
		}
		buf.WriteString(cr.removeComments(line))

		if errors.Is(err, io.EOF) {
			break
		}
	}

	if buf.Len() == 0 {
		return 0, io.EOF
	}

	// Copy the data from the buffer to the provided byte slice
	n, err := buf.Read(p)
	if err != nil && !errors.Is(err, io.EOF) {
		return n, err
	}

	return n, nil
}

// removeComments removes comments from a single line.
func (cr *JSONCommentsCleaner) removeComments(line string) string {
	var result strings.Builder
	for i := 0; i < len(line); i++ {
		// Check for the start of a comment block
		if line[i] == '/' && i+1 < len(line) && line[i+1] == '*' && cr.jsonNestingLevel == 0 {
			cr.insideComment = true
			continue
		}

		// Check for the end of a comment block
		if cr.insideComment && line[i] == '*' && i+1 < len(line) && line[i+1] == '/' {
			cr.insideComment = false
			i++ // Skip the closing '/'
			continue
		}

		// Update the JSON nesting level
		switch line[i] {
		case '{':
			cr.jsonNestingLevel++
		case '}':
			cr.jsonNestingLevel--
		}

		// Append the character to the result if it's not part of a comment block
		if !cr.insideComment {
			result.WriteByte(line[i])
		}
	}

	return result.String()
}
