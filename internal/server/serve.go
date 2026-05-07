package server

import (
	"comparify/internal/handler"
	"context"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

const (
	apiVersionPrefix = "/v1"
	requestTimeout   = 15 * time.Second
)

type Server struct {
	echo    *echo.Echo
	handler *handler.Handler
}

func NewServer(productHandler *handler.Handler) *Server {
	server := &Server{
		echo:    newEchoApplication(),
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
	productsGroup := v1Group.Group("/products")

	productsGroup.GET("/:id", s.handler.GetItem)
	productsGroup.GET("/compare", s.handler.Compare)
}

// newEchoApplication centraliza middleware comum para manter a criação do servidor previsível.
func newEchoApplication() *echo.Echo {
	application := echo.New()
	application.HideBanner = true
	application.Server.ReadTimeout = requestTimeout
	application.Server.WriteTimeout = requestTimeout
	application.Use(middleware.Recover())
	application.Use(middleware.RequestID())
	application.Use(requestTimeoutMiddleware(requestTimeout))

	return application
}

// requestTimeoutMiddleware adiciona um deadline de timeout ao contexto de cada request.
func requestTimeoutMiddleware(timeout time.Duration) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ctx, cancel := context.WithTimeout(c.Request().Context(), timeout)
			defer cancel()
			c.SetRequest(c.Request().WithContext(ctx))
			return next(c)
		}
	}
}
