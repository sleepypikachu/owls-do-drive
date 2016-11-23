package main

import "encoding/json"
import "github.com/boltdb/bolt"

func Latest(tx *bolt.Tx) *Post {
	b := tx.Bucket([]byte("posts"))

	cur := &Post{Num: -1, Title: "No Posts Found!", Alt: "No Posts Found!", Image: ""}
	b.ForEach(func(k, v []byte) error {
		p := &Post{}
		err := json.Unmarshal(v, p)
		if err != nil {
			return err
		}
		cur = p
		return nil
	})

	return cur
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
