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
	usersCtrl  *controllers.Users
	rolesCtrl  *controllers.Roles

	mwAuthenticated gin.HandlerFunc
}

func newWebServer(port string, svc *models.Services) *webServer {
	var ws = &webServer{}

	ws.mwAuthenticated = middleware.Authenticated(svc.User)

	ws.staticCtrl = controllers.NewStatic()
	ws.usersCtrl = controllers.NewUsers(svc.User)
	ws.rolesCtrl = controllers.NewRoles(svc.Role)

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

	// Authentication
	mux.POST("/api/v1/oauth/token/", ws.usersCtrl.Login)

	// restricted handlers
	{
		restricted := mux.Group("/")
		restricted.Use(middleware.ContentType("application/json"))
		restricted.Use(ws.mwAuthenticated)

		{
			apimux := restricted.Group("/api/v1/")

			ws.setupUsers(apimux)
			ws.setupRoles(apimux)
		}
	}

	mux.NoRoute(ws.staticCtrl.NotFound)

	ws.eng = mux
}

func (ws *webServer) setupUsers(mux *gin.RouterGroup) {
	mux.GET("/users/", middleware.Can(
		models.PermissionReadUsers,
		ws.usersCtrl.List,
	))
	mux.GET("/users/:id", middleware.Can(
		models.PermissionReadUsers,
		ws.usersCtrl.Get,
	))
	mux.POST("/users/", middleware.Can(
		models.PermissionWriteUsers,
		ws.usersCtrl.Create,
	))
	mux.PUT("/users/:id", middleware.Can(
		models.PermissionWriteUsers,
		ws.usersCtrl.Update,
	))
	mux.DELETE("/users/:id", middleware.Can(
		models.PermissionWriteUsers,
		ws.usersCtrl.Delete,
	))
}

func (ws *webServer) setupRoles(mux *gin.RouterGroup) {
	mux.GET("/roles/", middleware.Can(
		models.PermissionReadUsers,
		ws.rolesCtrl.List,
	))
	mux.GET("/roles/:id", middleware.Can(
		models.PermissionReadUsers,
		ws.rolesCtrl.Get,
	))
	mux.POST("/roles/", middleware.Can(
		models.PermissionWriteUsers,
		ws.rolesCtrl.Create,
	))
	mux.PUT("/roles/:id", middleware.Can(
		models.PermissionWriteUsers,
		ws.rolesCtrl.Update,
	))
	mux.DELETE("/roles/:id", middleware.Can(
		models.PermissionWriteUsers,
		ws.rolesCtrl.Delete,
	))
}
