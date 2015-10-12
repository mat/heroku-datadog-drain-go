package statslogdrain

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
)

// LogdrainServer does stuff #TODO
func LogdrainServer(w http.ResponseWriter, req *http.Request) {
	log.Println(req)

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
		log.Println(fmt.Sprintf("inc heroku.router.%s 1\n", values["status"]))
		io.WriteString(w, fmt.Sprintf("inc heroku.router.%s 1\n", values["status"]))
	}
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
