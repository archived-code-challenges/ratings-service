package app

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/noelruault/ratingsapp/internal/controllers"
	"github.com/noelruault/ratingsapp/internal/middleware"
	"github.com/noelruault/ratingsapp/internal/models"
)

type webServer struct {
	eng    *gin.Engine
	server http.Server

	staticCtrl *controllers.Static

	mwAuthenticated gin.HandlerFunc
}

func newWebServer(port string, svc *models.Services) *webServer {
	var ws = &webServer{}

	ws.staticCtrl = controllers.NewStatic()

	ws.setupRoutes()
	ws.server = http.Server{
		Addr:         ":" + port,
		Handler:      ws.eng,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	return ws
}

func (ws *webServer) Run() error {
	err := ws.server.ListenAndServe()
	if err != nil {
		if err == http.ErrServerClosed {
			return nil
		}

		return wrap("webServer.Run", err)
	}

	return nil
}

func (ws *webServer) Shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := ws.server.Shutdown(ctx)
	if err != nil {
		return wrap("webServer.Shutdown", err)
	}

	return nil
}

func (ws *webServer) setupRoutes() {
	gin.SetMode(gin.ReleaseMode)
	mux := gin.New()

	mux.Use(middleware.Log)
	mux.Use(gin.Recovery())
	mux.Use(middleware.SecureHeaders)

	// TODO: Authentication

	// restricted handlers
	{
		restricted := mux.Group("/")
		restricted.Use(middleware.ContentType("application/json"))
		restricted.Use(ws.mwAuthenticated)

		{
			apimux := restricted.Group("/api/v1/")

			// TODO: ws routes here...
		}
	}

	mux.NoRoute(ws.staticCtrl.NotFound)

	ws.eng = mux
}
