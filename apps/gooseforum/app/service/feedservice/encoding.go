package feedservice

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// Compact tuples keep a complete 300-candidate sample under 32 KiB without
// dropping candidates or features. The tuple version is part of the algorithm.
func (c Candidate) MarshalJSON() ([]byte, error) {
	flags := 0
	if c.Repeat {
		flags |= 1
	}
	if c.Explore {
		flags |= 2
	}
	if c.SoftFiltered {
		flags |= 4
	}
	if c.Fallback {
		flags |= 8
	}
	return json.Marshal([]any{c.ID, c.Author, c.Sources, c.Features, flags, c.Base, c.Adjusted, c.Pool, c.Team, c.Reason})
}
func (c *Candidate) UnmarshalJSON(data []byte) error {
	var values []json.RawMessage
	if err := json.Unmarshal(data, &values); err != nil {
		return err
	}
	if len(values) != 10 {
		return fmt.Errorf("unsupported candidate tuple")
	}
	flags := 0
	targets := []any{&c.ID, &c.Author, &c.Sources, &c.Features, &flags, &c.Base, &c.Adjusted, &c.Pool, &c.Team, &c.Reason}
	for i, dst := range targets {
		if err := json.Unmarshal(values[i], dst); err != nil {
			return err
		}
	}
	c.Repeat = flags&1 != 0
	c.Explore = flags&2 != 0
	c.SoftFiltered = flags&4 != 0
	c.Fallback = flags&8 != 0
	return nil
}

// Served pages are compressed to keep a full 20-item page within the 4 KiB
// ordinary queue budget. Decoding also accepts old/plain fixture records.
func encodeServed(items []Candidate) (string, error) {
	data, err := json.Marshal(items)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	writer := zlib.NewWriter(&buf)
	if _, err = writer.Write(data); err != nil {
		return "", err
	}
	if err = writer.Close(); err != nil {
		return "", err
	}
	return "z1:" + base64.RawStdEncoding.EncodeToString(buf.Bytes()), nil
}
func decodeServed(data string) ([]Candidate, error) {
	raw := []byte(data)
	if strings.HasPrefix(data, "z1:") {
		encoded, err := base64.RawStdEncoding.DecodeString(data[3:])
		if err != nil {
			return nil, err
		}
		reader, err := zlib.NewReader(bytes.NewReader(encoded))
		if err != nil {
			return nil, err
		}
		defer func() { _ = reader.Close() }()
		raw, err = io.ReadAll(io.LimitReader(reader, 128<<10))
		if err != nil {
			return nil, err
		}
	}
	var items []Candidate
	err := json.Unmarshal(raw, &items)
	return items, err
}
