package main

import "encoding/binary"
import "encoding/json"
import "io/ioutil"
import "fmt"
import "github.com/dgrijalva/jwt-go"
import "github.com/gin-contrib/multitemplate"
import "github.com/gin-gonic/gin"
import "github.com/google/uuid"
import "html/template"
import "net/http"
import "net/smtp"
import "net/url"
import "os"
import "path/filepath"
import "strconv"
import "strings"
import "gopkg.in/gcfg.v1"
import "log"
import "time"

const numPathParam = "num"
const apiRoute = "/api"
const loginPath = "/admin/login"
const contextKeyUser = "user"

type Location struct {
	Domain   string
	Protocol string
}

type Environment struct {
	Develop        bool
	StaticDataDir  string
	StaticAssetDir string
	TemplateDir    string
}

type SmtpConf struct {
	Address  string
	Port     int
	Email    string
	Password string
}

type Database struct {
	User string
	Pass string
	Url  string
	Name string
}

type Publication struct {
	Name      string
	Publisher string
}

type Cfg struct {
	Database    Database
	Location    Location
	Environment Environment
	Smtp        SmtpConf
	Publication Publication
}

type OddClaims struct {
	User string `json:"user"`
	jwt.StandardClaims
}

var location Location
var smtpConf SmtpConf
var jwtKey = []byte(uuid.New().String())

func main() {
	cfg := Cfg{}

	err := gcfg.ReadFileInto(&cfg, "./odd.cfg")

	if err != nil {
		log.Fatalf("Failed to parse gcfg data %s", err)
	}

	d := PgDatasource(cfg.Database.User, cfg.Database.Name, cfg.Environment.Develop)
	location = cfg.Location
	smtpConf = cfg.Smtp

	defaultUser(d)

	r := gin.Default()
	r.HTMLRender = makeMultiRenderer(cfg)

	absAssets, err := filepath.Abs(cfg.Environment.StaticAssetDir)
	if err != nil {
		panic(err)
	}

	err = os.MkdirAll(absAssets, os.ModePerm)
	if err != nil {
		panic(err)
	}
	r.Static("/assets", absAssets)

	absData, err := filepath.Abs(cfg.Environment.StaticDataDir)
	if err != nil {
		panic(err)
	}

	err = os.MkdirAll(absData, os.ModePerm)
	if err != nil {
		panic(err)
	}
	r.Static("/data", absData)

	r.GET("/", latestToon(d))
	r.GET("/post/:"+numPathParam, renderById(d))
	r.GET("/random", randomToon(d))
	r.GET("/archive", renderArchive(d))

	r.GET(loginPath, renderLogin())
	r.GET("/admin/forgot", renderForgot())
	r.GET("/admin/reset", renderReset())

	admin := r.Group("/admin")
	admin.Use(jwtFilter(d))
	{
		admin.GET("/", renderUpload())
		admin.GET("/archive", renderAdminArchive(d))
		admin.GET("/post/:"+numPathParam, renderEditPost(d))
		admin.GET("/users", renderUsers(d))
		admin.GET("/user/", renderNewUser())
		admin.GET("/user/:"+numPathParam, renderEditUser(d))
	}

	r.POST("/api/token", getTokenHandler(d))
	r.POST("/api/forgot", handleForgot(d))
	r.POST("/api/reset", handleReset(d))
	api := r.Group(apiRoute)
	api.Use(jwtFilter(d))
	{
		post := api.Group("/post")
		{
			post.POST("/", handleNewPost(d))
			post.DELETE("/:"+numPathParam, handleDeletePost(d))
			post.POST("/:"+numPathParam, handleUpdatePost(d))
			post.POST("/:"+numPathParam+"/restore", handleRestorePost(d))
		}

		user := api.Group("/user")
		{
			user.DELETE("/:"+numPathParam, handleDeleteUser(d))
			user.POST("/:"+numPathParam, handleEditUser(d))
			user.POST("/", handleNewUser(d))
			user.POST("/:"+numPathParam+"/restore", handleRestoreUser(d))
		}
	}

	r.NoRoute(noRoute())
	if cfg.Environment.Develop {
		r.Run()
	} else {
		//TODO:run on cfg.Environment.Port
		r.Run(":80")
	}
}

