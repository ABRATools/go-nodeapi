package main

import (
	"bytes"
	"context"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/sonarping/go-nodeapi/pkg/routes"
)

type bodyLogWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *bodyLogWriter) Write(b []byte) (int, error) {
	w.body.Write(b)                  // capture for logging
	return w.ResponseWriter.Write(b) // write out as normal
}

func LoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		var reqBody []byte
		if c.Request.Body != nil {
			reqBody, _ = io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(reqBody))
		}

		blw := &bodyLogWriter{body: new(bytes.Buffer), ResponseWriter: c.Writer}
		c.Writer = blw

		// let the handler ruN
		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()
		clientIP := c.ClientIP()
		method := c.Request.Method
		path := c.Request.URL.Path
		respBody := blw.body.String()

		log.Printf(
			`{"time":"%s", "client_ip":"%s", "method":"%s", "path":"%s", `+
				`"status":%d, "latency_ms":%d, "request":"%s", "response":"%s"}`,
			start.Format(time.RFC3339),
			clientIP,
			method,
			path,
			status,
			latency.Milliseconds(),
			sanitize(reqBody),
			sanitize([]byte(respBody)),
		)
	}
}

func sanitize(b []byte) string {
	s := string(b)
	s = string(bytes.ReplaceAll([]byte(s), []byte{'\n'}, []byte{' '}))
	s = string(bytes.ReplaceAll([]byte(s), []byte{'"'}, []byte{'`'}))
	return s
}

func main() {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	gin.DefaultWriter = os.Stdout
	// set middleware for all groups
	router.Use(LoggingMiddleware(), gin.Recovery())
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

	// for listing, starting, stopping, removing ebpf services
	routes.RegisterEBPFRoutes(router)

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
