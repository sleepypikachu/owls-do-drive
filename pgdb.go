package main

import "database/sql"
import "fmt"
import "log"
import _ "github.com/lib/pq"

type pgDatasource struct {
	db *sql.DB
}

func PgDatasource(user string, name string) Datasource {
	db, err := sql.Open("postgres", fmt.Sprintf("user=%s dbname=%s sslmode=disable", user, name))

	if err != nil {
		log.Fatal("Error: The data source arguments are not valid")
	}

	err = db.Ping()

	if err != nil {
		log.Fatal("Error: Could not establish a connection with the database")
	}

	return pgDatasource{db}
}

func (d pgDatasource) Latest() *Post {
	var p Post
	err := d.db.QueryRow("SELECT * FROM posts WHERE NOT deleted AND posted <= current_timestamp ORDER BY num DESC").Scan(&p.Num, &p.Title, &p.Alt, &p.Image, &p.Posted, &p.Deleted)

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
	adminQuery := "SELECT * FROM posts ORDER BY num DESC"
	userQuery := "SELECT * FROM posts WHERE NOT deleted AND posted <= current_timestamp ORDER BY num ASC"
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
	//FIXME:transactions!
	var err error
	if p.Num != 0 {
		//UPDATE
		_, err = d.db.Exec("UPDATE posts SET title = $2, alt = $3, image = $4, posted = $5, deleted = $6 where num = $1", p.Num, p.Title, p.Alt, p.Image, p.Posted, p.Deleted)
	} else {
		//CREATE
		_, err = d.db.Exec("INSERT INTO posts(title, alt, image, posted, deleted) values($1, $2, $3, $4, $5)", p.Title, p.Alt, p.Image, p.Posted, p.Deleted)
	}
	return err
}

func (d pgDatasource) Delete(p *Post) error {
	p.Deleted = true
	return d.Store(p)
}

func (d pgDatasource) Restore(p *Post) error {
	p.Deleted = false
	return d.Store(p)
}

func (d pgDatasource) PrevNext(p *Post) (*int, *int, error) {
	rows, err := d.db.Query("SELECT num FROM posts WHERE NOT deleted AND posted <= current_timestamp")
	if err != nil {
		return nil, nil, err
	}
	nums := make([]int, 0)
	defer rows.Close()
	for rows.Next() {
		var i int
		rows.Scan(&i)
		nums = append(nums, i)
	}
	var prev *int
	var next *int
	for _, i := range nums {
		if i < p.Num && (prev == nil || i > *prev) {
			v := int(i)
			prev = &v
		}

		if i > p.Num && (next == nil || i < *next) {
			v := int(i)
			next = &v
		}
	}

	return prev, next, nil
}

func (d pgDatasource) login(username string, password string) (*User, error) {
	return &User{"dummy", "dummy", false}, nil
}

func (d pgDatasource) changePassword(username string, newPassword string) error {
	return nil
}

func (d pgDatasource) create(u User) error {
	return nil
}
