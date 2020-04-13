package main

import (
	"log"
	"net"
	"net/http"
	"os"
	"strings"

	mylogger "snowplow-schema-server/app/logger"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/cors"
	"github.com/sirupsen/logrus"
	chitrace "gopkg.in/DataDog/dd-trace-go.v1/contrib/go-chi/chi"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

// GetEnvOrDefault gives env variable name or default value
func GetEnvOrDefault(key, defaultValue string) string {
	val := os.Getenv(key)
	if val != "" {
		return val
	}
	return defaultValue
}

var PATH_BASE string = GetEnvOrDefault("PATH_BASE", "/schemas")
var PORT string = GetEnvOrDefault("PORT", "3000")
var ENV string = GetEnvOrDefault("ENV", "dev")

func NewStructuredLogger(logger *logrus.Logger) func(next http.Handler) http.Handler {
	return middleware.RequestLogger(&mylogger.StructuredLogger{Logger: logger})
}

func main() {
	logger := logrus.New()
	logger.Formatter = &logrus.JSONFormatter{
		// disable, as we set our own
		DisableTimestamp: true,
	}
	addr := net.JoinHostPort(
		os.Getenv("DD_AGENT_HOST"),
		os.Getenv("DD_TRACE_AGENT_PORT"),
	)
	// start the tracer with zero or more options
	tracer.Start(tracer.WithServiceName("snowplow-schema-server"), tracer.WithAgentAddr(addr), tracer.WithEnv(ENV))
	defer tracer.Stop()
	router := chi.NewRouter() // init the http tracer
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(NewStructuredLogger(logger))
	router.Use(middleware.Recoverer)
	router.Use(chitrace.Middleware(chitrace.WithServiceName("snowplow-schema-server")))
	router.Use(middleware.Heartbeat("/ping"))
	cors := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		AllowCredentials: true,
		MaxAge:           300, // Maximum value not ignored by any of major browsers
	})
	router.Use(cors.Handler)
	router.Route(PATH_BASE, func(r chi.Router) {
		FileServer(r, "/", http.Dir("./schemas"))
	})

	log.Println("Start listening on port:", PORT)
	err := http.ListenAndServe(":"+PORT, router) // set listen port
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

// FileServer conveniently sets up a http.FileServer handler to serve
// static files from a http.FileSystem.
func FileServer(r chi.Router, path string, root http.FileSystem) {
	if strings.ContainsAny(path, "{}*") {
		panic("FileServer does not permit any URL parameters.")
	}

	if path != "/" && path[len(path)-1] != '/' {
		r.Get(path, http.RedirectHandler(path+"/", 301).ServeHTTP)
		path += "/"
	}
	path += "*"

	r.Get(path, func(w http.ResponseWriter, r *http.Request) {
		rctx := chi.RouteContext(r.Context())
		pathPrefix := strings.TrimSuffix(rctx.RoutePattern(), "/*")
		fs := http.StripPrefix(pathPrefix, http.FileServer(root))
		fs.ServeHTTP(w, r)
	})
}
