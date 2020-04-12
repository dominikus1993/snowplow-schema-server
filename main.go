package main

import (
	"net"
	"net/http"
	"os"

	log "github.com/sirupsen/logrus"
	httptrace "gopkg.in/DataDog/dd-trace-go.v1/contrib/net/http"
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

var PATH_BASE string = GetEnvOrDefault("PATH_BASE", "/")
var PORT string = GetEnvOrDefault("PORT", "3000")
var ENV string = GetEnvOrDefault("ENV", "dev")

func main() {

	log.SetFormatter(&log.JSONFormatter{})
	addr := net.JoinHostPort(
		os.Getenv("DD_AGENT_HOST"),
		os.Getenv("DD_TRACE_AGENT_PORT"),
	)
	// start the tracer with zero or more options
	tracer.Start(tracer.WithServiceName("snowplow-schema-server"), tracer.WithAgentAddr(addr), tracer.WithEnv(ENV))
	defer tracer.Stop()
	mux := httptrace.NewServeMux() // init the http tracer

	mux.Handle(PATH_BASE, http.FileServer(http.Dir("./schemas")))

	log.Println("Start listening on port:", PORT)
	err := http.ListenAndServe(":"+PORT, mux) // set listen port
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
