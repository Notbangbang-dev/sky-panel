package runtime

import (
	"encoding/binary"
	"fmt"
	"io"
)

const (
	streamStdin  = 0
	streamStdout = 1
	streamStderr = 2
)

// readMuxFrame reads one Docker-multiplexed stream frame: an 8 byte header
// ([type, 0, 0, 0, size(4 bytes big-endian)]) followed by size bytes of
// payload. It is used when a container is attached with Tty=false, which is
// how node-agent always creates containers (see toCreateRequest) so stdout
// and stderr can be told apart.
func readMuxFrame(r io.Reader) (streamType byte, payload []byte, err error) {
	var header [8]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return 0, nil, err
	}

	streamType = header[0]
	size := binary.BigEndian.Uint32(header[4:8])
	if size == 0 {
		return streamType, nil, nil
	}

	payload = make([]byte, size)
	if _, err := io.ReadFull(r, payload); err != nil {
		return 0, nil, fmt.Errorf("read frame payload: %w", err)
	}

	return streamType, payload, nil
}

// lineSplitter buffers partial writes and reports complete '\n'-terminated
// lines (with the newline stripped). It's used to turn a stream of
// arbitrarily-chunked container output into whole lines for the console UI.
type lineSplitter struct {
	buf []byte
}

// Feed appends data and returns any newly completed lines.
func (s *lineSplitter) Feed(data []byte) []string {
	s.buf = append(s.buf, data...)

	var lines []string
	for {
		i := indexByte(s.buf, '\n')
		if i < 0 {
			break
		}
		line := string(trimCR(s.buf[:i]))
		lines = append(lines, line)
		s.buf = s.buf[i+1:]
	}
	return lines
}

func indexByte(b []byte, c byte) int {
	for i, x := range b {
		if x == c {
			return i
		}
	}
	return -1
}

func trimCR(b []byte) []byte {
	if n := len(b); n > 0 && b[n-1] == '\r' {
		return b[:n-1]
	}
	return b
}
