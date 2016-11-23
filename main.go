package main

import "encoding/binary"
import "fmt"
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
		_, err := tx.CreateBucketIfNotExists([]byte("posts"))
		return err
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
			fmt.Println("there!")
			p = Latest(tx)
			return nil
		})
		c.HTML(http.StatusOK, "toon.tmpl", gin.H{
			"title": &p.Title,
			"image": &p.Image,
			"alt":   &p.Alt,
			"num":   &p.Num,
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
