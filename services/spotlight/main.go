package main

import (
	"fmt"
	"html/template"
	"net/http"

	"github.com/itskingori/go-wkhtml/wkhtmltox"
)

// User denotes a person
type User struct {
	AvatarURL string
	FullName  string
}

// Story denotes a story
type Story struct {
	Body    string
	Creator User
}

func main() {
	tmpl := template.Must(template.ParseFiles("story.html"))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		data := Story{
			Body: "Some story body goes here...",
			Creator: User{
				AvatarURL: "https://randomuser.me/api/portraits/men/32.jpg",
				FullName:  "Someone Random",
			},
		}
		tmpl.Execute(w, data)
	})

	http.HandleFunc("/spotlight", func(w http.ResponseWriter, r *http.Request) {
		ifs := make(wkhtmltox.ImageFlagSet)
		ifs.SetFormat("png")
		ifs.SetHeight(1200)
		ifs.SetWidth(630)
		outputLogs, _ := ifs.Generate("http://localhost:8080/", "./storage/image.png")
		fmt.Printf("%s", outputLogs)
	})

	fmt.Println("Starting the web server...")
	http.ListenAndServe(":8080", nil)

}
