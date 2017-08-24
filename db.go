package main

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
	login(username string, password string) (*User, error)
	changePassword(username string, newPassword string) error
	create(User) error
}

type dummyDatasource struct {
	name string
}

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

func (d dummyDatasource) login(username string, password string) (*User, error) {
	return &User{d.name, "foo", false}, nil
}

func (d dummyDatasource) changePassword(username string, newPassword string) error {
	return nil
}

func (d dummyDatasource) create(u User) error {
	d.name = u.Name
	return nil
}

func (d dummyDatasource) PrevNext(p *Post) (*int, *int) {
	return nil, nil
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
