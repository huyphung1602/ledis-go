package main

import (
	"log"
	"net/http"

	"github.com/zealotnt/ledis-go/handlers"
)

func main() {
	log.Printf("Ledis server started\n")
	addr := ":8080"

	handlers.InitStore()

	go handlers.ExpiredCleaner()

	mux := http.NewServeMux()
	handler := &handlers.LedisHandler{}
	mux.Handle("/", handler)
	mux.Handle("/cli/", http.StripPrefix("/cli/", http.FileServer(http.Dir("./public"))))
	log.Printf("Accepting connections at %s...\n", addr)
	server := http.Server{Handler: mux, Addr: addr}
	log.Fatal(server.ListenAndServe())
}
