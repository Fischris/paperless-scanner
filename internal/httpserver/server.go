package httpserver

import (
	"log"
	"net/http"
	"time"

	"github.com/Fischris/paperless-scanner/internal/scanner"
)

type ScannerHTTPServer struct {
	scannerService *scanner.Service
	authToken      string
}

func NewHandler(scannerService *scanner.Service, authToken string) http.Handler {
	scannerHTTPServer := &ScannerHTTPServer{
		scannerService: scannerService,
		authToken:      authToken,
	}

	authenticatedServeMux := http.NewServeMux()
	authenticatedServeMux.HandleFunc("/scan/flatbed", scannerHTTPServer.handleFlatbedScan)
	authenticatedServeMux.HandleFunc("/scan/adf", scannerHTTPServer.handleADFScan)

	rootServeMux := http.NewServeMux()
	rootServeMux.HandleFunc("/healthz", scannerHTTPServer.handleHealthz)
	rootServeMux.Handle("/", scannerHTTPServer.authMiddleware(authenticatedServeMux))

	return loggingMiddleware(rootServeMux)
}

func (scannerHTTPServer *ScannerHTTPServer) handleHealthz(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w, http.MethodGet)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok\n"))
}

func (scannerHTTPServer *ScannerHTTPServer) handleFlatbedScan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w, http.MethodPost)
		return
	}

	if !scannerHTTPServer.scannerService.TryAcquireScanSlot() {
		http.Error(w, "scan already in progress\n", http.StatusConflict)
		return
	}

	go func() {
		defer scannerHTTPServer.scannerService.ReleaseScanSlot()

		flatbedScanError := scannerHTTPServer.scannerService.RunFlatbedScan()
		if flatbedScanError != nil {
			log.Printf("flatbed scan failed: %v", flatbedScanError)
			return
		}

		log.Printf("flatbed scan completed")
	}()

	w.WriteHeader(http.StatusAccepted)
	_, _ = w.Write([]byte("flatbed scan accepted\n"))
}

func (scannerHTTPServer *ScannerHTTPServer) handleADFScan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w, http.MethodPost)
		return
	}

	if !scannerHTTPServer.scannerService.TryAcquireScanSlot() {
		http.Error(w, "scan already in progress\n", http.StatusConflict)
		return
	}

	go func() {
		defer scannerHTTPServer.scannerService.ReleaseScanSlot()

		adfScanError := scannerHTTPServer.scannerService.RunADFScan()
		if adfScanError != nil {
			log.Printf("adf scan failed: %v", adfScanError)
			return
		}

		log.Printf("adf scan completed")
	}()

	w.WriteHeader(http.StatusAccepted)
	_, _ = w.Write([]byte("adf scan accepted\n"))
}

func (scannerHTTPServer *ScannerHTTPServer) authMiddleware(nextHandler http.Handler) http.Handler {
	expectedAuthorizationHeader := "Bearer " + scannerHTTPServer.authToken

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authorizationHeader := r.Header.Get("Authorization")
		if authorizationHeader != expectedAuthorizationHeader {
			http.Error(w, "unauthorized\n", http.StatusUnauthorized)
			return
		}

		nextHandler.ServeHTTP(w, r)
	})
}

func loggingMiddleware(nextHandler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestStartTime := time.Now()
		nextHandler.ServeHTTP(w, r)
		log.Printf(
			"%s %s from=%s duration=%s",
			r.Method,
			r.URL.Path,
			r.RemoteAddr,
			time.Since(requestStartTime),
		)
	})
}

func writeMethodNotAllowed(w http.ResponseWriter, allowedMethod string) {
	w.Header().Set("Allow", allowedMethod)
	http.Error(w, "method not allowed\n", http.StatusMethodNotAllowed)
}
