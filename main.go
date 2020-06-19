package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"

	_ "github.com/jinzhu/gorm/dialects/mysql"
)

func mustGetenv(k string) string {
	v := os.Getenv(k)
	if v == "" {
		log.Panicf("%s environment variable not set.", k)
	}
	return v
}

func main() {
	r := mux.NewRouter()
	db, err := gorm.Open("mysql", mustGetenv("DATABASE_URI"))
	defer db.Close()
	if err != nil {
		log.Fatal(err)
		return
	}
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write([]byte("OK"))
	})
	r.HandleFunc("/siri.txt", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		var song Song
		db.Order("time desc").Last(&song)
		w.Write([]byte(fmt.Sprintf("%s の %s が %s に放送されました", song.Artist, song.Title, (*song.Time).Format("15時4分"))))
	})

	http.Handle("/", r)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
}
