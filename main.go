package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/gorilla/mux"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	collectionSongs = "songs"
	pageLimit       = 20
)

func mustGetenv(k string) string {
	v := os.Getenv(k)
	if v == "" {
		log.Panicf("%s environment variable not set.", k)
	}
	return v
}

type App struct {
	FS *firestore.Client
}

func main() {
	ctx := context.Background()
	fs, err := firestore.NewClient(ctx, mustGetenv("PROJECT_ID"))
	if err != nil {
		log.Fatalf("firestore: %v", err)
	}
	defer fs.Close()

	app := App{FS: fs}
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

func parseSince(s string) (time.Time, error) {
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid since: %q", s)
}

func (app App) HandleSongs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	q := app.FS.Collection(collectionSongs).
		OrderBy("time", firestore.Desc).
		Limit(pageLimit)
	if since := mux.Vars(r)["since"]; since != "" {
		t, err := parseSince(since)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		q = q.Where("time", "<", t)
	}
	songs, err := fetchSongs(ctx, q)
	if err != nil {
		log.Printf("HandleSongs: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, songs)
}

func (app App) HandleSong(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := mux.Vars(r)["id"]
	doc, err := app.FS.Collection(collectionSongs).Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		log.Printf("HandleSong: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var song Song
	if err := doc.DataTo(&song); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	song.ID = doc.Ref.ID
	writeJSON(w, song)
}

func (app App) HandleSiri(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	q := app.FS.Collection(collectionSongs).OrderBy("time", firestore.Desc).Limit(1)
	songs, err := fetchSongs(ctx, q)
	if err != nil {
		log.Printf("HandleSiri: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if len(songs) == 0 || songs[0].Time == nil {
		http.Error(w, "no song", http.StatusNotFound)
		return
	}
	song := songs[0]
	t := *song.Time
	var format string
	if t.Hour() <= 12 {
		format = "午前3時4分"
	} else {
		format = "午後3時4分"
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write([]byte(fmt.Sprintf("%s の %s が %s に放送されました", song.Artist, song.Title, t.Format(format))))
}

func fetchSongs(ctx context.Context, q firestore.Query) ([]Song, error) {
	iter := q.Documents(ctx)
	defer iter.Stop()
	songs := []Song{}
	for {
		doc, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}
		var s Song
		if err := doc.DataTo(&s); err != nil {
			return nil, err
		}
		s.ID = doc.Ref.ID
		songs = append(songs, s)
	}
	return songs, nil
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(v)
}
