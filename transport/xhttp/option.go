package xhttp

import (
	"net/http"
)

// Option описывает настройки транспорта XHTTP (SplitHTTP)
type Option struct {
	Host          string            `json:"host,omitempty" yaml:"host,omitempty"`
	Path          string            `json:"path,omitempty" yaml:"path,omitempty"`
	Mode          string            `json:"mode,omitempty" yaml:"mode,omitempty"` // "stream" | "packet"
	ExtraHeaders  map[string]string `json:"headers,omitempty" yaml:"headers,omitempty"`
	
	// Специфичные настройки для SplitHTTP (из SagerNet/Sing-box)
	UploadPath    string            `json:"upload_path,omitempty" yaml:"upload_path,omitempty"`
	DownloadPath  string            `json:"download_path,omitempty" yaml:"download_path,omitempty"`
	Params        map[string]string `json:"query,omitempty" yaml:"query,omitempty"`
	
	// Опционально: паддинг для скрытия размера пакетов
	Padding       bool              `json:"padding,omitempty" yaml:"padding,omitempty"`
}

// ApplyHeadersAndQuery применяет заголовки и Query параметры к запросу
func (o *Option) ApplyHeadersAndQuery(req *http.Request) {
	// 1. Применяем Host
	if o.Host != "" {
		req.Host = o.Host
		// Важно продублировать в Header, так как некоторые CDN смотрят именно сюда
		req.Header.Set("Host", o.Host)
	}
	
	// 2. Применяем дополнительные заголовки (Extra Headers)
	for k, v := range o.ExtraHeaders {
		req.Header.Set(k, v)
	}
	
	// 3. User-Agent по умолчанию (маскировка под Chrome)
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	}

	// 4. Применяем Query параметры (например ?ed=2048)
	if len(o.Params) > 0 {
		q := req.URL.Query()
		for k, v := range o.Params {
			q.Add(k, v)
		}
		req.URL.RawQuery = q.Encode()
	}
}
