package storage

import (
	"bufio"
	"encoding/json"
	"errors"
	"go-yandex/internal/app/config"
	"os"
	"strconv"
)

type URLRepository interface {
	Find(shortURL string) (string, error)
	Store(url string) (string, error)
}

type item struct {
	FullURL  string `json:"full_url"`
	ShortURL string `json:"short_url"`
}

type Repository struct {
	items []item
}

type FileRepository struct {
	fileStoragePath string
}

func New(config config.Config) URLRepository {
	var r URLRepository
	if config.FileStoragePath != "" {
		r = &FileRepository{config.FileStoragePath}
	} else {
		r = &Repository{}
	}

	return r
}

func (r *FileRepository) Store(newURL string) (string, error) {
	file, err := os.OpenFile(r.fileStoragePath, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0777)
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	ln := 0

	for scanner.Scan() {
		ln++
		data := scanner.Bytes()

		item := item{}
		err := json.Unmarshal(data, &item)

		if err != nil {
			return "", err
		}

		if item.FullURL == newURL {
			return item.ShortURL, nil
		}
	}

	newItem := item{
		newURL,
		strconv.Itoa(ln + 1),
	}

	writer := bufio.NewWriter(file)
	data, err := json.Marshal(&newItem)

	if err != nil {
		return "", err
	}

	if _, err := writer.Write(data); err != nil {
		return "", err
	}
	if _, err := writer.Write([]byte("\n")); err != nil {
		return "", err
	}
	if err := writer.Flush(); err != nil {
		return "", err
	}

	return newItem.ShortURL, nil
}

func (r *FileRepository) Find(shortURL string) (string, error) {
	file, err := os.OpenFile(r.fileStoragePath, os.O_APPEND|os.O_CREATE|os.O_RDONLY, 0777)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var shortURLInt int
	shortURLInt, err = strconv.Atoi(shortURL)
	if err != nil {
		return "", err
	}

	scanner := bufio.NewScanner(file)
	ln := 0
	for scanner.Scan() {
		ln++
		if ln == shortURLInt {
			data := scanner.Bytes()

			item := item{}
			err := json.Unmarshal(data, &item)
			if err != nil {
				return "", err
			}

			return item.FullURL, nil
		}
	}

	return "", errors.New("not found")
}

func (r *Repository) Store(u string) (string, error) {

	for i := 0; i < len(r.items); i++ {
		if r.items[i].FullURL == u {
			return strconv.Itoa(i + 1), nil
		}
	}

	newItem := item{FullURL: u}
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
			return r.items[i].FullURL, nil
		}
	}

	return "", err
}
