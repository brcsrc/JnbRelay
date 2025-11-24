package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"mime"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"time"
)

type Config struct {
	Host      string
	Port      int
	ProxyHost string
	ProxyPort int
	CertFile  string
	KeyFile   string
}

func parseFlags() *Config {
	config := &Config{}

	// Define flags
	flag.StringVar(&config.Host, "host", "", "Host address to listen on (required)")
	flag.IntVar(&config.Port, "port", 0, "Port to listen on (required)")
	flag.StringVar(&config.ProxyHost, "proxy-for-host", "", "Host to proxy requests to (required)")
	flag.IntVar(&config.ProxyPort, "proxy-for-port", 0, "Port to proxy requests to (required)")
	flag.StringVar(&config.CertFile, "cert", "", "Path to TLS certificate file (required)")
	flag.StringVar(&config.KeyFile, "key", "", "Path to TLS key file (required)")

	// Custom usage message
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "All flags are required:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExample:\n")
		fmt.Fprintf(os.Stderr, "  %s --host 0.0.0.0 --port 443 --proxy-for-host 127.0.0.1 --proxy-for-port 8443 --cert cert.crt --key key.pem\n", os.Args[0])
	}

	flag.Parse()

	// Verify all required flags are provided
	var missingFlags []string

	if config.Host == "" {
		missingFlags = append(missingFlags, "host")
	}
	if config.Port == 0 {
		missingFlags = append(missingFlags, "port")
	}
	if config.ProxyHost == "" {
		missingFlags = append(missingFlags, "proxy-for-host")
	}
	if config.ProxyPort == 0 {
		missingFlags = append(missingFlags, "proxy-for-port")
	}
	if config.CertFile == "" {
		missingFlags = append(missingFlags, "cert")
	}
	if config.KeyFile == "" {
		missingFlags = append(missingFlags, "key")
	}

	if len(missingFlags) > 0 {
		fmt.Fprintf(os.Stderr, "Error: missing required flags: %v\n\n", missingFlags)
		flag.Usage()
		os.Exit(1)
	}

	return config
}

func main() {
	// Parse command line flags
	config := parseFlags()

	// Verify certificate files exist
	if _, err := os.Stat(config.CertFile); os.IsNotExist(err) {
		log.Fatalf("Certificate file not found: %s", config.CertFile)
	}
	if _, err := os.Stat(config.KeyFile); os.IsNotExist(err) {
		log.Fatalf("Key file not found: %s", config.KeyFile)
	}

	// Construct target URL
	targetURL := fmt.Sprintf("http://%s:%d", config.ProxyHost, config.ProxyPort)
	target, err := url.Parse(targetURL)
	if err != nil {
		log.Fatal(err)
	}

	// Create a reverse proxy
	proxy := httputil.NewSingleHostReverseProxy(target)

	// Customize the director
	proxy.Director = func(req *http.Request) {
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.Host = target.Host

		// Add standard proxy headers
		req.Header.Add("X-Forwarded-Host", req.Host)
		req.Header.Add("X-Forwarded-Proto", req.URL.Scheme)
		req.Header.Add("X-Real-IP", req.RemoteAddr)
	}

	// Fix MIME types based on file extension
	proxy.ModifyResponse = func(resp *http.Response) error {
		// Get the request path
		path := resp.Request.URL.Path

		// Detect MIME type from file extension
		ext := filepath.Ext(path)
		if ext != "" {
			correctMimeType := mime.TypeByExtension(ext)

			// If we detected a MIME type, update the Content-Type header
			if correctMimeType != "" {
				currentContentType := resp.Header.Get("Content-Type")

				// Only override if the current type is wrong or generic
				if currentContentType == "" ||
					currentContentType == "text/plain" ||
					currentContentType == "application/octet-stream" {
					resp.Header.Set("Content-Type", correctMimeType)
					log.Printf("Fixed MIME type for %s: %s -> %s", path, currentContentType, correctMimeType)
				}
			}
		}

		return nil
	}

	// Add error handling
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("Proxy error: %v", err)
		w.WriteHeader(http.StatusBadGateway)
	}

	// Create server with timeouts
	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", config.Host, config.Port),
		Handler:      proxy,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Handle graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt)
		<-sigChan

		log.Println("Shutting down server...")
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}
	}()

	// Start the server
	log.Printf("Starting reverse proxy on %s:%d -> %s:%d",
		config.Host, config.Port, config.ProxyHost, config.ProxyPort)

	if err := server.ListenAndServeTLS(config.CertFile, config.KeyFile); err != http.ErrServerClosed {
		log.Fatal(err)
	}
}