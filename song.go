package main

import "time"

type Song struct {
	ID     string     `firestore:"-" json:"id"`
	Time   *time.Time `firestore:"time" json:"time"`
	Artist string     `firestore:"artist" json:"artist"`
	Title  string     `firestore:"title" json:"title"`
}
