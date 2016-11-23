package main

type Post struct {
	Num   int    `json:"num"`
	Title string `json:"title"`
	Alt   string `json:"alt"`
	Image string `json:"image"`
}
