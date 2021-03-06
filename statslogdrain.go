package statslogdrain

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"

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
}

const metricsPrefix = "sample#"

func processLine(line, userName string) {
	if strings.Contains(line, "router") {
		handleLine(handleRouterLine, line, userName)
	} else if strings.Contains(line, "logdrain-metrics") {
		handleLine(handleMetricLine, line, userName)
	} else if strings.Contains(line, "sample#load") || strings.Contains(line, "sample#memory") {
		handleLine(handleDynoMetrics, line, userName)
	} else {
		if enableDrainLogging {
			log.Println("unhandled line:", line)
		}
	}
}

func handleLine(handler lineHandler, line string, userName string) {
	values := mapFromLine(line)
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

func mapFromLine(line string) map[string]string {
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

var userPasswords map[string]string

func passwordValid(req *http.Request) (string, bool) {
	username, password, ok := req.BasicAuth()
	return username, (ok && (password == userPasswords[username]))
}

var floatRegexp = regexp.MustCompile(`[^.0-9]`)

func parseFloat(str string) float64 {
	str = floatRegexp.ReplaceAllLiteralString(str, "")

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

var enableDrainLogging = false

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

	enabled, err := strconv.ParseBool(os.Getenv("ENABLE_DRAIN_METRICS"))
	if err != nil {
		enableDrainLogging = enabled
	}
}
