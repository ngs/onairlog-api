package main

import (
	"fmt"
	"strings"
	"time"
)

// Song is the canonical entry for a track and aggregates how often /
// when it has aired.
type Song struct {
	ID         string     `firestore:"-" json:"id"`
	Title      string     `firestore:"title" json:"title"`
	Artist     string     `firestore:"artist" json:"artist"`
	FirstAired *time.Time `firestore:"firstAired" json:"firstAired"`
	LastAired  *time.Time `firestore:"lastAired" json:"lastAired"`
	PlayCount  int        `firestore:"playCount" json:"playCount"`

	// Enrichment fields populated by the Sync function via iTunes
	// Search + Vertex AI Gemini.
	EnrichedAt      *time.Time             `firestore:"enrichedAt,omitempty" json:"enrichedAt,omitempty"`
	ITunesTrackID   int64                  `firestore:"itunesTrackId,omitempty" json:"itunesTrackId,omitempty"`
	CanonicalTitle  string                 `firestore:"canonicalTitle,omitempty" json:"canonicalTitle,omitempty"`
	CanonicalArtist string                 `firestore:"canonicalArtist,omitempty" json:"canonicalArtist,omitempty"`
	CanonicalKey    string                 `firestore:"canonicalKey,omitempty" json:"canonicalKey,omitempty"`
	ITunesResponse  map[string]interface{} `firestore:"itunesResponse,omitempty" json:"-"`

	// Derived for the JSON response.
	DisplayTitle  string `firestore:"-" json:"displayTitle,omitempty"`
	DisplayArtist string `firestore:"-" json:"displayArtist,omitempty"`
	ITunesURL     string `firestore:"-" json:"itunesUrl,omitempty"`
	ArtworkURL    string `firestore:"-" json:"artworkUrl,omitempty"`
}

// hydrate fills the derived display fields after a Song is loaded
// from Firestore.
func (s *Song) hydrate() {
	if s.CanonicalTitle != "" {
		s.DisplayTitle = s.CanonicalTitle
	} else {
		s.DisplayTitle = s.Title
	}
	if s.CanonicalArtist != "" {
		s.DisplayArtist = s.CanonicalArtist
	} else {
		s.DisplayArtist = s.Artist
	}
	if s.ITunesTrackID > 0 {
		s.ITunesURL = fmt.Sprintf("https://music.apple.com/jp/song/%d", s.ITunesTrackID)
	}
	s.ArtworkURL = artworkFromITunes(s.ITunesResponse)
}

func artworkFromITunes(itunes map[string]interface{}) string {
	if itunes == nil {
		return ""
	}
	results, _ := itunes["results"].([]interface{})
	if len(results) == 0 {
		return ""
	}
	r0, _ := results[0].(map[string]interface{})
	u, _ := r0["artworkUrl100"].(string)
	if u == "" {
		return ""
	}
	return strings.ReplaceAll(u, "100x100bb", "600x600bb")
}

// Play is one airplay event referencing a Song.
type Play struct {
	ID        string     `firestore:"-" json:"id"`
	SongID    string     `firestore:"songId" json:"songId"`
	Time      *time.Time `firestore:"time" json:"time"`
	RawTitle  string     `firestore:"rawTitle" json:"rawTitle"`
	RawArtist string     `firestore:"rawArtist" json:"rawArtist"`
}
