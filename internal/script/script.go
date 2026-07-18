// Package script runs curlmoon's pre-request and test scripts through goja,
// exposing a small pm.* API modeled on Postman's, scoped to what curlmoon
// actually needs: reading/writing environment variables, inspecting the
// request that's about to be sent, inspecting the response that came back,
// and recording pass/fail test assertions.
package script

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/dop251/goja"

	"curlmoon/internal/httpclient"
)

// TestResult is the outcome of a single pm.test(name, fn) call.
type TestResult struct {
	Name   string
	Passed bool
	Err    string
}

// Result is everything a script run produced: an error if the pre-request
// script itself failed to execute, and any test results recorded.
type Result struct {
	PreRequestErr string
	TestErr       string
	Tests         []TestResult
}

// RunPreRequest executes preReq (may be empty) before the request is sent.
// env is mutated in place with any pm.environment.set calls, so callers
// should pass the same map used to resolve {{variable}} tokens.
func RunPreRequest(preReq string, env map[string]string) Result {
	var res Result
	if strings.TrimSpace(preReq) == "" {
		return res
	}
	vm := goja.New()
	setupPM(vm, env, nil, nil, &res)
	if _, err := vm.RunString(preReq); err != nil {
		res.PreRequestErr = err.Error()
	}
	return res
}

// RunTest executes test (may be empty) after a response has been received.
// env is mutated in place with any pm.environment.set calls.
func RunTest(test string, env map[string]string, req *httpclient.Request, resp *httpclient.Response) Result {
	var res Result
	if strings.TrimSpace(test) == "" {
		return res
	}
	vm := goja.New()
	setupPM(vm, env, req, resp, &res)
	if _, err := vm.RunString(test); err != nil {
		res.TestErr = err.Error()
	}
	return res
}

// setupPM installs the pm object into vm. req/resp may be nil (pre-request
// scripts have no response yet).
func setupPM(vm *goja.Runtime, env map[string]string, req *httpclient.Request, resp *httpclient.Response, res *Result) {
	pm := vm.NewObject()

	pmEnv := vm.NewObject()
	pmEnv.Set("get", func(key string) string { return env[key] })
	pmEnv.Set("set", func(key, value string) { env[key] = value })
	pm.Set("environment", pmEnv)

	vars := make(map[string]string)
	pmVars := vm.NewObject()
	pmVars.Set("get", func(key string) string { return vars[key] })
	pmVars.Set("set", func(key, value string) { vars[key] = value })
	pm.Set("variables", pmVars)

	if req != nil {
		pmReq := vm.NewObject()
		pmHeaders := vm.NewObject()
		pmHeaders.Set("get", func(key string) string { return req.Headers[key] })
		pmReq.Set("headers", pmHeaders)
		pm.Set("request", pmReq)
	}

	if resp != nil {
		pmResp := vm.NewObject()
		pmResp.Set("code", resp.StatusCode)
		pmResp.Set("status", resp.Status)
		pmRespHeaders := vm.NewObject()
		pmRespHeaders.Set("get", func(key string) string { return resp.Headers[key] })
		pmResp.Set("headers", pmRespHeaders)
		pmResp.Set("text", func() string { return resp.Body })
		pmResp.Set("json", func() (interface{}, error) {
			var v interface{}
			if err := json.Unmarshal([]byte(resp.Body), &v); err != nil {
				return nil, fmt.Errorf("response body is not valid JSON: %w", err)
			}
			return v, nil
		})
		pm.Set("response", pmResp)
	}

	pm.Set("test", func(call goja.FunctionCall) goja.Value {
		name := call.Argument(0).String()
		fn, ok := goja.AssertFunction(call.Argument(1))
		tr := TestResult{Name: name, Passed: true}
		if !ok {
			tr.Passed = false
			tr.Err = "pm.test: second argument is not a function"
			res.Tests = append(res.Tests, tr)
			return goja.Undefined()
		}
		ret, err := fn(goja.Undefined())
		switch {
		case err != nil:
			tr.Passed = false
			tr.Err = err.Error()
		case ret != nil && !goja.IsUndefined(ret) && ret.ExportType() != nil &&
			ret.ExportType().Kind() == reflect.Bool && !ret.ToBoolean():
			tr.Passed = false
			tr.Err = "assertion returned false"
		}
		res.Tests = append(res.Tests, tr)
		return goja.Undefined()
	})

	vm.Set("pm", pm)
}
