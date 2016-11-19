package main

import "encoding/json"
import "encoding/binary"
import "github.com/boltdb/bolt"
import "github.com/gin-gonic/gin"
import "net/http"

func main() {
	db, err := bolt.Open("owls.db", 0600, nil)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	err = db.Update(func(tx *bolt.Tx) error {
		var err error
		b := tx.Bucket([]byte("posts"))
		if err != nil {
			panic(err)
		}
		j, _ := json.Marshal(&Post{num: 1, title: "To-Do List", alt: "They really really needed to buy that wrench :(", image: "goat_toon.jpg"})
		id, _ := b.NextSequence()
		return b.Put(itob(int(id)), j)
	})

	if err != nil {
		panic(err)
	}

	r := gin.Default()
	r.LoadHTMLGlob("templates/*")
	r.Static("/assets", "static/assets")
	r.Static("/data", "static/data")
	r.GET("/", func(c *gin.Context) {
		p := &Post{}
		db.View(func(tx *bolt.Tx) error {
			p = Latest(tx)
			return nil
		})
		c.HTML(http.StatusOK, "toon.tmpl", gin.H{
			"title": &p.title,
			"image": &p.image,
			"alt":   &p.alt,
			"num":   &p.num,
		})
	})
	r.Run()
}

// itob returns an 8-byte big endian representation of v.
func itob(v int) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(v))
	return b
}
