package server

import (
	"comparify/internal/handler"
	"context"
	"time"

	"comparify/pkg/logger"

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

	productsGroup.GET("", s.handler.ListItems)
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
	application.Use(requestLogMiddleware())
	application.Use(requestTimeoutMiddleware(requestTimeout))

	return application
}

func requestLogMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			path := c.Path()
			if path == "" {
				path = c.Request().URL.Path
			}

			logger.Logger.Infow("route called",
				"component", "Server.requestLogMiddleware",
				"method", c.Request().Method,
				"path", path,
				"request_id", c.Response().Header().Get(echo.HeaderXRequestID),
			)

			return next(c)
		}
	}
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
