package main

import "encoding/binary"
import "github.com/gin-contrib/multitemplate"
import "github.com/gin-gonic/gin"
import "github.com/google/uuid"
import "html/template"
import "net/http"
import "path/filepath"
import "strconv"
import "strings"
import "gopkg.in/gcfg.v1"
import "log"
import "time"

const postPathParam = "num"
const apiRoute = "/api"

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
	r.GET("/", latestToon(d))
	r.GET("/post/:"+postPathParam, renderById(d))
	r.GET("/random", randomToon(d))
	r.GET("/archive", renderArchive(d))

	admin := r.Group("/admin")
	{
		admin.GET("/", renderUpload())
		admin.GET("/archive", renderAdminArchive(d))
		admin.GET("/post/:"+postPathParam, renderEditPost(d))
	}

	api := r.Group(apiRoute)
	{
		api.POST("/post", handleNewPost(d))
		api.DELETE("/post/:"+postPathParam, handleDeletePost(d))
		api.POST("/post/:"+postPathParam, handleUpdatePost(d))
		api.POST("/post/:"+postPathParam+"/restore", handleRestorePost(d))
	}
	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		if strings.HasPrefix(path, apiRoute) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "unknown_route",
			})
		} else {
			c.HTML(http.StatusNotFound, "error.tmpl", gin.H{})
		}
	})
	r.Run()
}

func makeMultiRenderer(templatesDir string) multitemplate.Render {
	r := multitemplate.New()

	layouts, err := filepath.Glob(templatesDir + "layouts/*")
	if err != nil {
		panic(err.Error())
	}

	prettyTime := func(t time.Time) string {
		return t.Format("Monday 2 Jan 2006 15:04")
	}

	scheduled := func(t time.Time) bool {
		return t.After(time.Now())
	}

	unixTime := func(t time.Time) int64 {
		return t.Unix() * 1000
	}

	funcs := template.FuncMap{
		"prettyTime": prettyTime,
		"scheduled":  scheduled,
		"unixTime":   unixTime,
	}

	for _, layout := range layouts {
		implements, err := filepath.Glob(layout + "/*.tmpl")
		if err != nil {
			panic(err.Error())
		}
		for _, implement := range implements {
			templateName := templatesDir + "includes/" + filepath.Base(layout) + ".tmpl"
			files := []string{templateName, implement}
			t := template.Must(template.New(filepath.Base(layout) + ".tmpl").Funcs(funcs).ParseFiles(files...))
			r.Add(filepath.Base(implement), t)
			log.Printf("Added %s with basefile %s", implement, filepath.Base(layout)+".tmpl")
		}
	}
	return r
}

func renderToon(p *Post, d Datasource) gin.HandlerFunc {
	if p == nil {
		return func(c *gin.Context) {
			c.HTML(http.StatusNotFound, "error.tmpl", gin.H{})
		}
	}
	prev, next, err := d.PrevNext(p)
	if err != nil {
		log.Print(err)
		return func(c *gin.Context) {
			c.HTML(http.StatusInternalServerError, "error.tmpl", gin.H{
				"Error": gin.H{
					"Code":    "500",
					"Message": "Database Error, this has been logged. :-(",
				},
			})
		}
	}
	return func(c *gin.Context) {
		content := gin.H{
			"title": &p.Title,
			"image": &p.Image,
			"alt":   &p.Alt,
			"num":   &p.Num,
		}
		if prev != nil {
			content["prev"] = &prev
		}
		if next != nil {
			content["next"] = &next
		}
		c.HTML(http.StatusOK, "toon.tmpl", content)
	}
}

func randomToon(d Datasource) gin.HandlerFunc {
	return func(c *gin.Context) {
		p := d.Random()
		c.Redirect(http.StatusFound, "/post/"+strconv.Itoa(p.Num))
	}
}

func latestToon(d Datasource) gin.HandlerFunc {
	return func(c *gin.Context) {
		p := d.Latest()
		renderToon(p, d)(c)
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
			p = d.Get(id, false)
		}

		renderToon(p, d)(c)
	}
}

func extractPostFromContext(d Datasource, c *gin.Context, admin bool) (*Post, error) {
	idStr := c.Param(postPathParam)
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return nil, err
	} else {
		return (d.Get(id, admin)), nil
	}

}

func renderArchive(d Datasource) gin.HandlerFunc {
	return func(c *gin.Context) {
		p := d.Archive(false)
		c.HTML(http.StatusOK, "archive.tmpl", gin.H{
			"posts": &p,
		})
	}
}

