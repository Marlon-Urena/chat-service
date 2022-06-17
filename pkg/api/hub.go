package api

import (
	"encoding/json"
	"log"
)

// Hub maintains the set of active clients and broadcasts messages to the clients.
type Hub struct {
	// Registered clients.
	clients map[string][]*Client

	// Inbound message to all clients.
	broadcast chan []byte

	// Register requests from the clients.
	Register chan *Client

	// Unregister requests from clients.
	unregister chan *Client

	// Inbound message to specified clients.
	send chan OutgoingEvent
}

func NewHub() *Hub {
	return &Hub{
		broadcast:  make(chan []byte),
		send:       make(chan OutgoingEvent),
		Register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[string][]*Client),
	}
}

func (h *Hub) Run() {
	for {
		select {
		// Register Client
		case client := <-h.Register:
			h.clients[client.id] = append(h.clients[client.id], client)
		// Unregister Client
		case client := <-h.unregister:
			// Check if Client with uid exists
			if _, ok := h.clients[client.id]; ok {
				// Search for Client that matches Client reference
				for i := 0; i < len(h.clients[client.id]); i++ {
					// If Client found then removing from array
					if client == h.clients[client.id][i] {
						length := len(h.clients[client.id]) - 1

						// Remove element at position i
						h.clients[client.id][i] = h.clients[client.id][length]
						h.clients[client.id][length] = nil
						h.clients[client.id] = h.clients[client.id][:length]

						// If no clients exist with id then remove key from clients map
						if len(h.clients[client.id]) == 0 {
							delete(h.clients, client.id)
						}
						break
					}
				}
				close(client.send)
			}
		// Send message to all clients
		case message := <-h.broadcast:
			for uid := range h.clients {
				for i, client := range h.clients[uid] {
					select {
					case client.send <- message:
					default:
						close(client.send)

						length := len(h.clients[client.id]) - 1

						// Remove element at position i
						h.clients[client.id][i] = h.clients[client.id][length]
						h.clients[client.id][length] = nil
						h.clients[client.id] = h.clients[client.id][:length]

						// If no clients exist with id then remove key from clients map
						if len(h.clients[uid]) == 0 {
							delete(h.clients, uid)
						}
					}
				}
			}
		// Send message to all participants of a conversation
		case outgoingEvent := <-h.send:
			currentClient := outgoingEvent.Client
			outgoingEvent.Client = nil

			message, err := json.Marshal(outgoingEvent)
			if err != nil {
				log.Printf("Could not process outgoing message: %v", err)
			}

			// Send message to all participants of conversation
			for _, uid := range outgoingEvent.Participants {
				for i, client := range h.clients[uid] {
					if currentClient != client {
						select {
						case client.send <- message:
						default:
							close(client.send)

							length := len(h.clients[client.id]) - 1

							// Remove element at position i
							h.clients[client.id][i] = h.clients[client.id][length]
							h.clients[client.id][length] = nil
							h.clients[client.id] = h.clients[client.id][:length]

							// If no clients exist with id then remove key from clients map
							if len(h.clients[uid]) == 0 {
								delete(h.clients, uid)
							}
						}
					}

				}
			}
		}

	}
}
