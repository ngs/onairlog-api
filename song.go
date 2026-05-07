package main

import "time"

// Song is the canonical entry for a track and aggregates how often /
// when it has aired.
type Song struct {
	ID         string     `firestore:"-" json:"id"`
	Title      string     `firestore:"title" json:"title"`
	Artist     string     `firestore:"artist" json:"artist"`
	FirstAired *time.Time `firestore:"firstAired" json:"firstAired"`
	LastAired  *time.Time `firestore:"lastAired" json:"lastAired"`
	PlayCount  int        `firestore:"playCount" json:"playCount"`
}

// Play is one airplay event referencing a Song.
type Play struct {
	ID        string     `firestore:"-" json:"id"`
	SongID    string     `firestore:"songId" json:"songId"`
	Time      *time.Time `firestore:"time" json:"time"`
	RawTitle  string     `firestore:"rawTitle" json:"rawTitle"`
	RawArtist string     `firestore:"rawArtist" json:"rawArtist"`
}
