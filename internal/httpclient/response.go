package httpclient

import "time"

type Response struct {
	Status     string
	StatusCode int
	Headers    map[string]string
	HeaderStr  string
	Body       string
	Elapsed    time.Duration
	Size       int64
}
