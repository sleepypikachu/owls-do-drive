package main

import "encoding/binary"
import "github.com/gin-contrib/multitemplate"
import "github.com/gin-gonic/gin"
import "html/template"
import "net/http"
import "path/filepath"
import "strconv"
import "gopkg.in/gcfg.v1"
import "log"

const postPathParam = "num"

func main() {
	cfg := struct {
		Database struct {
			User string
			Pass string
			Url  string
			Name string
		}
	}{}

	err := gcfg.ReadFileInto(&cfg, "./odd.cfg")

	if err != nil {
		log.Fatalf("Failed to parse gcfg data %s", err)
	}

	d := PgDatasource(cfg.Database.User, cfg.Database.Name)

	r := gin.Default()
	r.HTMLRender = makeMultiRenderer("./templates/")
	r.Static("/assets", "static/assets")
	r.Static("/data", "static/data")
	r.GET("/", renderToon(d.Latest()))
	r.GET("/post/:"+postPathParam, renderById(d))
	r.GET("/random", randomToon(d))
	r.GET("/archive", renderArchive(d))
	r.NoRoute(func(c *gin.Context) {
		c.HTML(http.StatusNotFound, "404.tmpl", gin.H{})
	})
	r.Run()
}

func makeMultiRenderer(templatesDir string) multitemplate.Render {
	r := multitemplate.New()

	layouts, err := filepath.Glob(templatesDir + "layouts/*.tmpl")
	if err != nil {
		panic(err.Error())
	}

	includes, err := filepath.Glob(templatesDir + "includes/*.tmpl")
	if err != nil {
		panic(err.Error())
	}

	// Generate our templates map from our layouts/ and includes/ directories
	for _, layout := range layouts {
		files := append(includes, layout)
		r.Add(filepath.Base(layout), template.Must(template.ParseFiles(files...)))
	}
	return r
}

func renderToon(p *Post) gin.HandlerFunc {
	if p == nil {
		return func(c *gin.Context) {
			c.HTML(http.StatusNotFound, "404.tmpl", gin.H{})
		}
	}
	return func(c *gin.Context) {
		c.HTML(http.StatusOK, "toon.tmpl", gin.H{
			"title": &p.Title,
			"image": &p.Image,
			"alt":   &p.Alt,
			"num":   &p.Num,
		})
	}
}

func randomToon(d Datasource) gin.HandlerFunc {
	return func(c *gin.Context) {
		p := d.Random()
		renderToon(p)(c)
	}
}

func renderById(d Datasource) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param(postPathParam)
		id, err := strconv.Atoi(idStr)
		var p *Post
		if err != nil {
			p = nil
		} else {
			p = d.Get(id)
		}

		renderToon(p)(c)
	}
}

func renderArchive(d Datasource) gin.HandlerFunc {
	return func(c *gin.Context) {
		p := d.Archive()
		c.HTML(http.StatusOK, "archive.tmpl", gin.H{
			"posts": &p,
		})
	}
}

// itob returns an 8-byte big endian representation of v.
func itob(v int) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(v))
	return b
}
