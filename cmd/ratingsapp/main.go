package main

import (
	"flag"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/noelruault/ratingsapp/internal/app"

	"github.com/sirupsen/logrus"
)

var VERSION string

var (
	application  app.App
	shuttingDown sync.Mutex // helps to keep the main goroutine from ending before the shutdown function is completely executed
)

// handleSignals runs on a separate goroutine and waits for termination signals to
// come, gracefully shutting down the application.
func handleSignals() {
	// set up signal catching
	signals := make(chan os.Signal)
	signal.Notify(signals, syscall.SIGINT)
	signal.Notify(signals, syscall.SIGTERM)

	// wait for signals to come in
	<-signals

	// shuts down the servers
	shuttingDown.Lock()
	err := application.Shutdown()
	shuttingDown.Unlock()
	if err != nil {
		logrus.WithError(err).Fatal("Failed to shutdown gracefully")
	}
}

// configureLogger sets up the logger variable before it should be used by the application. It will
// set the logging output to the standard output and will define the logging level depending on the
// verbose setting.
func configureLogger(verbose bool) {
	logrus.SetOutput(os.Stdout)
	logrus.SetReportCaller(true)

	if verbose {
		logrus.SetLevel(logrus.DebugLevel)
	} else {
		logrus.SetLevel(logrus.InfoLevel)
	}
}

// main sets up the gateway with a auxiliary JSON config file and brings up the
// servers and services.
func main() {
	// get CLI flags
	verbose := flag.Bool("v", false, "verbose logging")
	flag.Parse()

	// configure logger
	configureLogger(*verbose)

	// signal treatment
	go handleSignals()

	// greet the terminal
	logrus.WithField("version", VERSION).Info("Ratings App server starting")

	// configures the gateway application
	err := application.Configure(&app.Config{
		DSL:       os.Getenv("RATINGSAPP_POSTGRES_DSL"),
		JWTSecret: os.Getenv("RATINGSAPP_JWT_SECRET"),
		Port:      os.Getenv("PORT"),
	})
	if err != nil {
		logrus.WithError(err).Fatal("Failed to configure application")
	}

	// runs the application servers
	err = application.Run()
	if err != nil {
		logrus.WithError(err).Error("Error when running the application")
	}

	shuttingDown.Lock()
	logrus.Info("Server will exit now.")
}
