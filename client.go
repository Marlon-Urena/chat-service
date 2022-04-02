// Copyright 2013 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/gorilla/websocket"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	hub *Hub

	// The websocket connection.
	conn *websocket.Conn

	// Buffered channel of outbound messages.
	send chan []byte
}

type Message struct {
	Id          string    `json:"id,omitempty"`
	SenderId    string    `json:"senderId" firestore:"senderId"`
	ContentType string    `json:"contentType" firestore:"contentType"`
	Body        string    `json:"body" firestore:"body"`
	CreatedAt   time.Time `json:"createdAt" firestore:"createdAt"`
	Attachments []string  `json:"attachments,omitempty"`
}

type NewEvent struct {
	ConversationId string
	RequestType    int
	Message        *Message
	Participants   []string
}

type Conversation struct {
	Participants     []string           `json:"participants" firestore:"participants"`
	ConversationType string             `json:"conversationType" firestore:"conversationType"`
	Messages         map[string]Message `json:"messages" firestore:"messages"`
}

const (
	AddMessage        int = 1
	AddParticipant        = 2
	RemoveMessage         = 3
	RemoveParticipant     = 4
)

// readPump pumps messages from the websocket connection to the hub.
//
// The application runs readPump in a per-connection goroutine. The application
// ensures that there is at most one reader on a connection by executing all
// reads from this goroutine.
func (c *Client) readPump(firestoreClient *firestore.Client) {
	defer func() {
		c.hub.unregister <- c
		err := c.conn.Close()
		if err != nil {
			log.Printf("Could not close network connection: %v", err)
			return
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

	defer func(client *firestore.Client) {
		err := client.Close()
		if err != nil {
			log.Printf("Unable to close resources held by firestoreClient: %v", err)
		}
	}(firestoreClient)

	ctx := context.Background()

	for {
		_, message, err := c.conn.ReadMessage()

		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}
		message = bytes.TrimSpace(bytes.Replace(message, newline, space, -1))

		var newEvent NewEvent

		if err := json.Unmarshal(message, &newEvent); err != nil {
			log.Fatalf("Could not process message: %v", err)
		}

		switch newEvent.RequestType {
		case AddMessage:
			messageData := newEvent.Message
			conversationRef := firestoreClient.Collection("conversations").Doc(newEvent.ConversationId)
			messageRef, wr, err := conversationRef.Collection("messages").Add(context.Background(), map[string]interface{}{
				"senderId":    messageData.SenderId,
				"body":        messageData.Body,
				"contentType": messageData.ContentType,
				"createdAt":   firestore.ServerTimestamp,
			})

			if err != nil {
				log.Printf("Unable to add new document: %v", err)
			}

			log.Printf("Created document with  reference #: %s\n", (*messageRef).ID)
			log.Printf("Write Result contains %v\n", (*wr).UpdateTime)
		case AddParticipant:
			newParticipants := newEvent.Participants
			conversationRef := firestoreClient.Collection("conversations").Doc(newEvent.ConversationId)

			conversationSnap, err := conversationRef.Get(ctx)
			if status.Code(err) == codes.NotFound {
				log.Printf("Unable to find conversationRef with id %s", newEvent.ConversationId)
			}

			var conversation Conversation
			if err = conversationSnap.DataTo(conversation); err != nil {
				log.Println(err)
			}

			currentParticipants := conversation.Participants

			// Checks if new participants are already participating in the conversationRef
			for i, newParticipant := range newParticipants {
				for _, currentParticipant := range currentParticipants {
					// If new participant exists in currentParticipant list then remove it to avoid duplicates
					if newParticipant == currentParticipant {
						newParticipants = append(newParticipants[:i], newParticipants[i+1:]...)
					}
				}
			}

			wr, err := conversationRef.Update(ctx, []firestore.Update{
				{
					Path:  "participants",
					Value: newParticipants,
				},
			})

			if err != nil {
				log.Printf("Unable to update participants: %v", err)
			}

			log.Printf("Write Result contains %v\n", (*wr).UpdateTime)
		case RemoveMessage:
		case RemoveParticipant:
		}

		c.hub.broadcast <- message
	}
}

// writePump pumps messages from the hub to the websocket connection.
//
// A goroutine running writePump is started for each connection. The
// application ensures that there is at most one writer to a connection by
// executing all writes from this goroutine.
func (c *Client) writePump(firestoreClient *firestore.Client) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued chat messages to the current websocket message.
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write(newline)
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// serveWs handles websocket requests from the peer.
func serveWs(hub *Hub, w http.ResponseWriter, r *http.Request) {
	firestoreClient := r.Context().Value("firestore").(*firestore.Client)

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	client := &Client{hub: hub, conn: conn, send: make(chan []byte, 256)}
	client.hub.register <- client

	// Allow collection of memory referenced by the caller by doing all work in
	// new goroutines.
	go client.writePump(firestoreClient)
	go client.readPump(firestoreClient)
}
