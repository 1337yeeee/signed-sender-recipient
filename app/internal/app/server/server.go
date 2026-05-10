package server

import (
	"context"
	"electronic-digital-signature/internal/app/config"
	"electronic-digital-signature/internal/app/container"
	"electronic-digital-signature/internal/app/routes"
	"errors"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"
)

type Server struct {
	cfg    config.Config
	router *gin.Engine
}

func New(cfg config.Config, container *container.AppContainer) *Server {
	router := routes.SetupRouter(container)

	return &Server{
		cfg:    cfg,
		router: router,
	}
}

func (s *Server) Run() error {
	srv := &http.Server{
		Addr:    ":" + s.cfg.APIPort,
		Handler: s.router,
	}

	go func() {
		log.Printf("server running on port %s", s.cfg.APIPort)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)

	<-quit
	log.Println("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return srv.Shutdown(ctx)
}
