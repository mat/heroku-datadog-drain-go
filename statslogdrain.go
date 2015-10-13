package statslogdrain

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/DataDog/datadog-go/statsd"
)

// LogdrainServer parses Heroku logdrain requests
// and sends stats to datadog via statsd protocol
func LogdrainServer(w http.ResponseWriter, req *http.Request) {
	if !passwordValid(req) {
		log.Println("Unauthorized request:", req.URL)
		http.Error(w, "Unauthorized", 401)
		return
	}
	userName, _, _ := req.BasicAuth()

	scanner := bufio.NewScanner(req.Body)
	defer req.Body.Close()

	for scanner.Scan() {
		if err := scanner.Err(); err != nil {
			log.Println("error reading body:", err)
		} else {
			processLine(userName, scanner.Text())
		}
	}
}

const customMetricsPrefix = "sample#"

func processLine(userName string, line string) {
	if strings.Contains(line, "router") {
		handleRouterLine(line, userName)
	} else if strings.Contains(line, "logdrain-metrics") {
		handleMetricLine(line, userName)
	}
}

func handleRouterLine(line, userName string) {
	values := mapFromLine(line)
	tags := collectTags(values, userName)

	client.TimeInMilliseconds("heroku.router.request.connect", parseFloat(values["connect"]), tags, 1)
	client.TimeInMilliseconds("heroku.router.request.service", parseFloat(values["service"]), tags, 1)
}

func handleMetricLine(line, userName string) {
	values := mapFromLine(line)
	tags := collectTags(values, userName)

	for k, v := range values {
		if strings.HasPrefix(k, customMetricsPrefix) {
			sampleName := strings.TrimPrefix(k, customMetricsPrefix)
			client.TimeInMilliseconds(fmt.Sprintf("heroku.custom.%s", sampleName), parseFloat(v), tags, 1)
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

var pairRegexp = regexp.MustCompile(`\S+=(([^"]\S+)|(["][^"]*?["]))`)

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

func passwordValid(req *http.Request) bool {
	username, password, ok := req.BasicAuth()
	return ok && (password == userPasswords[username])
}

func parseFloat(str string) float64 {
	if str == "" {
		return -1
	}

	duration, err := time.ParseDuration(strings.TrimSpace(str))
	if err != nil {
		return -1
	}
	return duration.Seconds() * 1000.0
}

// StatsDClient is used to make testing easier
type statsDClient interface {
	TimeInMilliseconds(name string, value float64, tags []string, rate float64) error
}

var client statsDClient

const commandBufferSize = 1000

// SetUserpasswords sets the required user/password map for authentication
func SetUserpasswords(passwordMap map[string]string) {
	userPasswords = passwordMap
}

func init() {
	var err error
	client, err = statsd.NewBuffered("127.0.0.1:8125", commandBufferSize)
	if err != nil {
		log.Fatal(err)
	}
}
