package main

import (
	"time"
)

type Song struct {
	ID     int        `json:"id"`
	Time   *time.Time `json:"time"`
	Artist string     `json:"artist"`
	Title  string     `json:"title"`
}
