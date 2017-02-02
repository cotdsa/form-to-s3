package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"log"
	"net/http"
	"os"
	"time"
)

// Size constants
const (
	MB = 1 << 20
)

type Sizer interface {
	Size() int64
}

func healthzHandler(w http.ResponseWriter, r *http.Request) {
	return
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	bucket := os.Getenv("BUCKET")
	if bucket == "" {
		log.Printf("Missing BUCKET environment variable")
		os.Exit(1)
	}

	region := os.Getenv("REGION")
	if region == "" {
		region = "us-east-1"
	}

	s3_path := os.Getenv("S3_PATH")
	if s3_path == "" {
		s3_path = "upload/"
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "No file in request", 400)
		return
	}
	defer file.Close()

	now := time.Now()
	epoch := now.Unix()

	outfile := fmt.Sprintf("%s%d__%s", s3_path, epoch, header.Filename)

	fileHeader := make([]byte, 512)

	// Copy the headers into the fileHeader buffer
	_, err = file.Read(fileHeader)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	fileType := http.DetectContentType(fileHeader)
	fileSize := file.(Sizer).Size()

	sess, err := session.NewSession()
	if err != nil {
		fmt.Println("failed to create AWS session,", err)
		http.Error(w, err.Error(), 400)
		return
	}

	cfg := aws.NewConfig().WithRegion(region)
	svc := s3.New(sess, cfg)

	params := &s3.PutObjectInput{
		Bucket:        aws.String(bucket),
		Key:           aws.String(outfile),
		Body:          file,
		ContentLength: aws.Int64(fileSize),
		ContentType:   aws.String(fileType),
	}

	_, err = svc.PutObject(params)
	if err != nil {
		fmt.Println("failed to upload to s3,", err)
		http.Error(w, err.Error(), 400)
		return
	}

	fmt.Fprintf(w, "File uploaded successfully : ")
	fmt.Fprintf(w, header.Filename)
	return
}

func Log(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s %s", r.RemoteAddr, r.Method, r.URL)
		handler.ServeHTTP(w, r)
	})
}

func main() {
	bucket := os.Getenv("BUCKET")
	if bucket == "" {
		log.Printf("Missing BUCKET environment variable")
		os.Exit(1)
	}

	listen := os.Getenv("LISTEN")
	if listen == "" {
		listen = ":8080"
	}

	handler := os.Getenv("HANDLER")
	if handler == "" {
		handler = "/upload"
	}

	http.HandleFunc("/healthz", healthzHandler)
	http.HandleFunc(handler, uploadHandler)
	http.ListenAndServe(listen, Log(http.DefaultServeMux))
}
