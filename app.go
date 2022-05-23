package main

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"net"

	fthealth "github.com/Financial-Times/go-fthealth/v1_1"
	log "github.com/Financial-Times/go-logger"
	"github.com/Financial-Times/http-handlers-go/httphandlers"
	"github.com/Financial-Times/public-brands-api/v4/brands"
	status "github.com/Financial-Times/service-status-go/httphandlers"
	"github.com/gorilla/mux"
	cli "github.com/jawher/mow.cli"
	_ "github.com/joho/godotenv/autoload"
	"github.com/rcrowley/go-metrics"
)

var httpClient = http.Client{
	Transport: &http.Transport{
		MaxIdleConnsPerHost: 128,
		Dial: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).Dial,
	},
}

func main() {
	app := cli.App("public-brands-api", "A public RESTful API for accessing Brands in neo4j")
	appSystemCode := app.String(cli.StringOpt{
		Name:   "app-system-code",
		Value:  "public-brands-api",
		Desc:   "System Code of the application",
		EnvVar: "APP_SYSTEM_CODE",
	})
	appName := app.String(cli.StringOpt{
		Name:   "appName",
		Value:  "public-brands-api",
		Desc:   "Name of the service",
		EnvVar: "APP_NAME",
	})
	neoURL := app.String(cli.StringOpt{
		Name:   "neo-url",
		Value:  "http://localhost:7474/db/data",
		Desc:   "neo4j endpoint URL",
		EnvVar: "NEO_URL",
	})
	port := app.String(cli.StringOpt{
		Name:   "port",
		Value:  "8080",
		Desc:   "Port to listen on",
		EnvVar: "PORT",
	})
	logLevel := app.String(cli.StringOpt{
		Name:   "log-level",
		Value:  "info",
		Desc:   "Logging level (DEBUG, INFO, WARN, ERROR)",
		EnvVar: "LOG_LEVEL",
	})
	cacheDuration := app.String(cli.StringOpt{
		Name:   "cache-duration",
		Value:  "1h",
		Desc:   "Duration Get requests should be cached for. e.g. 2h45m would set the max-age value to '7440' seconds",
		EnvVar: "CACHE_DURATION",
	})
	healthcheckInterval := app.String(cli.StringOpt{
		Name:   "healthcheck-interval",
		Value:  "30s",
		Desc:   "How often the Neo4j healthcheck is called.",
		EnvVar: "HEALTHCHECK_INTERVAL",
	})
	conceptsApiUrl := app.String(cli.StringOpt{
		Name:   "conceptsApiUrl",
		Value:  "http://localhost:8080",
		Desc:   "Url of public concepts api",
		EnvVar: "CONCEPTS_API",
	})

	app.Action = func() {
		log.Infof("public-brands-api will listen on port: %s, connecting to: %s", *port, *neoURL)
		runServer(*appName, *port, *appSystemCode, *cacheDuration, *conceptsApiUrl)
	}

	log.InitLogger(*appSystemCode, *logLevel)
	log.WithFields(map[string]interface{}{
		"HEALTHCHECK_INTERVAL": *healthcheckInterval,
		"CACHE_DURATION":       *cacheDuration,
		"NEO_URL":              *neoURL,
		"LOG_LEVEL":            *logLevel,
	}).Info("Starting app with arguments")
	log.Infof("Application started with args %s", os.Args)
	app.Run(os.Args)
}

func runServer(appName string, port string, systemCode string, cacheDuration string, conceptsAPIURL string) {
	if duration, durationErr := time.ParseDuration(cacheDuration); durationErr != nil {
		log.Fatalf("Failed to parse cache duration string, %v", durationErr)
	} else {
		brands.CacheControlHeader = fmt.Sprintf("max-age=%s, public", strconv.FormatFloat(duration.Seconds(), 'f', 0, 64))
	}

	servicesRouter := mux.NewRouter()

	handler := brands.NewHandler(&httpClient, appName, systemCode, conceptsAPIURL)

	// Healthchecks and standards first
	healthCheck := fthealth.TimedHealthCheck{
		HealthCheck: fthealth.HealthCheck{
			SystemCode:  systemCode,
			Name:        "PublicBrandsRead Healthcheck",
			Description: "Checks downstream services health",
			Checks:      []fthealth.Check{handler.HealthCheck()},
		},
		Timeout: 10 * time.Second,
	}

	servicesRouter.HandleFunc("/__health", fthealth.Handler(healthCheck))

	// Then API specific ones:
	handler.RegisterHandlers(servicesRouter)

	var monitoringRouter http.Handler = servicesRouter
	monitoringRouter = httphandlers.TransactionAwareRequestLoggingHandler(log.Logger(), monitoringRouter)
	monitoringRouter = httphandlers.HTTPMetricsHandler(metrics.DefaultRegistry, monitoringRouter)

	http.HandleFunc(status.BuildInfoPath, status.BuildInfoHandler)
	http.HandleFunc(status.BuildInfoPathDW, status.BuildInfoHandler)
	servicesRouter.HandleFunc(status.GTGPath, status.NewGoodToGoHandler(handler.GTG))
	http.Handle("/", monitoringRouter)

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Unable to start server: %v", err)
	}

}
