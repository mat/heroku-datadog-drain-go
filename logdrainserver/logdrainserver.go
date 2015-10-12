package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/mat/statslogdrain"
)

func main() {
	http.HandleFunc("/", statslogdrain.LogdrainServer)
	port := os.Getenv("PORT")
	if port == "" {
		log.Println("cannot start, need a PORT")
		os.Exit(1)
	}
	log.Println("Server running on port", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
}
