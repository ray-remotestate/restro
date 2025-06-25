package server

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

type Server struct {
	Router *mux.Router
	server *http.Server
}

const (
	readTimeout		  = 5 * time.Minute
	readHeaderTimeout = 30 * time.Second
	writeTimeout	  = 5 * time.Minute
)

func SetupRoutes() *Server {
	router := mux.NewRouter
	
}

func (svr *Server) Run(port string) error {
	svr.server = &http.Server{
		Addr:	port,
		Handler: svr.Router,
		ReadTimeout: readTimeout,
		ReadHeaderTimeout: readHeaderTimeout,
		WriteTimeout: writeTimeout,
	}
	return svr.server.ListenAndServe()
}

func (svr *Server) Shutdown(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return svr.server.Shutdown(ctx)
}