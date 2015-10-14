package statslogdrain

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/DataDog/datadog-go/statsd"
)

// LogdrainServer parses Heroku logdrain requests
// and sends stats to datadog via statsd protocol
func LogdrainServer(w http.ResponseWriter, req *http.Request) {
	userName, valid := passwordValid(req)
	if !valid {
		log.Println("Unauthorized request:", req.URL)
		http.Error(w, "Unauthorized", 401)
		return
	}

	scanner := bufio.NewScanner(req.Body)
	defer req.Body.Close()

	lines := 0
	for scanner.Scan() {
		if err := scanner.Err(); err != nil {
			log.Println("error reading body:", err)
		} else {
			processLine(scanner.Text(), userName)
		}
		lines++
	}

	if enableDrainMetrics {
		client.Count("heroku.logdrain.lines", int64(lines), []string{"type:seen"}, 1)
	}

	// Don't return unnecessary headers
	header := w.Header()
	header["Content-Type"] = nil
	header["Connection"] = nil
	header["Date"] = nil
}

const metricsPrefix = "sample#"

func processLine(line, userName string) {
	startedAt := time.Now()
	defer trackTiming("processLine", startedAt)

	if strings.Contains(line, "router") {
		handleLine(handleRouterLine, line, userName)
	} else if strings.Contains(line, "logdrain-metrics") {
		handleLine(handleMetricLine, line, userName)
	} else if strings.Contains(line, "sample#load") || strings.Contains(line, "sample#memory") {
		handleLine(handleDynoMetrics, line, userName)
	} else {
		if enableDrainMetrics {
			client.Count("heroku.logdrain.lines", int64(1), []string{"type:ignored"}, 1)
		}
	}
}

func handleLine(handler lineHandler, line string, userName string) {
	values := mapFromLineReader(line)
	tags := collectTags(values, userName)

	handler(values, tags)
}

type lineHandler func(values map[string]string, tags []string)

func handleRouterLine(values map[string]string, tags []string) {
	client.Histogram("heroku.router.request.bytes", parseFloat(values["bytes"]), tags, 1)
	client.Histogram("heroku.router.request.connect", parseFloat(values["connect"]), tags, 1)
	client.Histogram("heroku.router.request.service", parseFloat(values["service"]), tags, 1)
}

func handleMetricLine(values map[string]string, tags []string) {
	for k, v := range values {
		if strings.HasPrefix(k, metricsPrefix) {
			sampleName := strings.TrimPrefix(k, metricsPrefix)
			client.Histogram(fmt.Sprintf("heroku.custom.%s", sampleName), parseFloat(v), tags, 1)
		}
	}
}

func handleDynoMetrics(values map[string]string, tags []string) {
	for k, v := range values {
		if strings.HasPrefix(k, metricsPrefix) {
			sampleName := strings.TrimPrefix(k, metricsPrefix)
			client.Histogram(fmt.Sprintf("heroku.dyno.%s", sampleName), parseFloat(v), tags, 1)
		}
	}
}

var tagsToUse = []string{"dyno", "method", "status", "host", "code", "source"}

func collectTags(values map[string]string, userName string) []string {
	startedAt := time.Now()
	defer trackTiming("collectTags", startedAt)

	tags := []string{}
	for _, tag := range tagsToUse {
		value := values[tag]
		if value != "" {
			tags = append(tags, fmt.Sprintf("%s:%v", tag, value))
		}
	}

	status := values["status"]
	if status != "" {
		tags = append(tags, fmt.Sprintf("statusgroup:%cxx", status[0]))
	}

	tags = append(tags, fmt.Sprintf("app:%v", userName))
	return tags
}

var pairRegexp = regexp.MustCompile(`\S+=(([^"]\S*)|(["][^"]*?["]))`)

func mapFromLineReader(line string) map[string]string {
	result := make(map[string]string)

	reader := bufio.NewReaderSize(strings.NewReader(line), len(line))

	parsingKey := true
	parsingValue := false
	quotedValue := false
	key := bytes.Buffer{}
	value := bytes.Buffer{}

	for {
		c, _, err := reader.ReadRune()
		if err == io.EOF {
			if value.Len() > 0 {
				result[key.String()] = value.String()
			}
			break
		} else if err != nil {
			log.Panic(err)
		}

		if parsingKey && c == '=' {
			parsingKey = false
			parsingValue = true
			peek, _ := reader.Peek(1)
			quotedValue = peek[0] == '"'
		} else if parsingKey {
			if unicode.IsSpace(c) {
				key.Reset()
			} else {
				key.WriteRune(c)
			}
		} else if parsingValue {
			if quotedValue && c == '"' {
				endOfValue := value.Len() > 0
				if endOfValue {
					parsingKey = true
					parsingValue = false
					quotedValue = false
					result[key.String()] = value.String()
					key.Reset()
					value.Reset()
				}
			} else if !quotedValue && unicode.IsSpace(c) {
				parsingKey = true
				parsingValue = false
				result[key.String()] = value.String()
				key.Reset()
				value.Reset()
			} else {
				value.WriteRune(c)
			}
		}
	}

	return result
}

func mapFromLine(line string) map[string]string {
	startedAt := time.Now()
	defer trackTiming("mapFromLine", startedAt)

	result := make(map[string]string)

	pairs := pairRegexp.FindAllString(line, -1)
	for _, p := range pairs {
		keyValue := strings.SplitN(p, "=", 2)
		key := keyValue[0]
		value := strings.Trim(keyValue[1], `"`)
		result[key] = value
	}

	return result
}

func trackTiming(funcName string, timestamp time.Time) {
	if enableDrainMetrics {
		ms := time.Since(timestamp).Seconds() / 1000.0
		client.Histogram(fmt.Sprintf("heroku.logdrain.timings.%s", funcName), ms, nil, 1)
	}
}

var userPasswords map[string]string

func passwordValid(req *http.Request) (string, bool) {
	username, password, ok := req.BasicAuth()
	return username, (ok && (password == userPasswords[username]))
}

func notFloatRune(r rune) bool {
	return !(r == '1' || r == '2' || r == '3' || r == '4' || r == '5' || r == '6' || r == '7' || r == '8' || r == '9' || r == '0' || r == '.')
}

func parseFloat(str string) float64 {
	str = strings.TrimRightFunc(str, notFloatRune)

	f, err := strconv.ParseFloat(str, 10)
	if err != nil {
		return -1
	}
	return f
}

// StatsDClient is used to make testing easier
type statsDClient interface {
	Histogram(name string, value float64, tags []string, rate float64) error
	Count(name string, value int64, tags []string, rate float64) error
}

var client statsDClient

var enableDrainMetrics = true

// SetUserpasswords sets the required user/password map for authentication
func SetUserpasswords(passwordMap map[string]string) {
	userPasswords = passwordMap
}

func init() {
	var err error
	client, err = statsd.New("127.0.0.1:8125")
	if err != nil {
		log.Fatal(err)
	}

	if os.Getenv("ENABLE_DRAIN_METRICS") == "" {
		enableDrainMetrics = true
	} else {
		enableDrainMetrics, _ = strconv.ParseBool(os.Getenv("ENABLE_DRAIN_METRICS"))
	}
}
