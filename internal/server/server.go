package server

import (
	"flag"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/template/html/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/kabenari/webrtc/internal/handlers"
	w "github.com/kabenari/webrtc/pkg/webrtc"
	"os"
	"time"
)

var (
	addr = flag.String("addr", ":"+os.Getenv("PORT"), "HTTP network address")
	cert = flag.String("cert", "", "")
	key  = flag.String("key", "", "")
)

func Run() error {
	flag.Parse()

	if *addr == "" {
		*addr = ":8080"
	}

	engine := html.New("./views", ".html")
	app := fiber.New(fiber.Config{Views: engine})
	app.Use(cors.New())
	app.Use(logger.New())

	app.Get("/", handlers.Welcome)
	app.Get("/room/create", handlers.RoomCreate)
	app.Get("/room/:uuid", handlers.Room)
	app.Get("/room/:uuid/websocket", websocket.New(handlers.RoomWebsocket, websocket.Config{
		HandshakeTimeout: 10 * time.Second,
	}))
	app.Get("/room/:uuid/chat", handlers.RoomChat)
	app.Get("/room/:uuid/chat/websocket", websocket.New(handlers.RoomChatWebsocket))
	app.Get("/room/:uuid/viewer/websocket", websocket.New(handlers.RoomViewerWebsocket))
	app.Get("/stream/:suuid", handlers.Stream)
	app.Get("/stream/:suuid/websocket", websocket.New(handlers.StreamChatWebsocket, websocket.Config{
		HandshakeTimeout: 10 * time.Second,
	}))
	app.Get("/stream/:suuid/chat/websocket", websocket.New(handlers.StreamWebsocket))
	app.Get("/stream/:suuid/viewer/websocket", websocket.New(handlers.StreamViewerWebsocket))
	app.Static("/", "./assets")

	w.Rooms = make(map[string]*w.Room)
	w.Streams = make(map[string]*w.Room)

	go dispatchKeyFrames()

	if *cert != "" && *key != "" {
		return app.ListenTLS(*addr, *cert, *key)
	}
	return app.Listen(*addr)
}

func dispatchKeyFrames() {
	for range time.NewTicker(time.Second * 3).C {
		for _, room := range w.Rooms {
			room.Peers.DispatchKeyFrame()
		}
	}
}
