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
	Num      int    `json:"num"`
	Name     string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"-"`
	Deleted  bool   `json:"deleted"`
}