func noRoute() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		if strings.HasPrefix(path, apiRoute) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "unknown_route",
			})
		} else {
			c.HTML(http.StatusNotFound, "error.tmpl", gin.H{})
		}
	}
}

func defaultUser(d Datasource) {
	user, err := d.Fetch(1)

	if err != nil {
		log.Printf("Default user missing, creating")
		user = &User{}
		user.Name = "Default"
		user.Email = ""
		user.Deleted = false
		err = d.Create(user)
		if err != nil {
			log.Print("Could not create default user.")
			panic(err)
		}
		user, err = d.Fetch(1)
		if err != nil {
			log.Print("Could not retrieve default user after creation.")
			panic(err)
		}
	}

	if !(*user).Deleted {
		password := uuid.New().String()
		log.Printf("Default user detected and not deleted, changing password to: %s", password)
		d.ChangePassword(user, password)
	}
}

func makeMultiRenderer(cfg Cfg) multitemplate.Render {
	r := multitemplate.New()
	adminTemplatesDir := "admin_templates"
	templatesDir := cfg.Environment.TemplateDir
	name := cfg.Publication.Name
	publisher := cfg.Publication.Publisher

	adminAbs, err := filepath.Abs(adminTemplatesDir)
	if err != nil {
		panic(err.Error())
	}

	abs, err := filepath.Abs(templatesDir)
	if err != nil {
		panic(err.Error())
	}

	var params map[string]string
	paramJson := abs + "/params.json"
	_, err = os.Stat(paramJson)

	if err == nil {
		raw, err := ioutil.ReadFile(abs + "/params.json")
		if err != nil {
			panic(err.Error())
		}
		json.Unmarshal(raw, &params)
	}

	param := func(k string) string {
		return params[k]
	}

	prettyTime := func(t time.Time) string {
		return t.Format("Monday 2 Jan 2006 15:04")
	}

	prettyDate := func(t time.Time) string {
		return t.Format("Monday 2 Jan 2006")
	}

	scheduled := func(t time.Time) bool {
		return t.After(time.Now())
	}

	unixTime := func(t time.Time) int64 {
		return t.Unix() * 1000
	}

	canonical := func(p Post) string {
		return location.Protocol + location.Domain + "/post/" + strconv.Itoa(p.Num)
	}

	image := func(p Post) string {
		return location.Protocol + location.Domain + "/data/" + p.Image
	}

	facebook := func(p Post) string {
		v := make(url.Values)
		v.Add("u", canonical(p))
		return "https://www.facebook.com/sharer/sharer.php" + v.Encode()
	}

	twitter := func(p Post) string {
		v := make(url.Values)
		v.Add("text", "Check out "+p.Title+" from "+name+"!")
		v.Add("url", canonical(p))
		return "https://twitter.com/intent/tweet" + v.Encode()
	}

	reddit := func(p Post) string {
		v := make(url.Values)
		v.Add("url", canonical(p))
		v.Add("title", p.Title)
		return "https://www.reddit.com/submit" + v.Encode()
	}

	tumblr := func(p Post) string {
		v := make(url.Values)
		v.Add("canonicalUrl", canonical(p))
		v.Add("posttype", "photo")
		v.Add("content", image(p))
		return "https://www.tumblr.com/widgets/share/tool" + v.Encode()
	}

	funcs := template.FuncMap{
		"params":     param,
		"prettyTime": prettyTime,
		"prettyDate": prettyDate,
		"scheduled":  scheduled,
		"unixTime":   unixTime,
		"url":        canonical,
		"facebook":   facebook,
		"twitter":    twitter,
		"reddit":     reddit,
		"tumblr":     tumblr,
		"image":      image,
		"name":       func() string { return name },
		"publisher":  func() string { return publisher },
	}
	compileLayouts(r, adminAbs, funcs)
	compileLayouts(r, abs, funcs)
	return r
}

