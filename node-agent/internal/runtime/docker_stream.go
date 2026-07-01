package runtime

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
)

// Attach opens a raw hijacked connection to the container's combined
// stdin/stdout/stderr stream. The Docker Engine API only exposes this over a
// protocol upgrade, not a normal JSON response, so it's done by hand here
// rather than through Docker (the higher-level client) — the same technique
// the official SDK uses internally.
func (d *Docker) Attach(_ context.Context, id string) (Console, error) {
	conn, err := net.Dial("unix", d.socketPath)
	if err != nil {
		return nil, fmt.Errorf("attach: dial docker socket: %w", err)
	}

	path := fmt.Sprintf("/%s/containers/%s/attach?stream=1&stdin=1&stdout=1&stderr=1", d.apiVersion, id)
	req := "POST " + path + " HTTP/1.1\r\nHost: docker\r\nConnection: Upgrade\r\nUpgrade: tcp\r\n\r\n"
	if _, err := conn.Write([]byte(req)); err != nil {
		conn.Close()
		return nil, fmt.Errorf("attach: write request: %w", err)
	}

	reader := bufio.NewReader(conn)
	resp, err := http.ReadResponse(reader, nil)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("attach: read response: %w", err)
	}

	if resp.StatusCode != http.StatusSwitchingProtocols {
		defer resp.Body.Close()
		conn.Close()
		return nil, fmt.Errorf("attach: unexpected status %s: %s", resp.Status, readDockerError(resp.Body))
	}

	console := &dockerConsole{
		conn:   conn,
		reader: reader,
		output: make(chan string, 64),
		closed: make(chan struct{}),
	}
	go console.pump()

	return console, nil
}

// dockerConsole implements Console over a hijacked attach connection.
type dockerConsole struct {
	conn   net.Conn
	reader *bufio.Reader
	output chan string
	closed chan struct{}
	once   sync.Once
}

func (c *dockerConsole) Write(p []byte) (int, error) {
	return c.conn.Write(p)
}

func (c *dockerConsole) Close() error {
	c.once.Do(func() { close(c.closed) })
	return c.conn.Close()
}

func (c *dockerConsole) Output() <-chan string {
	return c.output
}

func (c *dockerConsole) pump() {
	defer close(c.output)

	var splitter lineSplitter
	for {
		streamType, payload, err := readMuxFrame(c.reader)
		if err != nil {
			return
		}
		if streamType != streamStdout && streamType != streamStderr {
			continue
		}

		for _, line := range splitter.Feed(payload) {
			select {
			case c.output <- line:
			case <-c.closed:
				return
			}
		}
	}
}
