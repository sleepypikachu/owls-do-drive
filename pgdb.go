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
	err := d.db.QueryRow("SELECT * FROM posts WHERE NOT deleted ORDER BY num DESC").Scan(&p.Num, &p.Title, &p.Alt, &p.Image, &p.Deleted)

	if err != nil {
		return nil
	}

	return &p
}

func (d pgDatasource) Random() *Post {
	var p Post
	err := d.db.QueryRow("SELECT * FROM posts WHERE NOT deleted ORDER BY random() ASC").Scan(&p.Num, &p.Title, &p.Alt, &p.Image, &p.Deleted)

	if err != nil {
		return nil
	}
	return &p
}

func (d pgDatasource) Archive() *[]Post {
	rows, err := d.db.Query("SELECT * FROM POSTS WHERE NOT deleted")

	if err != nil {
		return nil
	}

	defer rows.Close()

	var archive = make([]Post, 0)
	for rows.Next() {
		var p Post
		rows.Scan(&p.Num, &p.Title, &p.Alt, &p.Image, &p.Deleted)
		archive = append(archive, p)
	}

	return &archive
}

func (d pgDatasource) Get(num int) *Post {
	var p Post
	err := d.db.QueryRow(fmt.Sprintf("SELECT * FROM posts WHERE num = %d AND NOT deleted", num)).Scan(&p.Num, &p.Title, &p.Alt, &p.Image, &p.Deleted)
	if err != nil {
		return nil
	}
	return &p
}

func (d pgDatasource) Store(p *Post) error {
	_, err := d.db.Exec("INSERT INTO posts(title, alt, image, deleted) values($1, $2, $3, $4)", p.Title, p.Alt, p.Image, p.Deleted)
	return err
}

func (d pgDatasource) Delete(p *Post) error {
	_, err := d.db.Exec(fmt.Sprintf("UPDATE posts SET deleted=true WHERE num = %d", p.Num))
	return err
}

func (d pgDatasource) Restore(p *Post) error {
	_, err := d.db.Exec(fmt.Sprintf("UPDATE posts SET deleted=false WHERE num = %d", p.Num))
	return err
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
