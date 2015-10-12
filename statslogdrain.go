package statslogdrain

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"

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
			// log.Println("line:", line)
			processLine(w, line)
		}
	}
}

func processLine(w http.ResponseWriter, line string) {
	if strings.Contains(line, "router") {
		values := mapFromLine(line)

		tags := collectTags(values)
		client.Count(fmt.Sprintf("heroku.router.%s", values["status"]), 1, tags, 1)
		client.Count(fmt.Sprintf("heroku.router.%cxx", values["status"][0]), 1, tags, 1)
	}
}

func collectTags(values map[string]string) []string {
	tags := []string{}
	tagsToUse := []string{"dyno", "method", "status", "host", "code"}
	for _, tag := range tagsToUse {
		value := values[tag]
		if value != "" {
			tags = append(tags, fmt.Sprintf("%s:%v", tag, value))
		}
	}
	return tags
}

func mapFromLine(line string) map[string]string {
	result := make(map[string]string)

	pairs := regexp.MustCompile(`[a-z]+=(([^"]\S+)|(["][^"]*?["]))`).FindAllString(line, -1)
	for _, p := range pairs {
		keyValue := strings.SplitN(p, "=", 2)
		key := keyValue[0]
		value := strings.Trim(keyValue[1], `"`)
		result[key] = value
	}

	return result
}

// StatsDClient is cool
type statsDClient interface {
	Count(string, int64, []string, float64) error
}

var client statsDClient

// func SetClient(c statsDClient) {
// 	client = c
// }

func init() {
	var err error
	client, err = statsd.New("127.0.0.1:8125")
	if err != nil {
		log.Fatal(err)
	}
	// SetClient(client)

	// statsdClient.Namespace = "flubber."
	// statsdClient.Tags = append(c.Tags, "us-east-1a")
	// err = c.Gauge("request.duration", 1.2, nil, 1)
}