func renderUpload() gin.HandlerFunc {
	return func(c *gin.Context) {
		//TODO: retrieve real user from context somehow...
		//TODO: add the user into the context^
		u := User{"Lauren", "foo", false}
		c.HTML(http.StatusOK, "upload.tmpl", gin.H{
			"User": gin.H{"Name": u.Name},
		})
	}
}

func renderEditPost(d Datasource) gin.HandlerFunc {
	return func(c *gin.Context) {
		u := User{"Lauren", "foo", false}
		p, err := extractPostFromContext(d, c, true)
		if err != nil {
			c.HTML(http.StatusNotFound, "error.tmpl", gin.H{})
			return
		}
		c.HTML(http.StatusOK, "upload.tmpl", gin.H{
			"User": gin.H{"Name": u.Name},
			"Post": &p,
		})
	}
}

func renderAdminArchive(d Datasource) gin.HandlerFunc {
	return func(c *gin.Context) {
		p := d.Archive(true)
		c.HTML(http.StatusOK, "admin_archive.tmpl", gin.H{
			"posts": &p,
			"User":  gin.H{"Name": "Lauren"},
		})
	}
}

func handleNewPost(d Datasource) gin.HandlerFunc {
	return func(c *gin.Context) {
		post := Post{}
		exField := extractFieldFromContext(c)
		var postPublishDate string
		if !exField("post-title", &post.Title) ||
			!exField("post-hover", &post.Alt) ||
			!exField("post-publish-date", &postPublishDate) {
			return
		}

		i, err := strconv.ParseInt(postPublishDate, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":      "conversion_error",
				"field_name": "post-publish-date",
			})
			return
		}

		post.Posted = time.Unix(i, 0)
		post.Deleted = false

		field, exists := c.GetPostForm("post-image-id")
		if exists {
			post.Image = field
		} else {
			file, err := c.FormFile("post-image")
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "extract_image",
				})
				return
			}

			fileUuid := uuid.New().String()
			err = c.SaveUploadedFile(file, "static/data/"+fileUuid)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "could_not_save_file",
				})
				return
			}
			post.Image = fileUuid

		}

		err = d.Store(&post)
		if err != nil {
			log.Print(err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":           "db_err",
				"additional_info": "logged",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{})
	}
}

func handleDeletePost(d Datasource) gin.HandlerFunc {
	return doSomethingWithAPost(d, d.Delete, "could_not_delete_post")
}

func handleRestorePost(d Datasource) gin.HandlerFunc {
	return doSomethingWithAPost(d, d.Restore, "could_not_restore_post")
}

func handleUpdatePost(d Datasource) gin.HandlerFunc {
	return func(c *gin.Context) {
		post := Post{}
		var postNum string
		var postPublishDate string
		var postDeleted string
		exField := extractFieldFromContext(c)
		//FIXME:constants
		if !exField("post-title", &post.Title) ||
			!exField("post-hover", &post.Alt) ||
			!exField("post-publish-date", &postPublishDate) ||
			!exField("post-deleted", &postDeleted) ||
			!exField("post-num", &postNum) {
			return
		}

		i, err := strconv.ParseInt(postPublishDate, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":      "conversion_error",
				"field_name": "post-publish-date",
			})
			return
		}

		post.Num, err = strconv.Atoi(postNum)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":      "conversion_error",
				"field_name": "post-num",
			})
			return
		}

		post.Posted = time.Unix(i, 0)
		post.Deleted, err = strconv.ParseBool(postDeleted)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":      "conversion_error",
				"field_name": "post-deleted",
			})
			return
		}

		err = d.Store(&post)
		if err != nil {
			log.Print(err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":           "db_err",
				"additional_info": "logged",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{})
	}
}

func doSomethingWithAPost(d Datasource, something func(*Post) error, errorMessage string) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param(postPathParam)
		id, err := strconv.Atoi(idStr)
		var p *Post
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "could_not_parse_id",
				"value": idStr,
			})
			return
		}
		p = d.Get(id, true)
		if p == nil {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "could_not_find_post",
				"value": id,
			})
		}
		err = something(p)
		if err != nil {
			log.Print(err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":           errorMessage,
				"additional_info": "logged",
			})
		} else {
			c.JSON(http.StatusOK, gin.H{})
		}
	}
}

func extractFieldFromContext(c *gin.Context) func(from string, to *string) bool {
	return func(from string, to *string) bool {
		field, exists := c.GetPostForm(from)

		if !exists {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":      "missing_field",
				"field_name": from,
			})
		} else {
			*to = field
		}

		return exists
	}
}

// itob returns an 8-byte big endian representation of v.
func itob(v int) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(v))
	return b
}
