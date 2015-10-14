package statslogdrain

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func BenchmarkHttpEndpoint(b *testing.B) {
	client = &noopClient{}
	SetUserpasswords(map[string]string{"test-app": "deadbeef"})
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("POST", "http://test-app:deadbeef@example.com/foo", strings.NewReader(strings.TrimSpace(routerMetricsBody)))
		req.SetBasicAuth("test-app", "deadbeef")
		w := httptest.NewRecorder()
		LogdrainServer(w, req)
	}
}

func BenchmarkMapFromLine(b *testing.B) {
	for i := 0; i < b.N; i++ {
		m := mapFromLine(`255 <158>1 2015-04-02T12:52:31.520012+00:00 host heroku router - at=error code=H12 desc="Request timeout" method=GET path="/" host=myapp.com fwd=17.17.17.17 dyno=web.1 connect=6ms service=30001ms status=503 bytes=0`)
		if len(m) != 12 {
			b.Fatalf("expected map to contain 12 but got %d", len(m))
		}
		m = mapFromLine(`329 <45>1 2015-04-02T11:48:16.839348+00:00 host heroku web.1 - source=web.1 dyno=heroku.35930502.b9de5fce-44b7-4287-99a7-504519070cba sample#memory_total=103.50MB sample#memory_rss=94.70MB sample#memory_cache=0.32MB sample#memory_swap=8.48MB sample#memory_pgpgin=36091pages sample#memory_pgpgout=11765pages`)
		if len(m) != 8 {
			b.Fatalf("expected map to contain 8 but got %d", len(m))
		}
	}
}

func BenchmarkMapFromLineReader(b *testing.B) {
	for i := 0; i < b.N; i++ {
		m := mapFromLineReader(`255 <158>1 2015-04-02T12:52:31.520012+00:00 host heroku router - at=error code=H12 desc="Request timeout" method=GET path="/" host=myapp.com fwd=17.17.17.17 dyno=web.1 connect=6ms service=30001ms status=503 bytes=0`)
		if len(m) != 12 {
			b.Fatalf("expected map to contain 12 but got %d", len(m))
		}
		m = mapFromLineReader(`329 <45>1 2015-04-02T11:48:16.839348+00:00 host heroku web.1 - source=web.1 dyno=heroku.35930502.b9de5fce-44b7-4287-99a7-504519070cba sample#memory_total=103.50MB sample#memory_rss=94.70MB sample#memory_cache=0.32MB sample#memory_swap=8.48MB sample#memory_pgpgin=36091pages sample#memory_pgpgout=11765pages`)
		if len(m) != 8 {
			b.Fatalf("expected map to contain 8 but got %d", len(m))
		}
	}
}

type noopClient struct {
}

func (c *noopClient) Histogram(name string, value float64, tags []string, rate float64) error {
	return nil
}

func (c *noopClient) Count(name string, value int64, tags []string, rate float64) error {
	return nil
}
