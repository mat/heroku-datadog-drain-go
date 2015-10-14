package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/mat/heroku-datadog-drain-go"
)

func main() {
	http.HandleFunc("/", statslogdrain.LogdrainServer)
	statslogdrain.SetUserpasswords(userPasswordsFromEnv())
	port := os.Getenv("PORT")
	if port == "" {
		log.Println("cannot start, need a PORT")
		os.Exit(1)
	}
	log.Println("Server running on port", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
}

func userPasswordsFromEnv() map[string]string {
	allowedApps := os.Getenv("ALLOWED_APPS")
	if allowedApps == "" {
		log.Panic("Cannot start, ALLOWED_APPS not set")
	}

	passwords := make(map[string]string)
	apps := strings.Split(allowedApps, ",")
	for _, app := range apps {
		passwordKey := fmt.Sprintf("%s_PASSWORD", strings.ToUpper(app))
		password := os.Getenv(passwordKey)
		if password == "" {
			log.Panicf("Cannot find password, %s not set", passwordKey)
		}
		passwords[app] = password
	}

	return passwords
}
