package main

import "errors"

type Datasource interface {
	/* Posts */
	Latest() *Post
	Random() *Post
	Get(num int, admin bool) *Post
	Store(*Post) error
	Delete(*Post) error
	Restore(*Post) error
	Archive(admin bool) *[]Post
	PrevNext(*Post) (*int, *int)

	/* Users */
	Fetch(userId int) (*User, error)
	FetchByName(username string) (*User, error)
	Login(username string, password string) (*User, error)
	ChangePassword(user *User, newPassword string) error
	ResetPassword(user *User) (*string, error)
	ChangePasswordWithToken(user *User, newPassword string, token string) error
	Create(*User) error
	Update(*User) error
	List() *[]User
}

var ErrUniqueConstraint = errors.New("sql: duplicate key value violates unique constraint")

type dummyDatasource struct{}

func (d dummyDatasource) Latest() *Post {
	return &Post{Num: 1, Title: "Sample Post", Alt: "Sample", Image: "goat_toon.jpg"}
}

func (d dummyDatasource) Random() *Post {
	return d.Latest()
}

func (d dummyDatasource) Archive(bool) *[]Post {
	var archive = make([]Post, 1)
	archive[0] = *d.Latest()
	return &archive
}

func (d dummyDatasource) Get(num int, admin bool) *Post {
	return d.Latest()
}

func (d dummyDatasource) Store(*Post) error {
	return nil
}

func (d dummyDatasource) Delete(*Post) error {
	return nil
}

func (d dummyDatasource) Restore(*Post) error {
	return nil
}

func (d dummyDatasource) Login(username string, password string) (*User, error) {
	return &User{1, "dummy", "dummy@dummy.net", "foo", false}, nil
}

func (d dummyDatasource) Fetch(userId int) (*User, error) {
	return d.Login("", "")
}

func (d dummyDatasource) ChangePassword(user *User, newPassword string) error {
	return nil
}

func (d dummyDatasource) FetchByName(username string) (*User, error) {
	return nil, nil
}

func (d dummyDatasource) ResetPassword(user *User) (*string, error) {
	return nil, nil
}

func (d dummyDatasource) ChangePasswordWithToken(user *User, newPassword string, token string) error {
	return nil
}

func (d dummyDatasource) Create(u *User) error {
	return nil
}

func (d dummyDatasource) Update(u *User) error {
	return nil
}

func (d dummyDatasource) PrevNext(p *Post) (*int, *int) {
	return nil, nil
}

func (d dummyDatasource) List() *[]User {
	var list = make([]User, 0)
	return &list
}

func DummyDatasource() Datasource {
	return dummyDatasource{}
}

func compare(cur *Post, p *Post) *Post {
	if cur == nil {
		return p
	}
	if p == nil {
		return cur
	}
	if cur.Num > p.Num {
		return cur
	}
	return p
}
