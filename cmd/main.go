package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/ray-remotestate/restro/database"
	"github.com/ray-remotestate/restro/server"
)

const shutdownTimeOut = 10 * time.Second

func main() {
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	svr := server.SetupRoutes()

	if err := database.ConnectAndMigrate(); err != nil {
		logrus.Panicf("failed to initialize database, error: %v", err)
	}
	logrus.Println("migration is successful")

	func() {
		log.Println("Server starting at :8080")
		if err := svr.Run(":8080"); err != nil {
			logrus.Panicf("Server didn't start! %+v", err)
		}
	}()

	<-done

	logrus.Info("shutting down server...")
	if err := database.ShutdownDatabase(); err != nil{
		logrus.WithError(err).Error("failed to close database connection!")
	}
	if err := svr.Shutdown(shutdownTimeOut); err != nil {
		logrus.WithError(err).Error("failed to gracefully shutdown server")
	}

	logrus.Info("system is shut ...zzz")
}