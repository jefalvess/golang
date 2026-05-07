package server

import (
	"comparify/internal/handler"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

const apiVersionPrefix = "/v1"

type Server struct {
	echo    *echo.Echo
	handler *handler.Handler
}

func NewServer(productHandler *handler.Handler) *Server {
	application := echo.New()
	application.HideBanner = true
	application.Use(middleware.Recover())
	application.Use(middleware.RequestID())

	server := &Server{
		echo:    application,
		handler: productHandler,
	}
	server.registerRoutes()

	return server
}

func (s *Server) Start(address string) error {
	return s.echo.Start(address)
}

func (s *Server) registerRoutes() {
	v1Group := s.echo.Group(apiVersionPrefix)
	itemsGroup := v1Group.Group("/items")

	v1Group.GET("/health", s.handler.Health)
	itemsGroup.GET("/:id", s.handler.GetItem)
	itemsGroup.GET("/compare", s.handler.Compare)
}
