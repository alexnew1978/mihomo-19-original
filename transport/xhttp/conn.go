package xhttp

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"
)

type Conn struct {
	ctx           context.Context
	dialer        *Dialer
	remoteAddrStr string
	remoteIP      net.IP

	uploadURL   string
	downloadURL string
	httpClient  *http.Client

	reader io.ReadCloser
	writer *io.PipeWriter

	closeOnce sync.Once
	closed    chan struct{}
}

func (c *Conn) connect() error {
	c.closed = make(chan struct{})

	// --- 1. DOWNLOAD STREAM (GET) ---
	req, err := http.NewRequestWithContext(c.ctx, "GET", c.downloadURL, nil)
	if err != nil {
		return err
	}
	c.dialer.option.ApplyHeadersAndQuery(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}

	// Проверяем статус ответа
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return errors.New("unexpected status code: " + resp.Status)
	}
	c.reader = resp.Body

	// --- 2. UPLOAD STREAM (POST) ---
	pr, pw := io.Pipe()
	c.writer = pw

	go c.uploadLoop(pr)

	return nil
}

func (c *Conn) uploadLoop(body io.Reader) {
	req, err := http.NewRequestWithContext(c.ctx, "POST", c.uploadURL, body)
	if err != nil {
		c.Close()
		return
	}
	c.dialer.option.ApplyHeadersAndQuery(req)
	
	// ContentLength = -1 включает Transfer-Encoding: chunked
	req.ContentLength = -1 

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.Close()
		return
	}
	if resp.StatusCode != http.StatusOK {
		c.Close()
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
}

func (c *Conn) Read(b []byte) (n int, err error) {
	select {
	case <-c.closed:
		return 0, io.EOF
	default:
		return c.reader.Read(b)
	}
}

func (c *Conn) Write(b []byte) (n int, err error) {
	select {
	case <-c.closed:
		return 0, io.ErrClosedPipe
	default:
		return c.writer.Write(b)
	}
}

func (c *Conn) Close() error {
	c.closeOnce.Do(func() {
		close(c.closed)
		if c.reader != nil {
			c.reader.Close()
		}
		if c.writer != nil {
			c.writer.CloseWithError(io.EOF)
		}
	})
	return nil
}

// --- Методы интерфейса net.Conn ---

func (c *Conn) LocalAddr() net.Addr {
	return &net.TCPAddr{IP: net.IPv4zero, Port: 0}
}

func (c *Conn) RemoteAddr() net.Addr {
	host, portStr, err := net.SplitHostPort(c.remoteAddrStr)
	if err != nil {
		return &net.TCPAddr{IP: net.IPv4zero, Port: 0}
	}
	port, _ := strconv.Atoi(portStr)
	if c.remoteIP != nil {
		return &net.TCPAddr{IP: c.remoteIP, Port: port}
	}
	return &net.TCPAddr{IP: net.ParseIP(host), Port: port}
}

func (c *Conn) SetDeadline(t time.Time) error      { return nil }
func (c *Conn) SetReadDeadline(t time.Time) error  { return nil }
func (c *Conn) SetWriteDeadline(t time.Time) error { return nil }
