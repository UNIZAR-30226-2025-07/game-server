package websockets

import (
	"log"
	"net/http"

	"galaxy.io/server/galaxy"
	pb "galaxy.io/server/proto"
	"google.golang.org/protobuf/proto"
)

type Client struct {
	conn *Connection
}

func (c *Client) SendEvent(event *pb.Event) error {
	data, err := proto.Marshal(event)
	if err != nil {
		return err
	}
	return c.conn.SendBinary(data)
}

func (c *Client) Close() {
	c.conn.Close()
}

type WebsocketFactory struct{}

func (f *WebsocketFactory) NewConnection(
	w http.ResponseWriter,
	r *http.Request,
	operationHandler func(*pb.Operation),
) (galaxy.ClientConnection, error) {
	handler := func(data []byte)  {
		operation := &pb.Operation{}
		err := proto.Unmarshal(data, operation)
		if err != nil {
			log.Printf("Error unmarshing operation: %v", err)
			return
		}
		operationHandler(operation)
	}

	conn, err := Upgrade(w, r, handler)
	if err != nil {
		return nil, err
	}

	return &Client{conn: conn}, nil
}
