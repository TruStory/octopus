package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/gorilla/mux"
)

// URL defines the struct for input data from the client
type URL struct {
	Name    string `json:"image_name"`
	Content string `json:"content_type"`
}

// Config defines the configuration TOML file and the variables with it
type Config struct {
	AWSKey      string
	AWSSecret   string
	BucketName  string
	Port        string
	Region      string
	ImageFolder string
}

// Define HTTP request routes
func main() {
	var conf Config
	if _, err := toml.DecodeFile("config.toml", &conf); err != nil {
		log.Fatal(err)
		return
	}

	r := mux.NewRouter()
	r.Use(CORSMiddleware)
	r.HandleFunc("/v1/upload/aws", getURL).Methods("POST", "OPTIONS")
	if err := http.ListenAndServe(":"+conf.Port, r); err != nil {
		log.Fatal(err)
	}
}

func getURL(w http.ResponseWriter, r *http.Request) {
	var url URL
	var conf Config
	if _, err := toml.DecodeFile("config.toml", &conf); err != nil {
		errorHandler(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := json.NewDecoder(r.Body).Decode(&url); err != nil {
		errorHandler(w, http.StatusBadRequest, err.Error())
		return
	}

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(conf.Region), Credentials: credentials.NewStaticCredentials(conf.AWSKey, conf.AWSSecret, "")})

	if err != nil {
		errorHandler(w, http.StatusBadRequest, err.Error())
		return
	}

	svc := s3.New(sess)

	req, _ := svc.PutObjectRequest(&s3.PutObjectInput{Bucket: aws.String(conf.BucketName), Key: aws.String(conf.ImageFolder + url.Name)})

	str, err := req.Presign(15 * time.Minute)

	if err != nil {
		errorHandler(w, http.StatusBadRequest, err.Error())
		return
	}

	response(w, http.StatusOK, map[string]interface{}{"url": str})
}

func errorHandler(w http.ResponseWriter, code int, msg string) {
	response(w, code, map[string]interface{}{"code": code, "message": msg})
}

func response(w http.ResponseWriter, code int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		errorHandler(w, http.StatusBadRequest, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_, err = w.Write(response)
	if err != nil {
		errorHandler(w, http.StatusBadRequest, err.Error())
		return
	}

}

// CORSMiddleware is an HTTP-handling middleware that adds `Access-Control-Allow-Origin: *` to the response.
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
		if r.Method == "OPTIONS" {
			return
		}
		next.ServeHTTP(w, r)
	})
}
