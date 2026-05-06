package server

import (
	"comparify/internal/handler"
	"net/http"
)

type Server struct {
	Handler *handler.Handler
}

func NewServer(h *handler.Handler) *Server {
	return &Server{Handler: h}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.Handler.Health)
	mux.HandleFunc("/items/", s.Handler.GetItem)
	mux.HandleFunc("/items/compare", s.Handler.Compare)
	return mux
}
