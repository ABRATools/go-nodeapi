package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/sonarping/go-nodeapi/pkg/routes"
)

func main() {
	// gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	gin.DefaultWriter = os.Stdout
	// set middleware for all groups
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))
	// for later use with auth, etc
	routes.RegisterCentralRoutes(router)

	// for listing, starting, stopping, creating containers
	routes.RegisterContainerRoutes(router)

	// for listing, building, removing images
	routes.RegisterImageRoutes(router)

	// for listing, creating, removing networks and adding/removing containers from networks
	routes.RegisterNetworkingRoutes(router)

	server := &http.Server{
		Addr:         ":8888",
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %s", err)
		}
	}()
	log.Println("Server running on :8888")

	<-quit
	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %s", err)
	}

	log.Println("Server exited")
}
