package chat

// Hub manages multiple WebSocket connections (clients) and
//facilitates real-time message broadcasting between them.

// clients holds all active WebSocket clients connected to this hub.
// The key is a pointer to a Client, and the value is a boolean indicating
//whether the client is connected.

// broadcast is a channel that carries messages to be sent to all connected clients.

// register is a channel used to register new WebSocket clients with the hub.

// unregister is a channel used to remove WebSocket clients from the hub.

type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
}

// NewHub creates and initializes a new Hub instance.
//
// Returns:
//   - A pointer to an initialized Hub with empty clients and communication channels.

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

// Run starts the event loop for the hub, listening for client registration, unregistration,
// and broadcast messages.

// Behavior:
//   - When a client is registered, it's added to the `clients` map.
//   - When a client is unregistered, it's removed from the `clients` map,
//     and its `Send` channel is closed.
//   - When a message is received on the `broadcast` channel, it is sent
//     to all connected clients.
//   - If a client's `Send` channel is full or closed, the client is
//     removed from the map to prevent resource leaks.

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			// Add the client to the hub's connected clients
			h.clients[client] = true
		case client := <-h.unregister:
			// Remove the client from the hub's connected clients
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.Send)
			}
		case message := <-h.broadcast:
			// Broadcast the message to all connected clients
			for client := range h.clients {
				select {
				case client.Send <- message: // Send the message to the client's personal mailbox that is the send channel
				default:
					close(client.Send)
					delete(h.clients, client)
				}
			}
		}
	}
}
