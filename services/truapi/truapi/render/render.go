package render

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// TruError holds data for a TruStory API error
type TruError struct {
	Code    int    `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

// Error implements error
func (e TruError) Error() string {
	return e.Message
}

type jsonResponse struct {
	Status int         `json:"status"`
	Data   interface{} `json:"data,omitempty"`
	Error  string      `json:"error,omitempty"`
}

// JSON renders json payloads
func JSON(w http.ResponseWriter, r *http.Request, v interface{}, code int) {
	buf := &bytes.Buffer{}
	err := json.NewEncoder(buf).Encode(v)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(code)
	_, err = io.Copy(w, buf)
	if err != nil {
		fmt.Println("error sending response to underlying writer", err)
	}
}

// Error renders a json error
func Error(w http.ResponseWriter, r *http.Request, msg string, code int) {
	response := &jsonResponse{
		Error:  msg,
		Status: code,
	}
	JSON(w, r, response, code)
}

// LoginError renders a json login error
func LoginError(w http.ResponseWriter, r *http.Request, err error, statusCode int) {
	response := &jsonResponse{
		Data:   err,
		Error:  err.Error(),
		Status: statusCode,
	}
	JSON(w, r, response, statusCode)
}

// Response renders a json response.
func Response(w http.ResponseWriter, r *http.Request, v interface{}, code int) {
	response := &jsonResponse{
		Data:   v,
		Status: code,
	}
	JSON(w, r, response, code)
}
