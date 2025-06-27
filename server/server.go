package server

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/ray-remotestate/restro/middlewares"
	"github.com/ray-remotestate/restro/handlers"
	"github.com/ray-remotestate/restro/models"
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
	router := mux.NewRouter()
	authRoutes := router.PathPrefix("/api").Subrouter()
	authRoutes.Use(middlewares.AuthMiddleware)

	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, `{"alive": true}`)
	}).Methods("GET")
	router.HandleFunc("/register", handlers.Register).Methods("POST")
	router.HandleFunc("/refresh", handlers.RefershToken).Methods("POST")
	router.HandleFunc("/login", handlers.Login).Methods("POST")
	authRoutes.HandleFunc("/logout", handlers.Logout).Methods("POST")

	authRoutes.HandleFunc("/restaurants", handlers.ListRestaurants).Methods("GET")
	authRoutes.HandleFunc("/restaurants/{id}/dishes", handlers.GetDishesByRestaurant).Methods("GET")
	authRoutes.HandleFunc("/restaurants/{id}/distance", handlers.GetDistance).Methods("GET")

	// admin only
	admin := authRoutes.PathPrefix("/admin").Subrouter()
	admin.Use(middlewares.RoleBasedMiddleware(models.RoleAdmin))

	admin.HandleFunc("/subadmins", handlers.CreateSubAdmin).Methods("POST")
	admin.HandleFunc("/subadmins", handlers.ListSubAdmins).Methods("GET")

	// admin n subadmin
	adminSub := authRoutes.PathPrefix("/admin").Subrouter()
	adminSub.Use(middlewares.RoleBasedMiddleware(models.RoleAdmin, models.RoleSubAdmin))

	adminSub.HandleFunc("/users", handlers.ListAllUsersBySubAdmin).Methods("GET")
	adminSub.HandleFunc("/resources", handlers.ListResources).Methods("GET")
	adminSub.HandleFunc("/resources", handlers.CreateResource).Methods("POST")

	return &Server {
		Router: router,
	}
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
