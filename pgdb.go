package main

import "database/sql"
import "fmt"
import "log"
import "time"
import "errors"
import _ "github.com/lib/pq"
import "crypto/sha256"
import "encoding/base64"
import "github.com/google/uuid"

type pgDatasource struct {
	db    *sql.DB
	debug bool
}

func PgDatasource(user string, name string, debug bool) Datasource {
	db, err := sql.Open("postgres", fmt.Sprintf("user=%s dbname=%s sslmode=disable", user, name))

	if err != nil {
		log.Fatal("Error: The data source arguments are not valid")
	}

	err = db.Ping()

	if err != nil {
		log.Fatal("Error: Could not establish a connection with the database")
	}

	return pgDatasource{db, debug}
}

func (d pgDatasource) Latest() *Post {
	var p Post
	err := d.db.QueryRow("SELECT * FROM posts WHERE NOT deleted AND posted <= current_timestamp ORDER BY posted DESC, num DESC").Scan(&p.Num, &p.Title, &p.Alt, &p.Image, &p.Posted, &p.Deleted)

	if err != nil {
		return nil
	}

	return &p
}

func (d pgDatasource) Random() *Post {
	var p Post
	err := d.db.QueryRow("SELECT * FROM posts WHERE NOT deleted AND posted <= current_timestamp ORDER BY random() ASC").Scan(&p.Num, &p.Title, &p.Alt, &p.Image, &p.Posted, &p.Deleted)

	if err != nil {
		return nil
	}
	return &p
}

func (d pgDatasource) Archive(admin bool) *[]Post {
	adminQuery := "SELECT * FROM posts ORDER BY posted DESC, num DESC"
	userQuery := "SELECT * FROM posts WHERE NOT deleted AND posted <= current_timestamp ORDER BY posted ASC, num ASC"
	var query string
	if admin {
		query = adminQuery
	} else {
		query = userQuery
	}
	rows, err := d.db.Query(query)

	if err != nil {
		return nil
	}

	defer rows.Close()

	var archive = make([]Post, 0)
	for rows.Next() {
		var p Post
		rows.Scan(&p.Num, &p.Title, &p.Alt, &p.Image, &p.Posted, &p.Deleted)
		archive = append(archive, p)
	}

	return &archive
}

func (d pgDatasource) Get(num int, admin bool) *Post {
	var p Post
	var query string
	if admin {
		query = "SELECT * FROM posts WHERE num = %d"
	} else {
		query = "SELECT * FROM posts WHERE num = %d AND NOT deleted AND posted <= current_timestamp"
	}
	err := d.db.QueryRow(fmt.Sprintf(query, num)).Scan(&p.Num, &p.Title, &p.Alt, &p.Image, &p.Posted, &p.Deleted)
	if err != nil {
		return nil
	}
	return &p
}

func (d pgDatasource) Store(p *Post) error {
	var err error
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	if p.Num != 0 {
		//UPDATE
		_, err = tx.Exec("UPDATE posts SET title = $2, alt = $3, image = $4, posted = $5, deleted = $6 where num = $1", p.Num, p.Title, p.Alt, p.Image, p.Posted, p.Deleted)
	} else {
		//CREATE
		_, err = tx.Exec("INSERT INTO posts(title, alt, image, posted, deleted) values($1, $2, $3, $4, $5)", p.Title, p.Alt, p.Image, p.Posted, p.Deleted)
	}
	return tx.Commit()
}

func (d pgDatasource) Delete(p *Post) error {
	p.Deleted = true
	return d.Store(p)
}

func (d pgDatasource) Restore(p *Post) error {
	p.Deleted = false
	return d.Store(p)
}

func (d pgDatasource) PrevNext(p *Post) (*int, *int) {
	var x int
	var y int
	var prev *int
	var next *int
	tx, err := d.db.Begin()
	if err != nil {
		return nil, nil
	}
	err = tx.QueryRow("SELECT num FROM posts WHERE NOT deleted AND posted <= current_timestamp AND ((posted = $2 AND num < $1) OR posted < $2) ORDER BY posted DESC, num DESC", &p.Num, &p.Posted).Scan(&x)
	if err != nil {
		log.Print(err)
		prev = nil
	} else {
		prev = &x
	}

	err = tx.QueryRow("SELECT num FROM posts WHERE NOT deleted AND posted <= current_timestamp AND ((posted = $2 AND num > $1) OR posted > $2) ORDER BY posted ASC, num ASC", &p.Num, &p.Posted).Scan(&y)
	if err != nil {
		log.Print(err)
		next = nil
	} else {
		next = &y
	}
	tx.Commit()
	return prev, next
}

