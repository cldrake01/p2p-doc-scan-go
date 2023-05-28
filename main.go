package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"cloud.google.com/go/vision/apiv1"
	"golang.org/x/net/context"
	"google.golang.org/api/option"
	_ "google.golang.org/genproto/googleapis/cloud/vision/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func main() {
	http.HandleFunc("/detect_text", detectTextHandler)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		log.Printf("Defaulting to port %s", port)
	}

	log.Printf("Listening on port %s", port)
	log.Printf("Open http://localhost:%s in the browser", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
}

func detectTextHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	if r.Method != "POST" {
		log.Printf("Invalid HTTP method: %s", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Create a custom HTTP client with insecure skip verify
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	httpClient := &http.Client{Transport: tr}

	// Create gRPC options with custom HTTP client
	opts := []option.ClientOption{
		option.WithGRPCDialOption(grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{InsecureSkipVerify: true}))),
		option.WithHTTPClient(httpClient),
	}

	// Create the vision client with options
	client, err := vision.NewImageAnnotatorClient(ctx, opts...)
	if err != nil {
		log.Printf("NewImageAnnotatorClient: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	defer r.Body.Close()
	image, err := vision.NewImageFromReader(r.Body)
	if err != nil {
		log.Printf("NewImageFromReader: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	annotations, err := client.DetectTexts(ctx, image, nil, 2000)
	if err != nil {
		log.Printf("Annotations Error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if len(annotations) == 0 {
		fmt.Fprintln(w, "No text found.")
		w.Write([]byte("No text found."))
	} else {
		texts := make([]string, len(annotations))
		for i, annotation := range annotations {
			texts[i] = annotation.Description
		}

		jsonBytes, err := json.Marshal(texts)
		if err != nil {
			log.Printf("JSON Marshal Error: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "OPTIONS, GET, POST, PUT, PATCH, DELETE")
		w.WriteHeader(http.StatusOK)
		w.Write(jsonBytes)
	}
}
