package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"vekr/internal/config"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "3300"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Vekr is running. (%s, default env %s)\n", config.Env(), config.DefaultEnv)
	})

	addr := ":" + port
	log.Printf("listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}
