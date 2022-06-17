// Copyright 2013 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package api

import (
	"bytes"
	"chatService/config"
	"context"
	"encoding/json"
	"github.com/gorilla/websocket"
	"log"
	"time"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 4096
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

// Client is a middleman between the ws connection and the Hub.
type Client struct {
	Hub *Hub

	// The websocket connection.
	conn *websocket.Conn

	// Buffered channel of outbound messages.
	send chan []byte

	// ID of the user
	id string

	// Access to chat features
	chatService ChatService

	// Whether the Client has sent over auth token
	isAuthenticated bool
}

func NewClient(hub *Hub, conn *websocket.Conn, send chan []byte, id string, chatService ChatService) *Client {
	return &Client{
		Hub:             hub,
		conn:            conn,
		send:            send,
		id:              id,
		isAuthenticated: false,
		chatService:     chatService,
	}
}

// TODO: Possibly look into better way of unmarshalling event

const (
	AddMessage        = 1
	AddParticipant    = 2
	RemoveMessage     = 3
	RemoveParticipant = 4
	Authenticate      = 5
)

// ReadPump pumps messages from the ws connection to the Hub.
//
// The application runs ReadPump in a per-connection goroutine. The application
// ensures that there is at most one reader on a connection by executing all
// reads from this goroutine.
func (c *Client) ReadPump() {
	defer func() {
		c.Hub.unregister <- c
		err := c.conn.Close()
		if err != nil {
			log.Printf("Could not close network connection: %v", err)
		}
	}()
	c.conn.SetReadLimit(maxMessageSize)
	err := c.conn.SetReadDeadline(time.Now().Add(pongWait))
	if err != nil {
		log.Printf("Unable to set read deadline: %v", err)
		return
	}

	c.conn.SetPongHandler(func(string) error {
		err := c.conn.SetReadDeadline(time.Now().Add(pongWait))
		if err != nil {
			log.Printf("Unable to set read deadline: %v", err)
			return err
		}
		return nil
	})

	ctx := context.Background()

	auth, err := config.SetupFirebase().Auth(ctx)
	if err != nil {
		return
	}

	// Starts timer for user to authenticate in 5 seconds
	disconnectTimer := time.NewTimer(30 * time.Second)

	// If user does not authenticate within allotted time then disconnect Client
	go func() {
		<-disconnectTimer.C
		errMessage, _ := json.Marshal("Did not authenticate Client within 5 seconds")
		c.send <- errMessage
		return
	}()

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			return
		}
		message = bytes.TrimSpace(bytes.Replace(message, newline, space, -1))

		var incomingEvent IncomingEvent

		if err := json.Unmarshal(message, &incomingEvent); err != nil {
			log.Printf("Could not process message: %v", err)
			continue
		}

		if c.isAuthenticated {
			switch incomingEvent.RequestType {
			case AddMessage:
				outgoingEvent, err := c.chatService.AddMessage(incomingEvent)
				if err != nil {
					continue
				}

				outgoingEvent.Client = c
				c.Hub.send <- outgoingEvent
			case AddParticipant:
				outgoingEvent, err := c.chatService.AddParticipant(incomingEvent)
				if err != nil {
					continue
				}

				outgoingEvent.Client = c
				c.Hub.send <- outgoingEvent
			case RemoveMessage:
			case RemoveParticipant:

			}
		} else if incomingEvent.RequestType == Authenticate {
			token, err := auth.VerifyIDToken(ctx, incomingEvent.Token)
			if err != nil {
				errMessage, _ := json.Marshal("Token not valid.")
				c.send <- errMessage
				return
			} else if token.UID != c.id {
				errMessage, _ := json.Marshal("Token does not match Client uid")
				c.send <- errMessage
				return
			}
			c.isAuthenticated = true
			// Stops disconnect timer when user is authenticated
			if !disconnectTimer.Stop() {
				<-disconnectTimer.C
			}

		}
	}
}

// WritePump pumps messages from the Hub to the ws connection.
//
// A goroutine running WritePump is started for each connection. The
// application ensures that there is at most one writer to a connection by
// executing all writes from this goroutine.
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		_ = c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			_, _ = w.Write(message)

			// Add queued chat messages to the current ws message.
			n := len(c.send)
			for i := 0; i < n; i++ {
				_, _ = w.Write(newline)
				_, _ = w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
