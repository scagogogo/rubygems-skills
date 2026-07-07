package repository

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"sync"
	"testing"
)

// fakeRoundTripper is a test http.RoundTripper that records requests and
// returns canned responses keyed by URL path.
type fakeRoundTripper struct {
	mu        sync.Mutex
	requests  []recordedRequest
	responses map[string]cannedResponse // keyed by URL path
	sequences map[string][]cannedResponse
	seqIdx    map[string]int
	callCount int
}

type recordedRequest struct {
	Method string
	URL    string
	Path   string
	Header http.Header
	Body   []byte
}

type cannedResponse struct {
	statusCode  int
	body        []byte
	header      http.Header
	err         error
	bodyOverride io.ReadCloser // when set, replaces nopBody in the response
}

func newFakeTransport() *fakeRoundTripper {
	return &fakeRoundTripper{
		responses: map[string]cannedResponse{},
		sequences: map[string][]cannedResponse{},
		seqIdx:    map[string]int{},
	}
}

// stub registers a canned response for a given URL path.
func (t *fakeRoundTripper) stub(path string, status int, body string) *fakeRoundTripper {
	t.responses[path] = cannedResponse{statusCode: status, body: []byte(body), header: http.Header{}}
	return t
}

// stubErr registers a transport-level error for a given URL path.
func (t *fakeRoundTripper) stubErr(path string, err error) *fakeRoundTripper {
	t.responses[path] = cannedResponse{err: err}
	return t
}

// stubSequence registers an ordered sequence of responses for a given URL path.
// Each request consumes the next response; when the sequence is exhausted the
// last response repeats.
func (t *fakeRoundTripper) stubSequence(path string, resps ...cannedResponse) *fakeRoundTripper {
	t.sequences[path] = resps
	return t
}

// overrideBody replaces the response body for a path with a custom ReadCloser.
// Used to inject bodies that fail on read, exercising error paths.
func (t *fakeRoundTripper) overrideBody(path string, body io.ReadCloser) *fakeRoundTripper {
	if c, ok := t.responses[path]; ok {
		c.bodyOverride = body
		t.responses[path] = c
	} else {
		t.responses[path] = cannedResponse{statusCode: 200, bodyOverride: body}
	}
	return t
}

// brokenBody is a ReadCloser whose Read always returns an error.
type brokenBody struct{}

func (brokenBody) Read(p []byte) (int, error) { return 0, errors.New("read failed") }
func (brokenBody) Close() error               { return nil }

func (t *fakeRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	t.mu.Lock()
	t.callCount++
	var body []byte
	if req.Body != nil {
		body, _ = readAllAndReset(req)
	}
	t.requests = append(t.requests, recordedRequest{
		Method: req.Method,
		URL:    req.URL.String(),
		Path:   req.URL.Path,
		Header: req.Header.Clone(),
		Body:   body,
	})
	canned, ok := t.responses[req.URL.Path]
	if seq, hasSeq := t.sequences[req.URL.Path]; hasSeq {
		idx := t.seqIdx[req.URL.Path]
		if idx >= len(seq) {
			idx = len(seq) - 1
		}
		canned = seq[idx]
		ok = true
		t.seqIdx[req.URL.Path] = idx + 1
	}
	t.mu.Unlock()

	if !ok {
		return &http.Response{StatusCode: http.StatusNotFound, Body: http.NoBody, Header: http.Header{}, Request: req}, nil
	}
	if canned.err != nil {
		return nil, canned.err
	}
	hdr := canned.header
	if hdr == nil {
		hdr = http.Header{}
	}
	var respBody io.ReadCloser
	if canned.bodyOverride != nil {
		respBody = canned.bodyOverride
	} else {
		respBody = nopBody(canned.body)
	}
	return &http.Response{
		StatusCode: canned.statusCode,
		Body:       respBody,
		Header:     hdr,
		Request:    req,
	}, nil
}

// newStubbedRepo builds a RepositoryImpl whose HTTP layer is stubbed. Retry is
// disabled so tests assert single-shot behavior (unless explicitly testing retry).
func newStubbedRepo(t *testing.T, transport *fakeRoundTripper) *RepositoryImpl {
	t.Helper()
	opts := NewOptions()
	opts.SetHTTPClient(&http.Client{Transport: transport})
	opts.DisableRetry()
	return NewRepository(opts)
}

// newStubbedWriteRepo builds a WriteRepositoryImpl whose HTTP layer is stubbed.
func newStubbedWriteRepo(transport *fakeRoundTripper) *WriteRepositoryImpl {
	opts := NewOptions()
	opts.SetHTTPClient(&http.Client{Transport: transport})
	opts.DisableRetry()
	return NewWriteRepository(opts)
}

// readAllAndReset reads req.Body and restores it via GetBody when available.
func readAllAndReset(req *http.Request) ([]byte, error) {
	if req.GetBody != nil {
		b, err := req.GetBody()
		if err == nil {
			defer b.Close()
			return io.ReadAll(b)
		}
	}
	if req.Body == nil {
		return nil, nil
	}
	b, err := io.ReadAll(req.Body)
	req.Body = nil
	return b, err
}

func nopBody(b []byte) io.ReadCloser      { return io.NopCloser(bytes.NewReader(b)) }
