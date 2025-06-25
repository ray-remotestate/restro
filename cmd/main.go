package main

import(
	_"log"
	_"os"
	_"os/signal"
	_"syscall"
	_"time"

	"github.com/sirupsen/logrus"
	"github.com/ray-remotestate/restro/database"
)

// const shutdownTimeOut = 10 * time.Second

func main() {
	// done := make(chan os.Signal, 1)
	// signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	if err := database.ConnectAndMigrate(); err != nil {
		logrus.Panicf("failed to initialize database, error: %v", err)
	}
	logrus.Println("migration is successful")

	// <-done

	// logrus.Info("shutting down...")
	// if err := database.ShutdownDatabase(); err != nil{
	// 	logrus.WithError(err).Error("failed to close database connection!")
	// }

	// logrus.Info("system is shut ..zzz")
}