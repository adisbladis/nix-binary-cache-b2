package main // import "github.com/adisbladis/nix-binary-cache-b2"

import (
	"context"
	"fmt"
	"github.com/bakins/logrus-middleware"
	"github.com/kurin/blazer/b2"
	"github.com/sirupsen/logrus"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
)

func main() {

	b2id := os.Getenv("B2_ACCOUNT_ID")
	b2key := os.Getenv("B2_APPLICATION_KEY")
	bucketName := os.Getenv("B2_BUCKET_NAME")

	if b2id == "" {
		panic("Missing B2_ACCOUNT_ID")
	}
	if b2key == "" {
		panic("Missing B2_APPLICATION_KEY")
	}
	if bucketName == "" {
		bucketName = "REDACTED"
	}

	logger := logrus.New()
	logger.Level = logrus.InfoLevel

	ctx := context.Background()
	b2Client, err := b2.NewClient(ctx, b2id, b2key)
	if err != nil {
		panic(err)
	}

	bucket, err := b2Client.Bucket(ctx, bucketName)
	if err != nil {
		panic(err)
	}

	// credentialManager := NewCredentialManager()
	tokenManager := NewTokenManager(&ctx, bucket)

	handler := func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		// Request path relative to the root (excluding leading /)
		rootPath := strings.TrimLeft(r.URL.Path, "/")

		// Serve static root handlers without auth
		switch rootPath {
		case "robots.txt":
			w.Header().Set("Content-Type", "text/plain")
			fmt.Fprintf(w, "User-Agent: *\nDisallow: /")
			return
		case "":
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprintf(w, "<center><h2>This binary cache is provided by REDACTED</h2></center>")
			return
		}

		filePath := strings.Split(rootPath, "/")
		// TODO: Check directory ACL
		directoryPrefix := filePath[0]

		if r.Method == "PUT" {
			obj := bucket.Object(rootPath)
			objWriter := obj.NewWriter(ctx)
			defer objWriter.Close()

			_, err := io.Copy(objWriter, r.Body)
			if err != nil {
				return
			}

		} else if r.Method == "HEAD" || r.Method == "GET" {
			// Path relative to bucket
			relPath := strings.TrimLeft(rootPath, directoryPrefix)

			if relPath == "nix-cache-info" {
				// Returned by cache.nixos.org
				w.Header().Set("Content-Type", "application/octet-stream")
				fmt.Fprintf(w, "StoreDir: /nix/store\n")
				fmt.Fprintf(w, "WantMassQuery: 1\n")
				fmt.Fprintf(w, "Priority: 42\n")
				return
			}

			// Get cached backblaze b2 access token
			token, err := tokenManager.GetToken(directoryPrefix)
			if err != nil {
				logger.Error(err.Error)
				w.WriteHeader(500)
				fmt.Fprintf(w, "Internal server error\n")
				return
			}

			// Assume all other traffic should directly hit B2
			URL, err := url.Parse(bucket.BaseURL())
			if err != nil {
				logger.Error(err.Error)
				w.WriteHeader(500)
				fmt.Fprintf(w, "Internal server error\n")
				return
			}
			URL.Path = path.Join("/", "file", bucketName, directoryPrefix, relPath)
			q := URL.Query()
			q.Set("Authorization", token) // Backblaze B2 auth
			URL.RawQuery = q.Encode()
			urlPath := URL.String()
			http.Redirect(w, r, urlPath, 302)

		} else { // We dont know how to handle other types of requests
			w.WriteHeader(400)
		}
	}

	l := logrusmiddleware.Middleware{
		Name:   "nix-binary-cache-b2",
		Logger: logger,
	}
	http.Handle("/", l.Handler(http.HandlerFunc(handler), "/"))

	logger.Info("Starting")

	log.Fatal(http.ListenAndServe(":8080", nil))

}
