package statslogdrain

import (
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

const routerMetricsBody = `
255 <158>1 2015-04-02T11:52:34.520012+00:00 host heroku router - at=info method=POST path="/users" host=myapp.com request_id=c1806361-2081-42e7-a8aa-92b6808eac8e fwd="24.76.242.18" dyno=web.1 connect=1ms service=37ms status=201 bytes=828\n
293 <158>1 2015-04-02T11:52:37.888674+00:00 host heroku router - at=info method=GET path="/users/me/tasks" host=myapp.com request_id=d66b4e46-049f-4592-b0b0-a253dc1b0c62 fwd="24.77.243.44" dyno=web.2 connect=1ms service=64ms status=200 bytes=54414\n
255 <158>1 2015-04-02T12:52:31.520012+00:00 host heroku router - at=error code=H12 desc="Request timeout" method=GET path="/" host=myapp.com fwd=17.17.17.17 dyno=web.1 connect=6ms service=30001ms status=503 bytes=0
`

func TestUnauthorized(t *testing.T) {
	initServer()
	req, _ := http.NewRequest("POST", "http://test-app:wrong_password@example.com/foo", strings.NewReader(""))

	w := httptest.NewRecorder()
	LogdrainServer(w, req)
	assert.Equal(t, 401, w.Code)
}

func TestRouterMetrics(t *testing.T) {
	initServer()

	req, _ := http.NewRequest("POST", "http://test-app:deadbeef@example.com/foo", strings.NewReader(strings.TrimSpace(routerMetricsBody)))
	req.SetBasicAuth("test-app", "deadbeef")

	w := httptest.NewRecorder()
	LogdrainServer(w, req)
	assert.Equal(t, 200, w.Code)

	assert.Equal(t, []command{
		{"heroku.router.request.count", 1, []string{"dyno:web.1", "method:POST", "status:201", "host:myapp.com", "statusgroup:2xx", "app:test-app"}},
		{"heroku.router.request.count", 1, []string{"dyno:web.2", "method:GET", "status:200", "host:myapp.com", "statusgroup:2xx", "app:test-app"}},
		{"heroku.router.request.count", 1, []string{"dyno:web.1", "method:GET", "status:503", "host:myapp.com", "code:H12", "statusgroup:5xx", "app:test-app"}},
	}, client.(*stubClient).counts)
	assert.Equal(t, []command{
		{"heroku.router.request.bytes", 828, []string{"dyno:web.1", "method:POST", "status:201", "host:myapp.com", "statusgroup:2xx", "app:test-app"}},
		{"heroku.router.request.connect", 1, []string{"dyno:web.1", "method:POST", "status:201", "host:myapp.com", "statusgroup:2xx", "app:test-app"}},
		{"heroku.router.request.service", 37, []string{"dyno:web.1", "method:POST", "status:201", "host:myapp.com", "statusgroup:2xx", "app:test-app"}},
		{"heroku.router.request.bytes", 54414, []string{"dyno:web.2", "method:GET", "status:200", "host:myapp.com", "statusgroup:2xx", "app:test-app"}},
		{"heroku.router.request.connect", 1, []string{"dyno:web.2", "method:GET", "status:200", "host:myapp.com", "statusgroup:2xx", "app:test-app"}},
		{"heroku.router.request.service", 64, []string{"dyno:web.2", "method:GET", "status:200", "host:myapp.com", "statusgroup:2xx", "app:test-app"}},
		{"heroku.router.request.bytes", 0, []string{"dyno:web.1", "method:GET", "status:503", "host:myapp.com", "code:H12", "statusgroup:5xx", "app:test-app"}},
		{"heroku.router.request.connect", 6, []string{"dyno:web.1", "method:GET", "status:503", "host:myapp.com", "code:H12", "statusgroup:5xx", "app:test-app"}},
		{"heroku.router.request.service", 30001, []string{"dyno:web.1", "method:GET", "status:503", "host:myapp.com", "code:H12", "statusgroup:5xx", "app:test-app"}},
	}, client.(*stubClient).histograms)
}

const customMetricsBody = `
542 <134>1 2015-10-06T12:23:58.066218+00:00 app web.10: logdrain-metrics source=logdrain-metrics sample#s3_request.total=537.543ms
`

func TestCustomMetrics(t *testing.T) {
	initServer()

	req, _ := http.NewRequest("POST", "http://example.com/foo", strings.NewReader(strings.TrimSpace(customMetricsBody)))
	req.SetBasicAuth("test-app", "deadbeef")

	w := httptest.NewRecorder()
	LogdrainServer(w, req)
	assert.Equal(t, []command{
		{"heroku.custom.s3_request.total", 537, []string{"source:logdrain-metrics", "app:test-app"}},
	}, client.(*stubClient).histograms)
}

const dynoMetricsBody = `
229 <45>1 2015-04-02T11:48:16.839257+00:00 host heroku web.1 - source=web.1 dyno=heroku.35930502.b9de5fce-44b7-4287-99a7-504519070cba sample#load_avg_1m=0.01 sample#load_avg_5m=0.02 sample#load_avg_15m=0.03\n
329 <45>1 2015-04-02T11:48:16.839348+00:00 host heroku web.1 - source=web.1 dyno=heroku.35930502.b9de5fce-44b7-4287-99a7-504519070cba sample#memory_total=103.50MB sample#memory_rss=94.70MB sample#memory_cache=0.32MB sample#memory_swap=8.48MB sample#memory_pgpgin=36091pages sample#memory_pgpgout=11765pages
`

func TestDynoMetrics(t *testing.T) {
	initServer()

	req, _ := http.NewRequest("POST", "http://example.com/foo", strings.NewReader(strings.TrimSpace(dynoMetricsBody)))
	req.SetBasicAuth("test-app", "deadbeef")

	w := httptest.NewRecorder()
	LogdrainServer(w, req)
	assert.Len(t, client.(*stubClient).histograms, 9)
	assert.Contains(t, client.(*stubClient).histograms, command{"heroku.dyno.load_avg_1m", 0, []string{"dyno:heroku.35930502.b9de5fce-44b7-4287-99a7-504519070cba", "source:web.1", "app:test-app"}})
	assert.Contains(t, client.(*stubClient).histograms, command{"heroku.dyno.load_avg_5m", 0, []string{"dyno:heroku.35930502.b9de5fce-44b7-4287-99a7-504519070cba", "source:web.1", "app:test-app"}})
	assert.Contains(t, client.(*stubClient).histograms, command{"heroku.dyno.load_avg_15m", 0, []string{"dyno:heroku.35930502.b9de5fce-44b7-4287-99a7-504519070cba", "source:web.1", "app:test-app"}})
	assert.Contains(t, client.(*stubClient).histograms, command{"heroku.dyno.memory_total", 103, []string{"dyno:heroku.35930502.b9de5fce-44b7-4287-99a7-504519070cba", "source:web.1", "app:test-app"}})
	assert.Contains(t, client.(*stubClient).histograms, command{"heroku.dyno.memory_rss", 94, []string{"dyno:heroku.35930502.b9de5fce-44b7-4287-99a7-504519070cba", "source:web.1", "app:test-app"}})
	assert.Contains(t, client.(*stubClient).histograms, command{"heroku.dyno.memory_cache", 0, []string{"dyno:heroku.35930502.b9de5fce-44b7-4287-99a7-504519070cba", "source:web.1", "app:test-app"}})
	assert.Contains(t, client.(*stubClient).histograms, command{"heroku.dyno.memory_swap", 8, []string{"dyno:heroku.35930502.b9de5fce-44b7-4287-99a7-504519070cba", "source:web.1", "app:test-app"}})
	assert.Contains(t, client.(*stubClient).histograms, command{"heroku.dyno.memory_pgpgin", 36091, []string{"dyno:heroku.35930502.b9de5fce-44b7-4287-99a7-504519070cba", "source:web.1", "app:test-app"}})
	assert.Contains(t, client.(*stubClient).histograms, command{"heroku.dyno.memory_pgpgout", 11765, []string{"dyno:heroku.35930502.b9de5fce-44b7-4287-99a7-504519070cba", "source:web.1", "app:test-app"}})
}

func TestMapFromLine(t *testing.T) {
	line := `255 <158>1 2015-04-02T12:52:31.520012+00:00 host heroku router - at=error code=H12 desc="Request timeout" method=GET path="/" host=myapp.com fwd=17.17.17.17 dyno=web.1 connect=6ms service=30001ms status=503 bytes=0`
	actual := mapFromLine(line)
	expected := map[string]string{
		"method":  "GET",
		"host":    "myapp.com",
		"fwd":     "17.17.17.17",
		"dyno":    "web.1",
		"service": "30001ms",
		"status":  "503",
		"code":    "H12",
		"desc":    "Request timeout",
		"path":    "/",
		"connect": "6ms",
		"at":      "error",
		"bytes":   "0",
	}
	assert.Equal(t, expected, actual)
}

func TestParseFloat(t *testing.T) {
	assert.Equal(t, 32.0, parseFloat("32ms"))
	assert.Equal(t, 32.0, parseFloat(" 32ms"))
	assert.Equal(t, 32.0, parseFloat(" 32ms "))
	assert.Equal(t, 32.0, parseFloat("32ms "))
	assert.Equal(t, 32.0, parseFloat("32KB"))
	assert.Equal(t, 32.0, parseFloat("32MB"))
	assert.Equal(t, 32.0, parseFloat("32 MB"))
	assert.Equal(t, 537.543, parseFloat("537.543ms"))
	assert.Equal(t, 537.543, parseFloat("537.543"))
	assert.Equal(t, 0.0, parseFloat("0"))

	assert.Equal(t, -1, int(parseFloat("")))
}

type stubClient struct {
	histograms []command
	counts     []command
}

func (c *stubClient) Histogram(name string, value float64, tags []string, rate float64) error {
	c.histograms = append(c.histograms, command{name, int64(value), tags})
	return nil
}

func (c *stubClient) Count(name string, value int64, tags []string, rate float64) error {
	c.counts = append(c.counts, command{name, int64(value), tags})
	return nil
}

type command struct {
	key   string
	value int64
	tags  []string
}

func initServer() {
	client = &stubClient{}
	SetUserpasswords(map[string]string{"test-app": "deadbeef"})
	enableDrainMetrics = false
	log.SetOutput(ioutil.Discard)
}