func compileLayouts(r multitemplate.Render, abs string, funcs template.FuncMap) {
	layouts, err := filepath.Glob(abs + "/layouts/*")
	if err != nil {
		panic(err.Error())
	}
	for _, layout := range layouts {
		implements, err := filepath.Glob(layout + "/*.tmpl")
		if err != nil {
			panic(err.Error())
		}
		for _, implement := range implements {
			templateName := abs + "/includes/" + filepath.Base(layout) + ".tmpl"
			files := []string{templateName, implement}
			t := template.Must(template.New(filepath.Base(layout) + ".tmpl").Funcs(funcs).ParseFiles(files...))
			r.Add(filepath.Base(implement), t)
			log.Printf("Added %s with basefile %s", implement, filepath.Base(layout)+".tmpl")
		}
	}
}

func renderToon(p *Post, d Datasource) gin.HandlerFunc {
	if p == nil {
		return func(c *gin.Context) {
			c.HTML(http.StatusNotFound, "error.tmpl", gin.H{})
		}
	}
	prev, next := d.PrevNext(p)
	return func(c *gin.Context) {
		content := gin.H{
			"post": p,
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
		if p != nil {
			c.Redirect(http.StatusFound, "/post/"+strconv.Itoa(p.Num))
		} else {
			noRoute()(c)
		}
	}
}

func latestToon(d Datasource) gin.HandlerFunc {
	return func(c *gin.Context) {
		p := d.Latest()
		if p != nil {
			c.Redirect(http.StatusFound, "/post/"+strconv.Itoa(p.Num))
		} else {
			noRoute()(c)
		}
	}
}

func renderById(d Datasource) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param(numPathParam)
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
	idStr := c.Param(numPathParam)
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return nil, err
	} else {
		return (d.Get(id, admin)), nil
	}

}

func extractIdFromContext(c *gin.Context) (int, error) {
	return strconv.Atoi(c.Param(numPathParam))
}

func renderArchive(d Datasource) gin.HandlerFunc {
	return func(c *gin.Context) {
		p := d.Archive(false)
		c.HTML(http.StatusOK, "archive.tmpl", gin.H{
			"posts": &p,
		})
	}
}

func renderLogin() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.HTML(http.StatusOK, "login.tmpl", gin.H{})
	}
}

func renderForgot() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.HTML(http.StatusOK, "forgot.tmpl", gin.H{})
	}
}

func renderReset() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.HTML(http.StatusOK, "reset.tmpl", gin.H{})
	}
}

func renderUpload() gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _ := c.Get(contextKeyUser)
		c.HTML(http.StatusOK, "upload.tmpl", gin.H{
			"User": gin.H{"Name": user},
		})
	}
}

func renderEditPost(d Datasource) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _ := c.Get(contextKeyUser)
		p, err := extractPostFromContext(d, c, true)
		if err != nil {
			c.HTML(http.StatusNotFound, "error.tmpl", gin.H{})
			return
		}
		c.HTML(http.StatusOK, "upload.tmpl", gin.H{
			"User": gin.H{"Name": user},
			"Post": &p,
		})
	}
}

func renderAdminArchive(d Datasource) gin.HandlerFunc {
	return func(c *gin.Context) {
		p := d.Archive(true)
		user, _ := c.Get(contextKeyUser)
		c.HTML(http.StatusOK, "admin_archive.tmpl", gin.H{
			"posts": &p,
			"User":  gin.H{"Name": user},
		})
	}
}

