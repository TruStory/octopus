package main

import (
	"encoding/json"
	"log"
	"net/http"
	"github.com/gorilla/mux"
	"github.com/TruStory/uploader/models"
	"github.com/aws/aws-sdk-go/aws"
    "github.com/aws/aws-sdk-go/aws/session"
    "github.com/aws/aws-sdk-go/service/s3"
    "github.com/aws/aws-sdk-go/aws/credentials"
    "time"
	)

// Define HTTP request routes
func main() {
	r := mux.NewRouter()
	r.HandleFunc("/upload/aws", getUrl).Methods("POST")
	if err := http.ListenAndServe(":4000", r); err != nil {
		log.Fatal(err)
		}
	}

func getUrl(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var url Url

	// decode URL
	if err := json.NewDecoder(r.Body).Decode(&url); err != nil {
		errorHandler(w, http.StatusBadRequest, err.Error())
		return
		}

	sess, err := session.NewSession(&aws.Config{
        Region: aws.String("us-west-1"), Credentials: credentials.NewStaticCredentials("ADDAWSKEY", "ADDAWSSECRET", "") })

    // Create S3 service client
    svc := s3.New(sess)

    req, _ := svc.PutObjectRequest(&s3.PutObjectInput{ Bucket: aws.String("trustory"), Key: aws.String("images/" + url.Name) })
    
    str, err := req.Presign(15 * time.Minute)
    
    if err != nil {
		errorHandler(w, http.StatusBadRequest, err.Error())
		return
		}

    log.Println("The URL is:", str, " err:", err)

	response(w, http.StatusOK, map[string]interface{}{"url": str})
	}

func errorHandler(w http.ResponseWriter, code int, msg string) {
	response(w, code, map[string]interface{}{"code": code, "message": msg})
	}

func response(w http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
	}


// open a connection to the database
// var dao = TrustoryDAO{}

// connect to db
// func init() {
// 	dao.Host = "localhost"
// 	dao.Db = "trustory"
// 	dao.Connect()
// 	}


