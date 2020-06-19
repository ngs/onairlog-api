package main

import (
	"time"
)

type Song struct {
	Time   *time.Time `json:"time"`
	Artist string     `json:"artist"`
	Title  string     `json:"title"`
}
