package server_test

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	"metrics/internal/config"
	"metrics/internal/repository"
	"metrics/internal/server"
	"metrics/internal/service"
)

// newExampleServer creates a test HTTP server backed by an in-memory store.
// The caller is responsible for calling ts.Close() when done.
func newExampleServer() *httptest.Server {
	cfg := &config.Config{}
	repo := repository.NewMemStorage()
	svc := service.NewMetricsService(repo, nil)
	srv := server.New(cfg, svc, svc, svc, nil)
	return httptest.NewServer(srv.Handler())
}

// Example_updateGauge demonstrates updating a gauge metric via the URL-parameter endpoint.
func Example_updateGauge() {
	ts := newExampleServer()
	defer ts.Close()

	resp, _ := http.Post(ts.URL+"/update/gauge/Alloc/1.5", "", nil)
	resp.Body.Close()
	fmt.Println(resp.StatusCode)

	// Output:
	// 200
}

// Example_updateCounter demonstrates that counter metrics accumulate across requests.
func Example_updateCounter() {
	ts := newExampleServer()
	defer ts.Close()

	r1, _ := http.Post(ts.URL+"/update/counter/PollCount/3", "", nil)
	r1.Body.Close()
	r2, _ := http.Post(ts.URL+"/update/counter/PollCount/7", "", nil)
	r2.Body.Close()

	resp, _ := http.Get(ts.URL + "/value/counter/PollCount")
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	fmt.Println(string(body))

	// Output:
	// 10
}

// Example_getMetricaValue demonstrates reading a gauge value via the URL-parameter endpoint.
func Example_getMetricaValue() {
	ts := newExampleServer()
	defer ts.Close()

	r, _ := http.Post(ts.URL+"/update/gauge/Alloc/1.5", "", nil)
	r.Body.Close()

	resp, _ := http.Get(ts.URL + "/value/gauge/Alloc")
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	fmt.Println(string(body))

	// Output:
	// 1.5
}

// Example_updateJSON demonstrates updating a metric using a JSON body.
// The response contains the stored metric serialised as JSON.
func Example_updateJSON() {
	ts := newExampleServer()
	defer ts.Close()

	body := `{"id":"Alloc","type":"gauge","value":2.5}`
	resp, _ := http.Post(ts.URL+"/update/", "application/json", strings.NewReader(body))
	data, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	fmt.Println(strings.TrimSpace(string(data)))

	// Output:
	// {"id":"Alloc","type":"gauge","value":2.5}
}

// Example_getMetricaJSON demonstrates reading a metric using a JSON body.
func Example_getMetricaJSON() {
	ts := newExampleServer()
	defer ts.Close()

	// Store the metric first.
	r, _ := http.Post(ts.URL+"/update/", "application/json",
		strings.NewReader(`{"id":"Alloc","type":"gauge","value":3.5}`))
	r.Body.Close()

	// Retrieve it via JSON.
	resp, _ := http.Post(ts.URL+"/value/", "application/json",
		strings.NewReader(`{"id":"Alloc","type":"gauge"}`))
	data, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	fmt.Println(strings.TrimSpace(string(data)))

	// Output:
	// {"id":"Alloc","type":"gauge","value":3.5}
}

// Example_updateBatch demonstrates atomically storing multiple metrics in one request.
func Example_updateBatch() {
	ts := newExampleServer()
	defer ts.Close()

	batch := `[
		{"id":"Alloc","type":"gauge","value":1.0},
		{"id":"PollCount","type":"counter","delta":5}
	]`
	resp, _ := http.Post(ts.URL+"/updates/", "application/json", strings.NewReader(batch))
	resp.Body.Close()
	fmt.Println(resp.StatusCode)

	// Output:
	// 200
}
