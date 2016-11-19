package main

import "encoding/json"
import "fmt"
import "github.com/boltdb/bolt"

func Latest(tx *bolt.Tx) *Post {
	b := tx.Bucket([]byte("posts"))

	var cur *Post
	b.ForEach(func(k, v []byte) error {
		var p *Post
		err := json.Unmarshal(v, p)
		if err != nil {
			return err
		}
		fmt.Println(p.num)
		cur = compare(cur, p)
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
	if cur.num > p.num {
		return cur
	}
	return p
}
