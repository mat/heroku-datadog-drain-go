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

// LogdrainServer does stuff #TODO
func LogdrainServer(w http.ResponseWriter, req *http.Request) {
	scanner := bufio.NewScanner(req.Body)
	defer req.Body.Close()

	for scanner.Scan() {
		if err := scanner.Err(); err != nil {
			log.Println("error reading body:", err)

		} else {
			line := scanner.Text()
			// TODO remove w param
			processLine(w, req.URL.User.Username(), line)
		}
	}
}

const customMetricsPrefix = "sample#"

func processLine(w http.ResponseWriter, userName string, line string) {
	if strings.Contains(line, "router") {
		values := mapFromLine(line)
		tags := collectTags(values, userName)

		client.Count(fmt.Sprintf("heroku.router.%s", values["status"]), 1, tags, 1)
		client.Count(fmt.Sprintf("heroku.router.%cxx", values["status"][0]), 1, tags, 1)
		client.TimeInMilliseconds("heroku.router.request.connect", parseFloat(values["connect"]), tags, 1)
		client.TimeInMilliseconds("heroku.router.request.service", parseFloat(values["service"]), tags, 1)
	} else if strings.Contains(line, "logdrain-metrics") {
		values := mapFromLine(line)
		tags := collectTags(values, userName)

		for k, v := range values {
			if strings.HasPrefix(k, customMetricsPrefix) {
				sampleName := strings.TrimPrefix(k, customMetricsPrefix)
				client.TimeInMilliseconds(fmt.Sprintf("heroku.custom.%s", sampleName), parseFloat(v), tags, 1)
			}
		}
	}
}

func collectTags(values map[string]string, userName string) []string {
	tags := []string{}
	tagsToUse := []string{"dyno", "method", "status", "host", "code", "source"}
	for _, tag := range tagsToUse {
		value := values[tag]
		if value != "" {
			tags = append(tags, fmt.Sprintf("%s:%v", tag, value))
		}
	}

	tags = append(tags, fmt.Sprintf("app:%v", userName))
	return tags
}

var pairRegexp *regexp.Regexp

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

// StatsDClient is cool
type statsDClient interface {
	Count(name string, value int64, tags []string, rate float64) error
	TimeInMilliseconds(name string, value float64, tags []string, rate float64) error
}

var client statsDClient

const commandBufferSize = 1000

func init() {
	pairRegexp = regexp.MustCompile(`\S+=(([^"]\S+)|(["][^"]*?["]))`)

	var err error
	client, err = statsd.NewBuffered("127.0.0.1:8125", commandBufferSize)
	if err != nil {
		log.Fatal(err)
	}
}