func (d pgDatasource) Login(username string, password string) (*User, error) {
	tx, err := d.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Commit()
	var salt sql.NullString
	tx.QueryRow("SELECT salt FROM users WHERE NOT deleted AND name = $1", username).Scan(&salt)

	if !salt.Valid {
		log.Print(salt)
		return nil, fmt.Errorf("User %s does not have a salt", username)
	}

	hashedPassword := hash(password, salt.String)

	u := User{}
	u.Name = username
	u.Password = hashedPassword
	u.Deleted = false

	err = tx.QueryRow("SELECT num, email FROM users WHERE NOT deleted AND name=$1 AND password=$2", username, hashedPassword).Scan(&u.Num, &u.Email)

	if err != nil {
		log.Print(err)
		return nil, err
	}
	return &u, nil
}

func (d pgDatasource) Fetch(userId int) (*User, error) {
	var password sql.NullString
	u := User{}
	u.Num = userId
	err := d.db.QueryRow("SELECT name, email, password, deleted FROM users WHERE num=$1", userId).Scan(&u.Name, &u.Email, &password, &u.Deleted)

	if err != nil {
		log.Print(err)
		return nil, err
	}

	if password.Valid {
		u.Password = password.String
	}

	return &u, nil
}

func (d pgDatasource) FetchByName(username string) (*User, error) {
	u := User{}
	u.Name = username
	err := d.db.QueryRow("SELECT num, email, deleted FROM users WHERE name=$1", username).Scan(&u.Num, &u.Email, &u.Deleted)

	if err != nil {
		log.Print(err)
		return nil, err
	}

	return &u, nil
}

func (d pgDatasource) ChangePassword(user *User, newPassword string) error {
	salt := uuid.New().String()
	hashedPassword := hash(newPassword, salt)
	_, err := d.db.Exec("UPDATE users SET password=$2, salt=$3 WHERE num = $1", (*user).Num, hashedPassword, salt)
	return err
}

func (d pgDatasource) ChangePasswordWithToken(user *User, newPassword string, token string) error {
	var salt string
	var num int

	tx, err := d.db.Begin()

	if err != nil {
		return err
	}

	err = tx.QueryRow("SELECT num, salt FROM password_resets ORDER BY num DESC WHERE for_user = $1 AND NOT used AND current_timestamp < not_after", user.Num).Scan(&num, &salt)

	if err != nil {
		tx.Commit()
		return err
	}

	hashedToken := hash(token, salt)

	result, err := tx.Exec("UPDATE password_resets SET used = TRUE WHERE num = $1 AND reset_token = $2", num, hashedToken)
	tx.Commit()

	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()

	if rows != 1 {
		return errors.New("db: invalid token")
	}

	return d.ChangePassword(user, newPassword)
}

func (d pgDatasource) ResetPassword(user *User) (*string, error) {
	salt := uuid.New().String()
	token := uuid.New().String()
	hashedToken := hash(token, salt)
	_, err := d.db.Exec("INSERT INTO password_resets(reset_token, salt, for_user, not_after) VALUES($1, $2, $3, $4)", hashedToken, salt, user.Num, time.Now().Add(time.Hour*12))

	if err != nil {
		return nil, err
	}

	return &token, nil
}

func (d pgDatasource) Create(user *User) error {
	u := *user
	_, err := d.db.Exec("INSERT INTO users(name, email, deleted) values($1, $2, $3)", u.Name, u.Email, u.Deleted)
	return err
}

//FIXME:don't let me delete the last user (or add a switch to undelete Default)
func (d pgDatasource) Update(user *User) error {
	u := *user
	_, err := d.db.Exec("UPDATE users SET name = $2, email = $3, deleted = $4 WHERE num = $1", u.Num, u.Name, u.Email, u.Deleted)
	return err
}

func (d pgDatasource) List() *[]User {
	rows, err := d.db.Query("SELECT num, name, email, deleted FROM users ORDER BY name ASC")

	if err != nil {
		log.Print(err)
		return nil
	}
	defer rows.Close()

	var list = make([]User, 0)
	for rows.Next() {
		var u User
		rows.Scan(&u.Num, &u.Name, &u.Email, &u.Deleted)
		list = append(list, u)
	}
	return &list
}

func hash(password string, salt string) string {
	h := sha256.New()
	h.Write([]byte(password))
	h.Write([]byte(salt))
	return base64.URLEncoding.EncodeToString(h.Sum(nil))
}
