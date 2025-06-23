package server

import (
	"flag"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
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

	app := fiber.New()
	app.Use(cors.New())
	app.Use(logger.New())

	// this are all data api handler paths //////////////////////////////////////////////////////////////////////
	api := app.Group("/api")
	api.Get("/room/create", handlers.RoomCreate)
	api.Get("/room/:uuid", handlers.Room)      //returns a json
	api.Get("/stream/:suuid", handlers.Stream) //returns a json
	/////////////////////////////////////////////////////////////////////////////////////////////////////////////

	// websocket handlers ///////////////////////////////////////////////////////////////////////////////////////
	ws := app.Group("/")
	ws.Get("/room/:uuid/websocket", websocket.New(handlers.RoomWebsocket, websocket.Config{
		HandshakeTimeout: 10 * time.Second,
	}))
	ws.Get("/room/:uuid/chat/websocket", websocket.New(handlers.RoomChatWebsocket))
	ws.Get("/room/:uuid/viewer/websocket", websocket.New(handlers.RoomViewerWebsocket))

	ws.Get("/stream/:suuid/websocket", websocket.New(handlers.StreamChatWebsocket, websocket.Config{
		HandshakeTimeout: 10 * time.Second,
	}))
	ws.Get("/stream/:suuid/chat/websocket", websocket.New(handlers.StreamWebsocket))
	ws.Get("/stream/:suuid/viewer/websocket", websocket.New(handlers.StreamViewerWebsocket))

	///////////////////////////////////////////////////////////////////////////////////////////////////////////////

	app.Static("/", "./build")

	// This is the catch-all handler for a Single-Page Application (SPA).
	// It makes sure that all paths that are not matched by an API or WebSocket route
	// will serve the main index.html file of the React app.
	// This allows React Router to handle the routing on the client-side.
	app.Get("/*", func(c *fiber.Ctx) error {
		return c.SendFile("./build/index.html")
	})

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
