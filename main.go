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
	songsCollection = "songs"
	playsCollection = "plays"
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
	db := os.Getenv("FIRESTORE_DATABASE")
	if db == "" {
		db = firestore.DefaultDatabaseID
	}
	fs, err := firestore.NewClientWithDatabase(ctx, mustGetenv("PROJECT_ID"), db)
	if err != nil {
		log.Fatalf("firestore: %v", err)
	}
	defer fs.Close()

	app := App{FS: fs}
	r := mux.NewRouter()
	r.HandleFunc("/", app.HandleRoot)
	r.HandleFunc("/plays", app.HandlePlays)
	r.HandleFunc("/plays/{id}", app.HandlePlay)
	r.HandleFunc("/songs", app.HandleSongs)
	r.HandleFunc("/songs/{id}", app.HandleSong)
	r.HandleFunc("/songs/{id}/plays", app.HandleSongPlays)
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

// HandlePlays returns the most recent plays. Optional ?since=<RFC3339>
// pages backwards through history.
func (app App) HandlePlays(w http.ResponseWriter, r *http.Request) {
	q := app.FS.Collection(playsCollection).
		OrderBy("time", firestore.Desc).
		Limit(pageLimit)
	if since := r.URL.Query().Get("since"); since != "" {
		t, err := parseSince(since)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		q = q.Where("time", "<", t)
	}
	plays, err := fetchPlays(r.Context(), q)
	if err != nil {
		log.Printf("HandlePlays: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, plays)
}

func (app App) HandlePlay(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	doc, err := app.FS.Collection(playsCollection).Doc(id).Get(r.Context())
	if err != nil {
		if status.Code(err) == codes.NotFound {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		log.Printf("HandlePlay: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var play Play
	if err := doc.DataTo(&play); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	play.ID = doc.Ref.ID
	writeJSON(w, play)
}

// HandleSongs returns canonical songs ordered by most recent airplay.
// Optional ?since=<RFC3339> pages backwards through lastAired.
func (app App) HandleSongs(w http.ResponseWriter, r *http.Request) {
	q := app.FS.Collection(songsCollection).
		OrderBy("lastAired", firestore.Desc).
		Limit(pageLimit)
	if since := r.URL.Query().Get("since"); since != "" {
		t, err := parseSince(since)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		q = q.Where("lastAired", "<", t)
	}
	songs, err := fetchSongs(r.Context(), q)
	if err != nil {
		log.Printf("HandleSongs: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, songs)
}

func (app App) HandleSong(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	doc, err := app.FS.Collection(songsCollection).Doc(id).Get(r.Context())
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

// HandleSongPlays returns the recent plays of a specific song, latest
// first. Optional ?since=<RFC3339> pages backwards.
func (app App) HandleSongPlays(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	q := app.FS.Collection(playsCollection).
		Where("songId", "==", id).
		OrderBy("time", firestore.Desc).
		Limit(pageLimit)
	if since := r.URL.Query().Get("since"); since != "" {
		t, err := parseSince(since)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		q = q.Where("time", "<", t)
	}
	plays, err := fetchPlays(r.Context(), q)
	if err != nil {
		log.Printf("HandleSongPlays: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, plays)
}

// HandleSiri renders a Japanese sentence describing the latest airplay.
func (app App) HandleSiri(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	q := app.FS.Collection(playsCollection).OrderBy("time", firestore.Desc).Limit(1)
	plays, err := fetchPlays(ctx, q)
	if err != nil {
		log.Printf("HandleSiri: fetch plays: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if len(plays) == 0 || plays[0].Time == nil {
		http.Error(w, "no play", http.StatusNotFound)
		return
	}
	play := plays[0]

	title := play.RawTitle
	artist := play.RawArtist
	if play.SongID != "" {
		if doc, err := app.FS.Collection(songsCollection).Doc(play.SongID).Get(ctx); err == nil {
			var song Song
			if err := doc.DataTo(&song); err == nil {
				if song.Title != "" {
					title = song.Title
				}
				if song.Artist != "" {
					artist = song.Artist
				}
			}
		}
	}

	t := *play.Time
	var format string
	if t.Hour() <= 12 {
		format = "午前3時4分"
	} else {
		format = "午後3時4分"
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write([]byte(fmt.Sprintf("%s の %s が %s に放送されました", artist, title, t.Format(format))))
}

func fetchPlays(ctx context.Context, q firestore.Query) ([]Play, error) {
	iter := q.Documents(ctx)
	defer iter.Stop()
	plays := []Play{}
	for {
		doc, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}
		var p Play
		if err := doc.DataTo(&p); err != nil {
			return nil, err
		}
		p.ID = doc.Ref.ID
		plays = append(plays, p)
	}
	return plays, nil
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
