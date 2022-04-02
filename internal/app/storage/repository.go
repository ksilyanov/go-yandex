package storage

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	_ "github.com/jackc/pgx/stdlib"
	"go-yandex/internal/app/config"
	"log"
	"os"
	"strconv"
	"time"
)

type URLRepository interface {
	Find(shortURL string) (string, error)
	Store(url string, userToken string) (string, error)
	GetByUser(token string) ([]ItemURL, error)
	PingDB() bool
}

type item struct {
	FullURL  string `json:"full_url"`
	ShortURL string `json:"short_url"`
	User     string `json:"user"`
}

type ItemURL struct {
	ShortURL string `json:"short_url"`
	FullURL  string `json:"original_url"`
}

type Repository struct {
	config config.Config
	items  []item
}

type FileRepository struct {
	config config.Config
}

type PGRepository struct {
	DB     *sql.DB
	config config.Config
	ctx    context.Context
}

func New(config config.Config, ctx context.Context) URLRepository {
	var r URLRepository
	if config.FileStoragePath != "" {
		r = &FileRepository{config}
	} else {
		r = &Repository{config: config}
	}

	if config.DBDSN == "" {
		return r
	}

	db, err := sql.Open("pgx", config.DBDSN)
	if err != nil {
		db.Close()
	} else {
		_, err = db.Exec("create table if not exists urls (id BIGSERIAL primary key, full_url text, user_token text)")
		if err != nil {
			log.Fatalln(err.Error())
		}

		r = &PGRepository{
			DB:     db,
			ctx:    ctx,
			config: config,
		}
	}

	return r
}

func (r PGRepository) Store(url string, userToken string) (string, error) {
	var shortURL string

	err := r.DB.QueryRowContext(
		r.ctx,
		"select id from urls where full_url = $1 and user_token = $2",
		url,
		userToken,
	).Scan(&shortURL)

	if err != nil && err != sql.ErrNoRows {
		return "", err
	}

	if shortURL != "" {
		return r.config.BaseURL + "/" + shortURL, nil
	}

	err = r.DB.QueryRowContext(
		r.ctx,
		"insert into urls (full_url, user_token) VALUES ($1, $2) RETURNING id",
		url,
		userToken,
	).Scan(&shortURL)

	if err != nil {
		return "", err
	}

	return r.config.BaseURL + "/" + shortURL, nil
}

func (r PGRepository) Find(shortURL string) (string, error) {
	var fullURL string

	err := r.DB.QueryRowContext(
		r.ctx,
		"select full_url from urls where id = $1",
		shortURL,
	).Scan(&fullURL)

	if err != nil {
		return "", err
	}

	return fullURL, nil
}

func (r PGRepository) GetByUser(token string) ([]ItemURL, error) {
	var res []ItemURL
	var itemURL ItemURL

	row, err := r.DB.QueryContext(
		r.ctx,
		"select id, full_url from urls where user_token = $1",
		token,
	)
	if err != nil {
		return nil, err
	}

	for row.Next() {
		err = row.Scan(&itemURL.ShortURL, &itemURL.FullURL)
		if err != nil {
			return nil, err
		}

		res = append(res, itemURL)
	}

	return res, nil
}

func (r PGRepository) PingDB() bool {
	ctx, cancel := context.WithTimeout(r.ctx, 1*time.Second)
	defer cancel()

	err := r.DB.PingContext(ctx)
	if err != nil {
		log.Print(err.Error())
		return false
	}

	return true
}

func (r *FileRepository) Store(newURL string, userToken string) (string, error) {
	file, err := os.OpenFile(r.config.FileStoragePath, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0777)
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
		r.config.BaseURL + "/" + strconv.Itoa(ln+1),
		userToken,
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
	file, err := os.OpenFile(r.config.FileStoragePath, os.O_APPEND|os.O_CREATE|os.O_RDONLY, 0777)
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

func (r *FileRepository) GetByUser(token string) ([]ItemURL, error) {
	file, err := os.OpenFile(r.config.FileStoragePath, os.O_APPEND|os.O_CREATE|os.O_RDONLY, 0777)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var items []ItemURL

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		item := item{}
		data := scanner.Bytes()
		err := json.Unmarshal(data, &item)
		if err != nil {
			return nil, err
		}

		if token == item.User {
			items = append(items, ItemURL{
				item.ShortURL,
				item.FullURL,
			})
		}
	}

	return items, nil
}

func (r *FileRepository) PingDB() bool {
	return true
}

func (r *Repository) Store(u string, userToken string) (string, error) {

	for i := 0; i < len(r.items); i++ {
		if r.items[i].FullURL == u {
			return strconv.Itoa(i + 1), nil
		}
	}

	curLen := len(r.items)

	newItem := item{
		FullURL:  u,
		ShortURL: r.config.BaseURL + "/" + strconv.Itoa(curLen+1),
		User:     userToken,
	}
	r.items = append(r.items, newItem)

	return newItem.ShortURL, nil
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

func (r *Repository) GetByUser(token string) ([]ItemURL, error) {

	var res []ItemURL

	for i := range r.items {
		if r.items[i].User == token {
			resItem := ItemURL{
				r.items[i].ShortURL,
				r.items[i].FullURL,
			}
			res = append(res, resItem)
		}
	}

	return res, nil
}

func (r *Repository) PingDB() bool {
	return true
}
