package httpclient

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

func Execute(req *Request) (*Response, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: false},
		},
		CheckRedirect: func(r *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}

	var bodyReader io.Reader
	if req.Body != "" {
		bodyReader = strings.NewReader(req.Body)
	}

	httpReq, err := http.NewRequest(req.Method, req.URL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}

	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}

	start := time.Now()
	resp, err := client.Do(httpReq)
	elapsed := time.Since(start)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	result := &Response{
		Status:     resp.Status,
		StatusCode: resp.StatusCode,
		Headers:    make(map[string]string),
		HeaderStr:  formatHeaders(resp.Header),
		Body:       string(bodyBytes),
		Elapsed:    elapsed,
		Size:       int64(len(bodyBytes)),
	}

	for k, vals := range resp.Header {
		result.Headers[k] = strings.Join(vals, ", ")
	}

	return result, nil
}

func formatHeaders(h http.Header) string {
	var sb strings.Builder
	for k, vals := range h {
		sb.WriteString(fmt.Sprintf("%s: %s\n", k, strings.Join(vals, ", ")))
	}
	return strings.TrimRight(sb.String(), "\n")
}
