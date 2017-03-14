package websocket

import (
	"fmt"
	"net/http"

	"github.com/go-kit/kit/log"
	"github.com/gorilla/websocket"
)

// ProtocolHandler is an interface defining the needed functionality to Decode and Encode messages
type ProtocolHandler interface {
	Decode(message []byte) (service [3]byte, method [3]byte, data interface{}, err error)
	Encode(service [3]byte, method [3]byte, data interface{}) (message []byte, err error)
}

// Server is a struct type containing every value needed for a websocket server
type Server struct {
	protocolHandler ProtocolHandler
	logger          log.Logger
	services        map[[3]byte]*ServiceDescription
}

// RegisterService adds the given ServiceDescription to the Server's services map
func (s *Server) RegisterService(sd *ServiceDescription) error {
	_, exist := s.services[sd.protocolName]
	if exist {
		return fmt.Errorf("Service Endpoint %s already exists", sd.protocolName)
	}

	s.services[sd.protocolName] = sd
	return nil
}

// Serve starts the http transport for the websocket, listening on addr
func (s *Server) Serve(addr string) error {
	return http.ListenAndServe(addr, s)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Log(err)
		return
	}

	go s.handleConnection(conn)
}

func (s *Server) handleConnection(conn *websocket.Conn) {
	for {
		messageType, request, err := conn.ReadMessage()
		if err != nil {
			conn.Close()
			return
		}

		srv, me, data, err := s.protocolHandler.Decode(request)
		if err != nil {
			err = conn.WriteMessage(messageType, []byte(err.Error()))
			if err != nil {
				s.logger.Log(err)
				conn.Close()
				return
			}
			continue
		}

		handler, err := s.services[srv].EndpointHandler(me)
		if err != nil {
			err = conn.WriteMessage(messageType, []byte(err.Error()))
			if err != nil {
				s.logger.Log(err)
				conn.Close()
				return
			}
			continue
		}

		res, err := handler(data)
		if err != nil {
			err = conn.WriteMessage(messageType, []byte(err.Error()))
			if err != nil {
				s.logger.Log(err)
				conn.Close()
				return
			}
			continue
		}

		response, err := s.protocolHandler.Encode(srv, me, res)
		if err != nil {
			err = conn.WriteMessage(messageType, []byte(err.Error()))
			if err != nil {
				s.logger.Log(err)
				conn.Close()
				return
			}
			continue
		}

		err = conn.WriteMessage(messageType, response)
		if err != nil {
			s.logger.Log(err)
			conn.Close()
			return
		}
	}
}

// NewServer returns a pointer to a Server instance
func NewServer(
	ph ProtocolHandler,
	logger log.Logger,
) *Server {
	return &Server{
		protocolHandler: ph,
		logger:          logger,
	}
}