func renderUsers(d Datasource) gin.HandlerFunc {
	return func(c *gin.Context) {
		u := d.List()
		user, _ := c.Get(contextKeyUser)
		c.HTML(http.StatusOK, "users.tmpl", gin.H{
			"users": &u,
			"User":  gin.H{"Name": user},
		})
	}
}

func renderEditUser(d Datasource) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _ := c.Get(contextKeyUser)
		idStr := c.Param(numPathParam)
		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.HTML(http.StatusNotFound, "error.tmpl", gin.H{})
			return
		}
		eUser, err := d.Fetch(id)
		if err != nil {
			c.HTML(http.StatusNotFound, "error.tmpl", gin.H{})
			return
		}

		c.HTML(http.StatusOK, "user.tmpl", gin.H{
			"eUser": eUser,
			"User":  gin.H{"Name": user},
		})

	}
}

func renderNewUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		//TODO de-duplicate this into a function which takes Context and H and returns an H
		user, _ := c.Get(contextKeyUser)
		c.HTML(http.StatusOK, "user.tmpl", gin.H{
			"User": gin.H{"Name": user},
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

		field, exists := c.GetPostForm("post-image-id")
		if exists {
			post.Image = field
			var num string
			var deleted string
			if !exField("post-num", &num) ||

				!exField("post-deleted", &deleted) {
				return
			}

			post.Num, err = strconv.Atoi(num)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error":      "conversion_error",
					"field_name": "post-num",
				})
				return
			}

			post.Deleted, err = strconv.ParseBool(deleted)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error":      "conversion_error",
					"field_name": "post-deleted",
				})
				return

			}
		} else {
			post.Deleted = false
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

func handleDeleteUser(d Datasource) gin.HandlerFunc {
	//FIXME:don't allow deletion of last user
	//FIXME:side effects of mutation of logged in user (fix with id in context and look up from id -> name cache which is invalidated for id/globally by mutation functions, should probably live inside the DB
	del := func(u *User) error {
		u.Deleted = true
		return d.Update(u)
	}
	return doSomethingWithAUser(d, del, "could_not_delete_user")
}

func handleEditUser(d Datasource) gin.HandlerFunc {
	return func(c *gin.Context) {
		u := User{}
		c.BindJSON(&u)
		err := d.Update(&u)
		if err != nil {
			log.Print(err)
			c.JSON(http.StatusBadRequest, gin.H{})
		} else {
			c.JSON(http.StatusOK, gin.H{})
		}
	}
}

func handleNewUser(d Datasource) gin.HandlerFunc {
	return func(c *gin.Context) {
		u := User{}
		c.BindJSON(&u)
		err := d.Create(&u)
		if err != nil {
			log.Print(err)
			c.JSON(http.StatusBadRequest, gin.H{})
		} else {
			c.JSON(http.StatusOK, gin.H{})
		}
	}
}

func handleRestoreUser(d Datasource) gin.HandlerFunc {
	restore := func(u *User) error {
		u.Deleted = false
		return d.Update(u)
	}
	return doSomethingWithAUser(d, restore, "could_not_restore_user")
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

func doSomethingWithAUser(d Datasource, something func(*User) error, errorMessage string) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := extractIdFromContext(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "could_not_parse_id",
			})
			return
		}
		var u *User
		u, err = d.Fetch(id)
		if err != nil {
			log.Print(err) //should I log these like this?
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "could_not_retrieve_user",
			})
			return
		}

		err = something(u)
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

func doSomethingWithAPost(d Datasource, something func(*Post) error, errorMessage string) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := extractIdFromContext(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "could_not_parse_id",
			})
			return
		}
		var p *Post
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

