package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

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

type App struct {
	DB *gorm.DB
}

func main() {
	db, err := gorm.Open("mysql", mustGetenv("DATABASE_URI"))
	defer db.Close()
	if err != nil {
		log.Fatal(err)
		return
	}

	app := App{DB: db}
	r := mux.NewRouter()
	r.HandleFunc("/", app.HandleRoot)
	r.HandleFunc("/songs", app.HandleSongs).Queries("since", "{since}")
	r.HandleFunc("/songs", app.HandleSongs)
	r.HandleFunc("/songs/{id}", app.HandleSong)
	r.HandleFunc("/siri", app.HandleSiri)
	http.Handle("/", r)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
}

func (app App) HandleRoot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write([]byte("OK"))
}

func (app App) HandleSongs(w http.ResponseWriter, r *http.Request) {
	since := mux.Vars(r)["since"]
	result := app.DB.Order("time desc").Limit(20)
	if since != "" {
		result = result.Where("time < ?", since)
	}
	var songs []Song
	result.Find(&songs)
	data, _ := json.Marshal(songs)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Write(data)
}

func (app App) HandleSong(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		w.WriteHeader(400)
		w.Write([]byte(fmt.Sprintf("%s", err)))
		return
	}
	var song Song
	app.DB.Order("time desc").First(&song, id)
	data, _ := json.Marshal(song)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Write(data)
}

func (app App) HandleSiri(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	var song Song
	app.DB.Order("time desc").Last(&song)
	w.Write([]byte(fmt.Sprintf("%s の %s が %s に放送されました", song.Artist, song.Title, (*song.Time).Format("15時4分"))))
}
