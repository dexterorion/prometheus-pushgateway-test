package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	restful "github.com/emicklei/go-restful/v3"
	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
)

type User struct {
	Id   int
	Name string
}

var (
	duration = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "request_duration_seconds",
		Help: "The duration of the request.",
	})
)

//
// This example shows how to log both the request body and response body

func main() {
	restful.Add(newUserService())
	log.Print("start listening on localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func newUserService() *restful.WebService {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	promgate := os.Getenv("PUSHGATEWAYHOST")

	// now do something with s3 or whatever

	fmt.Printf("Prometheus gateway url: %s\n", promgate)

	// We use a registry here to benefit from the consistency checks that
	// happen during registration.
	registry := prometheus.NewRegistry()
	registry.MustRegister(duration)
	// Note that successTime is not registered.

	pusher := push.New(promgate, "request_type").Gatherer(registry)

	if err := pusher.Add(); err != nil {
		fmt.Println("Could not add to Pushgateway:", err)
	}

	if err := pusher.Push(); err != nil {
		fmt.Println("Could not push to Pushgateway:", err)
	}

	ws := new(restful.WebService)
	ws.Filter(HTTPFilter())

	ws.
		Path("/users").
		Consumes(restful.MIME_XML, restful.MIME_JSON).
		Produces(restful.MIME_JSON, restful.MIME_XML)

	ws.Route(ws.POST("/").To(createUser))
	return ws
}

// curl -H "content-type:application/json" http://localhost:8080/users -d '{"Id":42, "Name":"Captain Marvel"}'
//
func createUser(request *restful.Request, response *restful.Response) {
	u := new(User)
	err := request.ReadEntity(u)
	log.Print("createUser", err, u)
	response.WriteEntity(u)
}

func HTTPFilter() restful.FilterFunction {
	return func(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
		start := time.Now()
		log.Printf("Starting at... %d\n\n", start.Unix())

		chain.ProcessFilter(req, resp)

		end := time.Now()
		log.Printf("Finishing at... %d\n\n", end.Unix())

		log.Printf("Diff ... %d\n\n", end.Sub(start).Nanoseconds())
		log.Printf("Diff float... %f\n\n", float64(end.Sub(start).Nanoseconds()))

		log.Printf("Response is ... %d\n\n", resp.StatusCode())

		// Note that time.Since only uses a monotonic clock in Go1.9+.
		duration.Set(float64(end.Sub(start).Nanoseconds()))
	}
}
