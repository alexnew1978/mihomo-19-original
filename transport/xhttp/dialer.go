package xhttp

import (
	"context"
	"crypto/tls" // Используем стандартный TLS
	"fmt"
	"net"
	"net/http"
	"sync"

	"golang.org/x/net/http2"
)

type Dialer struct {
	option    *Option
	tlsConfig *tls.Config
	transport *http2.Transport
	mu        sync.Mutex
}

func NewDialer(option *Option, tlsConfig interface{}) *Dialer {
	if option.Mode == "" {
		option.Mode = "stream"
	}

	d := &Dialer{
		option: option,
	}

	// Пытаемся безопасно привести к стандартному типу
	if cfg, ok := tlsConfig.(*tls.Config); ok {
		d.tlsConfig = cfg
	}

	d.transport = &http2.Transport{
		DialTLSContext: func(ctx context.Context, network, addr string, cfg *tls.Config) (net.Conn, error) {
			// Используем наш конфиг, если он есть, иначе тот, что передал транспорт
			targetCfg := d.tlsConfig
			if targetCfg == nil {
				targetCfg = cfg
			}
			
			// Создаем dialer внутри для каждого вызова
			dialer := &net.Dialer{}
			return tls.DialWithDialer(dialer, network, addr, targetCfg)
		},
	}

	return d
}

func (d *Dialer) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	host := d.option.Host
	if host == "" {
		host, _, _ = net.SplitHostPort(addr)
	}

	path := d.option.Path
	if path == "" {
		path = "/"
	}

	url := fmt.Sprintf("https://%s%s", host, path)
	left, right := net.Pipe()

	req, err := http.NewRequestWithContext(ctx, "POST", url, left)
	if err != nil {
		left.Close()
		right.Close()
		return nil, err
	}

	if d.option.ExtraHeaders != nil {
		for k, v := range d.option.ExtraHeaders {
			req.Header.Set(k, v)
		}
	}

	go func() {
		resp, err := d.transport.RoundTrip(req)
		if err != nil {
			right.Close()
			return
		}
		defer resp.Body.Close()

		buf := make([]byte, 32*1024)
		for {
			n, err := resp.Body.Read(buf)
			if n > 0 {
				right.Write(buf[:n])
			}
			if err != nil {
				break
			}
		}
		right.Close()
	}()

	return right, nil
}
