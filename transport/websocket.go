package websocket

import (
	"log"
	"net/http"
	"time"

	"github.com/awdng/triebwerk/model"
	"github.com/gorilla/websocket"
)

// Connection represents a websocket connection
type Connection struct {
	conn *websocket.Conn
}

// Transport represents the websocket context
type Transport struct {
	upgrader websocket.Upgrader
	register func(conn model.Connection)
}

// NewTransport creates the websocket context
func NewTransport() *Transport {
	return &Transport{
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	}
}

// RegisterNewConnHandler is a callback for new connections
func (t *Transport) RegisterNewConnHandler(register func(conn model.Connection)) {
	t.register = register
}

// Init ...
func (t *Transport) Init() {
	http.HandleFunc("/echo", func(w http.ResponseWriter, r *http.Request) {
		ws, err := t.upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println(err)
			return
		}
		conn := NewConnection(ws)
		t.register(conn)
	})
}

// Run ...
func (t *Transport) Run() error {
	log.Printf("Starting Triebwerk Websocket Server on Port %s...", "8080")
	return http.ListenAndServe(":8080", nil)
}

// NewConnection creates a new connection
func NewConnection(conn *websocket.Conn) *Connection {
	return &Connection{
		conn: conn,
	}
}

// Close sends the websocket CloseMessage
// https://tools.ietf.org/html/rfc6455#section-5.5.1
// graceful == false closes immediatly
func (c *Connection) Close(writeWait time.Duration, graceful bool) {
	if !graceful {
		c.conn.Close()
		return
	}

	c.conn.SetWriteDeadline(time.Now().Add(writeWait))
	c.conn.WriteMessage(websocket.CloseMessage, []byte{})
}

// PrepareRead prepares the websocket connection for reading
func (c *Connection) PrepareRead(maxMessageSize int64, pongWait time.Duration) {
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
}

// Read from the network connection
func (c *Connection) Read() ([]byte, error) {
	_, message, err := c.conn.ReadMessage()
	if err != nil {
		if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
			// TODO: wrap in another error
		}
		return nil, err
	}
	return message, nil
}

// PrepareWrite prepares the websocket connection for writing
func (c *Connection) PrepareWrite(writeWait time.Duration) {
	c.conn.SetWriteDeadline(time.Now().Add(writeWait))
}

// Write to the network connection
func (c *Connection) Write(data []byte) error {
	c.conn.WriteMessage(websocket.CloseMessage, data)
	writer, err := c.conn.NextWriter(websocket.BinaryMessage)
	if err != nil {
		return err
	}
	writer.Write(data)

	// Flush data to the network
	if err := writer.Close(); err != nil {
		return err
	}

	return nil
}

// Ping sends a ping message to the client
func (c *Connection) Ping(writeWait time.Duration) {
	c.conn.SetWriteDeadline(time.Now().Add(writeWait))
	if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
		return
	}
}
