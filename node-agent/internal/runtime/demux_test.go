package runtime

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func muxFrame(streamType byte, payload string) []byte {
	header := make([]byte, 8)
	header[0] = streamType
	binary.BigEndian.PutUint32(header[4:], uint32(len(payload)))
	return append(header, []byte(payload)...)
}

func TestReadMuxFrame(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	buf.Write(muxFrame(streamStdout, "hello"))
	buf.Write(muxFrame(streamStderr, "oops"))

	streamType, payload, err := readMuxFrame(buf)
	if err != nil {
		t.Fatalf("readMuxFrame: %v", err)
	}
	if streamType != streamStdout || string(payload) != "hello" {
		t.Errorf("got streamType=%d payload=%q", streamType, payload)
	}

	streamType, payload, err = readMuxFrame(buf)
	if err != nil {
		t.Fatalf("readMuxFrame: %v", err)
	}
	if streamType != streamStderr || string(payload) != "oops" {
		t.Errorf("got streamType=%d payload=%q", streamType, payload)
	}
}

func TestReadMuxFrameEOF(t *testing.T) {
	if _, _, err := readMuxFrame(bytes.NewBuffer(nil)); err == nil {
		t.Error("expected an error reading from an empty stream")
	}
}

func TestLineSplitterCompleteLines(t *testing.T) {
	var s lineSplitter

	lines := s.Feed([]byte("hello\nworld\n"))
	if len(lines) != 2 || lines[0] != "hello" || lines[1] != "world" {
		t.Errorf("unexpected lines: %v", lines)
	}
}

func TestLineSplitterPartialLineAcrossFeeds(t *testing.T) {
	var s lineSplitter

	lines := s.Feed([]byte("hel"))
	if len(lines) != 0 {
		t.Errorf("expected no complete lines yet, got %v", lines)
	}

	lines = s.Feed([]byte("lo\nwor"))
	if len(lines) != 1 || lines[0] != "hello" {
		t.Errorf("expected [\"hello\"], got %v", lines)
	}

	lines = s.Feed([]byte("ld\n"))
	if len(lines) != 1 || lines[0] != "world" {
		t.Errorf("expected [\"world\"], got %v", lines)
	}
}

func TestLineSplitterStripsCarriageReturn(t *testing.T) {
	var s lineSplitter

	lines := s.Feed([]byte("hello\r\n"))
	if len(lines) != 1 || lines[0] != "hello" {
		t.Errorf("expected CR to be stripped, got %v", lines)
	}
}
