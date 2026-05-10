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
	cfg      config.Config
	router   *gin.Engine
	services []container.BackgroundService
}

func New(cfg config.Config, appContainer *container.AppContainer) *Server {
	router := routes.SetupRouter(appContainer)
	services := []container.BackgroundService(nil)
	if appContainer != nil {
		services = appContainer.BackgroundServices
	}

	return &Server{
		cfg:      cfg,
		router:   router,
		services: services,
	}
}

func (s *Server) Run() error {
	appCtx, cancelWorkers := context.WithCancel(context.Background())
	defer cancelWorkers()

	for _, service := range s.services {
		if service == nil {
			continue
		}
		go service.Run(appCtx)
	}

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
	cancelWorkers()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return srv.Shutdown(ctx)
}
