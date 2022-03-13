package storage

import (
	"errors"
	"strconv"
)

type URLRepository interface {
	Find(shortURL string) (string, error)
	Store(url string) (string, error)
}

type item struct {
	fullURL  string
	shortURL string
}

type Repository struct {
	items []item
}

func New() *Repository {
	r := &Repository{}
	return r
}

func (r *Repository) Store(u string) (string, error) {

	for i := 0; i < len(r.items); i++ {
		if r.items[i].fullURL == u {
			return strconv.Itoa(i + 1), nil
		}
	}

	newItem := item{fullURL: u}
	r.items = append(r.items, newItem)
	result := len(r.items)
	return strconv.Itoa(result), nil
}

func (r *Repository) Find(shortURL string) (string, error) {
	id, err := strconv.Atoi(shortURL)
	if err != nil {
		return "", err
	}

	if id < 1 || id > len(r.items) {
		return "", errors.New("not found")
	}

	for i := range r.items {
		if i == id-1 {
			return r.items[i].fullURL, nil
		}
	}

	return "", err
}
