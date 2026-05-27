package main

import (
	"log"

	"loop/internal/config"
	"loop/internal/httpserver"
)

func main() {
	cfg := config.Load()
	srv := httpserver.NewServer(cfg)
	if err := srv.Run(); err != nil {
		log.Fatalf("server exited with error: %v", err)
	}
}
