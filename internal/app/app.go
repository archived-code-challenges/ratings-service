// Package app implements the tickets service features.
package app

import (
	"github.com/noelruault/ratingsapp/internal/errors"
	"github.com/noelruault/ratingsapp/internal/models"
	"github.com/sirupsen/logrus"
)

var (
	wrap  = errors.Wrapper("app")
	wrapi = errors.WrapInternal
)

// App contains all the application dependencies required by the handlers and other
// methods, as well as general configuration.
type App struct {
	// webserver is the HTTP server for ratingsapp.
	webServer *webServer

	services *models.Services
}

// Config contains settings used to instantiate an App when calling its Configure method.
type Config struct {
	// DSL is the database connection string. This value
	// depends on which database drives is being used. For
	// Postgres, it can look like
	// postgres://user:pass@localhost:5432/ratingsappportal
	// for example.
	DSL string

	// Port is the HTTP server port number as a string.
	Port string

	// JWTSecret is a base64 sequence used to encode
	// and validate JWT tokens
	JWTSecret string
}

// Configure sets the application parameters in the internal struct value. The function will
// start and test the database connection. The dsl is a Postgres connection string, jwtSecret
// is used to encrypt JWT tokens used by end users and jwtDuration indications how long will
// a JWT token be valid. The port may be optionally set to the TCP port where the HTTP server
// should listen, the default being 8000.
//
// The default for jwtDuration is 10 days. The default for jwtSecret is "SAMPLE_DEVELOPMENT_SECRET".
func (a *App) Configure(c *Config) error {
	err := c.check()
	if err != nil {
		return wrap("App.Configure", err)
	}

	a.services, err = models.NewServices(&models.Config{
		JWTSecret:   []byte(c.JWTSecret),
		DatabaseDSL: c.DSL,
	})
	if err != nil {
		return wrap("App.Configure", err)
	}

	// configure services
	a.webServer = newWebServer(c.Port, a.services)

	return nil
}

// Run starts serving the HTTP routes configured with an App object.
// The server listens on port 8000 by default.
func (a *App) Run() error {
	const serviceCount = 1
	err := make(chan error, serviceCount)

	go func() {
		logrus.WithField("addr", a.webServer.server.Addr).Info("HTTP server starts")
		err <- a.webServer.Run()
	}()

	var i int
	for e := range err {
		if e != nil {
			return wrap("error when running application services", e)
		}

		i++
		if i == serviceCount {
			break
		}
	}

	return nil
}

// Shutdown gracefully stops all server services so the process can terminate.
func (a *App) Shutdown() error {
	errWS := a.webServer.Shutdown()
	errSVC := a.services.Close()

	if errWS != nil {
		return wrap("could not stop the webserver service", errWS)
	}
	if errSVC != nil {
		return wrap("could not stop the models services", errSVC)
	}

	return nil
}

// check verifies that the configuration is valid. It may also set default values for fields left
// undefined.
func (c *Config) check() error {
	if c.DSL == "" {
		return wrapi("postgres DSL not defined", nil)
	}
	if c.JWTSecret == "" {
		return wrapi("jwt secret not defined", nil)
	}
	if c.Port == "" {
		c.Port = "8000"
	}

	return nil
}