func getTokenHandler(d Datasource) gin.HandlerFunc {
	type Credentials struct {
		User     string `json:"user" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	return func(c *gin.Context) {
		creds := Credentials{}
		c.BindJSON(&creds)
		name := creds.User
		pass := creds.Password
		user, err := d.Login(name, pass)

		if err != nil {
			log.Print(err)
			c.JSON(http.StatusForbidden, gin.H{
				"error": "not_logged_in",
			})
			return
		}

		tokenString := makeToken(user.Name)

		c.JSON(http.StatusOK, gin.H{
			"jwt": tokenString,
		})

	}
}

func makeToken(username string) string {
	claims := OddClaims{
		username,
		jwt.StandardClaims{
			ExpiresAt: time.Now().Add(time.Minute * 30).Unix(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString(jwtKey)

	return tokenString
}

func handleForgot(d Datasource) gin.HandlerFunc {
	type PasswordRetrieval struct {
		User string `json:"user" binding:"required"`
	}

	return func(c *gin.Context) {
		retrieval := PasswordRetrieval{}
		c.BindJSON(&retrieval)
		u, err := d.FetchByName(retrieval.User)

		if err != nil {
			log.Print(err)
		} else if !u.Deleted {
			token, err := d.ResetPassword(u)

			if err != nil {
				log.Print(err)
			} else {
				err := mailToken(*token, u.Email)
				if err != nil {
					log.Print(err)
				}
			}
		}

		/*
		 * Always say OK, to prevent user enumeration
		 * if this changes to an email address then we
		 * can indicate more about db errors and just
		 * always tell them to check their email.
		 */

		c.JSON(http.StatusOK, gin.H{})
	}
}

func handleReset(d Datasource) gin.HandlerFunc {
	type Reset struct {
		User     string `json:"user" binding:"required"`
		Password string `json:"password" binding:"required"`
		Token    string `json:"token" binding:"required"`
	}

	return func(c *gin.Context) {
		reset := Reset{}
		c.BindJSON(&reset)
		user, err := d.FetchByName(reset.User)
		if err != nil {
			log.Print(err)
			c.JSON(http.StatusBadRequest, gin.H{})
			return
		}

		err = d.ChangePasswordWithToken(user, reset.Password, reset.Token)

		if err != nil {
			log.Print(err)
			c.JSON(http.StatusBadRequest, gin.H{})
			return
		}

		tokenString := makeToken(user.Name)

		c.JSON(http.StatusOK, gin.H{
			"jwt": tokenString,
		})

	}
}

func mailToken(token string, email string) error {
	//TODO: debounce to avoid spam
	auth := smtp.PlainAuth("", smtpConf.Email, smtpConf.Password, smtpConf.Address)
	to := []string{email}
	msg := []byte("To: " + email + "\r\n" +
		"Subject: Forgotten Password\r\n" +
		"From: " + smtpConf.Email + "\r\n" +
		"\r\n" +
		"Please reset your password here: " + location.Protocol + location.Domain + "/admin/reset/\r\n" +
		"With token: " + token)
	err := smtp.SendMail(smtpConf.Address+":"+strconv.Itoa(smtpConf.Port), auth, smtpConf.Email, to, msg)
	return err
}

func jwtFilter(d Datasource) gin.HandlerFunc {
	return func(c *gin.Context) {
		authorization, err := c.Cookie("jwt")

		if err != nil {
			unauthorized(c)
			return
		}

		token, err := jwt.Parse(authorization, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
			}

			return jwtKey, nil
		})

		if err != nil {
			unauthorized(c)
			return
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			c.Set(contextKeyUser, claims["user"])
			c.Next()
		} else {
			fmt.Println(err)
			unauthorized(c)
			return
		}
	}
}

func unauthorized(c *gin.Context) {
	path := c.Request.URL.Path
	if strings.HasPrefix(path, apiRoute) {
		c.JSON(http.StatusUnauthorized, gin.H{
			"Error": gin.H{
				"Code":    401,
				"Message": "Unauthorized Access Rejected",
			},
		})
	} else {
		c.Redirect(http.StatusTemporaryRedirect, loginPath)
	}
}

// itob returns an 8-byte big endian representation of v.
func itob(v int) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(v))
	return b
}
