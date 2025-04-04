package galaxy

import (
	"net/http"
	pb "galaxy.io/server/proto"
)

type ClientConnection interface {
	SendEvent(event *pb.Event) error

	Close()
}


type ConnectionFactory interface {
	NewConnection(w http.ResponseWriter, r *http.Request, operationHandler func(*pb.Operation)) (ClientConnection, error)
}
