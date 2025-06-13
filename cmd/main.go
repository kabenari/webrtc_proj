package main

import (
	"github.com/gofiber/fiber/v2/log"
	"github.com/kabenari/webrtc/internal/server"
)

func main() {
	if err := server.Run(); err != nil {
		log.Fatal(err.Error())
	}
}
