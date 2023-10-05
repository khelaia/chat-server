package server

import (
	"fmt"
	"net"
	"sync"
)

type Server struct {
	clients map[string]net.Conn
	mutex   sync.Mutex
}

func NewServer() *Server {
	return &Server{
		clients: make(map[string]net.Conn),
	}
}

func (s *Server) Start(port string) {
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		fmt.Println("Error starting server:", err)
		return
	}
	defer listener.Close()

	fmt.Println("Server is listening on port", port)

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}

		go s.handleClient(conn)
	}
}

func (s *Server) handleClient(conn net.Conn) {
	// Implement client handling logic here
	// You can manage user registration, message handling, etc.
	// For simplicity, let's just print the received messages for now.
	defer conn.Close()

	s.mutex.Lock()
	s.clients[conn.RemoteAddr().String()] = conn
	s.mutex.Unlock()

	fmt.Println("New client connected:", conn.RemoteAddr())

	buf := make([]byte, 1024)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			fmt.Println("Client disconnected:", conn.RemoteAddr())
			s.mutex.Lock()
			delete(s.clients, conn.RemoteAddr().String())
			s.mutex.Unlock()
			break
		}

		message := string(buf[:n])
		fmt.Printf("Received message from %s: %s\n", conn.RemoteAddr(), message)

		// Broadcast the message to other clients
		s.broadcast(conn, message)
	}
}

func (s *Server) broadcast(sender net.Conn, message string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	for _, conn := range s.clients {
		if conn != sender {
			_, err := conn.Write([]byte(message))
			if err != nil {
				fmt.Println("Error broadcasting message to", conn.RemoteAddr(), ":", err)
			}
		}
	}
}
