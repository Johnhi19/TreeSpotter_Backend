package models

import "time"

type Image struct {
	ID          int       `json:"id"`
	Path        string    `json:"path"`
	Description string    `json:"description"`
	Datetime    time.Time `json:"datetime"`
}
