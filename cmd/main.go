package main

import (
	"fmt"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"os"
	"path"
	"time"
)

func main() {
	// Setup HTTP Routes
	r := mux.NewRouter()
	r.PathPrefix("/").Handler(http.FileServer(http.Dir("./web/")))
	// api := r.PathPrefix("/api").Subrouter()

	r.HandleFunc("/", HomeHandler)

	// Start HTTP Server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		log.Printf("Defaulting to port %s", port)
	}

	log.Printf("Listening on port %s", port)
	log.Printf("Open http://localhost:%s in the browser", port)

	server := &http.Server{
		Handler:      r,
		Addr:         fmt.Sprintf(":%s", port),
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	log.Fatal(server.ListenAndServe())
}

func HomeHandler(w http.ResponseWriter, r *http.Request) {
	p := path.Dir("./web/index.html")
	w.Header().Set("Content-Type", "text/html")
	http.ServeFile(w, r, p)
}
