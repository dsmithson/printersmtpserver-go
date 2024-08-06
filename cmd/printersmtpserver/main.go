// main.go
package main

import (
	"context"
	"os"
	"os/signal"
	"printersmtpserver/internal/config"
	"printersmtpserver/internal/smtp"
	"syscall"

	"github.com/sirupsen/logrus"
)

func main() {
	// Configure Logrus
	logrus.SetFormatter(&logrus.JSONFormatter{})
	logrus.SetOutput(os.Stdout)
	logrus.SetLevel(logrus.InfoLevel)

	// Load settings
	settings, err := config.LoadSettings(os.Args[1:])
	if err != nil {
		logrus.WithError(err).Fatal("Failed to load settings.")
		return
	}
	if settings.FilePath == "" {
		logrus.Fatal("Output path not specified.")
		return
	}

	logrus.Info("Starting up...")
	logrus.Infof("Listening for SMTP requests on port %d", settings.SmtpPort)
	logrus.Infof("Email attachments will be saved to %s", settings.FilePath)

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigs
		logrus.Info("Process exit requested.")
		cancel()
	}()

	// Start relay server
	relayServer := smtp.NewHomePrinterRelay(&settings)
	if err := relayServer.Startup(); err != nil {
		logrus.WithError(err).Fatal("Failed to start relay server.")
		return
	}
	defer relayServer.Shutdown()

	// Wait indefinitely or until the context is cancelled
	<-ctx.Done()
	logrus.Info("Shutdown initiated.")
	relayServer.Shutdown()
	logrus.Info("Shutdown complete.")

	// Ensure to flush logs and perform any final cleanup
	logrus.Info("Application exiting...")
}
