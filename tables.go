package main

import "time"

type Post struct {
	Num     int       `json:"num"`
	Title   string    `json:"title"`
	Alt     string    `json:"alt"`
	Image   string    `json:"image"`
	Posted  time.Time `json:"-"`
	Deleted bool      `json:"-"`
}

type User struct {
	Name     string `json:"username"`
	Password string `json:"-"`
	Deleted  bool   `json"-"`
}
