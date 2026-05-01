package main

import (
	"log"
	"net/http"
	"time"

	"github.com/Fischris/paperless-scanner/internal/configuration"
	"github.com/Fischris/paperless-scanner/internal/httpserver"
	"github.com/Fischris/paperless-scanner/internal/scanner"
)

func main() {
	scannerConfiguration, configurationError := configuration.LoadScannerConfiguration()
	if configurationError != nil {
		log.Fatalf("configuration error: %v", configurationError)
	}

	log.Printf("starting scanner service")
	log.Printf("listen port: %s", scannerConfiguration.ListenPort)
	log.Printf("target directory: %s", scannerConfiguration.TargetDirectory)
	log.Printf("scan resolution: %s dpi", scannerConfiguration.ScanResolutionDPI)

	scannerService := scanner.NewService(scannerConfiguration)

	if scannerConfiguration.ScannerDevice == "" {
		log.Printf("SCANNER_DEVICE is not set")

		discoveryError := scannerService.DiscoverScanners()
		if discoveryError != nil {
			log.Fatalf("scanner discovery failed: %v", discoveryError)
		}

		log.Fatalf("SCANNER_DEVICE is required")
	}

	log.Printf("scanner device: %s", scannerConfiguration.ScannerDevice)

	httpHandler := httpserver.NewHandler(scannerService, scannerConfiguration.AuthToken)

	listenAddress := ":" + scannerConfiguration.ListenPort
	log.Printf("listening on %s", listenAddress)

	httpService := &http.Server{
		Addr:              listenAddress,
		Handler:           httpHandler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	httpServiceError := httpService.ListenAndServe()
	if httpServiceError != nil {
		log.Fatalf("server failed: %v", httpServiceError)
	}
}
