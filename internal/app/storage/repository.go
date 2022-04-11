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
	Batch(items []BatchItem, token string) ([]BatchResultItem, error)
}

type item struct {
	FullURL  string `json:"full_url"`
	ShortURL string `json:"short_url"`
	User     string `json:"user_token"`
	CorrID   string `json:"correlation_id"`
}

type ItemURL struct {
	ShortURL string `json:"short_url"`
	FullURL  string `json:"original_url"`
}

type BatchItem struct {
	CorrectionID string `json:"correlation_id"`
	OriginalURL  string `json:"original_url"`
}

type BatchResultItem struct {
	CorrectionID string `json:"correlation_id"`
	ShortURL     string `json:"short_url"`
}

type Repository struct {
	config config.Config
	items  []item
}

type FileRepository struct {
	config config.Config
}

type PGRepository struct {
	DB     *DataBase
	config config.Config
	ctx    context.Context
}

type DataBase struct {
	conn *sql.DB
}

func New(config config.Config, ctx context.Context) URLRepository {
	var r URLRepository

	if config.DBDSN != "" {
		db, err := sql.Open("pgx", config.DBDSN)
		if err != nil {
			db.Close()
			log.Fatalln(err.Error())
		} else {
			_, err = db.Exec("create table if not exists urls (id BIGSERIAL primary key, full_url text, user_token text, correlation_id text)")
			if err != nil {
				db.Close()
				log.Fatalln(err.Error())
			}

			if err := db.Ping(); err != nil {
				db.Close()
				log.Fatalln(err.Error())
			}

			_, err := db.Exec("select 'public.urls'::regclass")
			if err != nil {
				println(err.Error())
				db.Close()
			}

			_, err = db.Exec("create unique index if not exists urls_full_url_uindex on urls (full_url);")
			if err != nil {
				db.Close()
				log.Fatalln(err.Error())
			}

			dataBase := &DataBase{
				conn: db,
			}

			r = &PGRepository{
				DB:     dataBase,
				ctx:    ctx,
				config: config,
			}

			return r
		}
	}

	if config.FileStoragePath != "" {
		r = &FileRepository{config}
		return r
	}

	return &Repository{config: config}
}

func (r PGRepository) Store(url string, userToken string) (string, error) {
	var shortURL string

	err := r.DB.conn.QueryRowContext(
		r.ctx,
		"select id from urls where full_url = $1",
		url,
	).Scan(&shortURL)

	if err != nil && err != sql.ErrNoRows {
		return "", err
	}

	if shortURL != "" {
		return r.config.BaseURL + "/" + shortURL, nil
	}

	err = r.DB.conn.QueryRowContext(
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

	err := r.DB.conn.QueryRowContext(
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

	row, err := r.DB.conn.QueryContext(
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

		itemURL.ShortURL = r.config.BaseURL + "/" + itemURL.ShortURL

		res = append(res, itemURL)
	}

	err = row.Err()
	if err != nil {
		return res, err
	}

	return res, nil
}

func (r PGRepository) PingDB() bool {
	ctx, cancel := context.WithTimeout(r.ctx, 1*time.Second)
	defer cancel()

	err := r.DB.conn.PingContext(ctx)
	if err != nil {
		log.Print(err.Error())
		return false
	}

	return true
}

func (r PGRepository) Batch(items []BatchItem, token string) ([]BatchResultItem, error) {
	var res []BatchResultItem
	var id = 0

	for _, batchItem := range items {
		row := r.DB.conn.QueryRowContext(
			r.ctx,
			"insert into urls (full_url, correlation_id, user_token) VALUES ($1, $2, $3)"+
				" on conflict(full_url) do update set full_url = excluded.full_url, correlation_id = $2"+
				" returning id",
			batchItem.OriginalURL,
			batchItem.CorrectionID,
			token,
		)
		err := row.Scan(&id)
		if err != nil {
			return nil, err
		}

		resItem := BatchResultItem{
			CorrectionID: batchItem.CorrectionID,
			ShortURL:     r.config.BaseURL + "/" + strconv.Itoa(id),
		}

		res = append(res, resItem)
	}

	return res, nil
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
		"",
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

func (r *FileRepository) Batch(items []BatchItem, token string) ([]BatchResultItem, error) {
	var res []BatchResultItem

	file, err := os.OpenFile(r.config.FileStoragePath, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0777)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	itemsToSkip := make(map[int]struct{})

	scanner := bufio.NewScanner(file)
	ln := 0
	for scanner.Scan() {
		ln++
		data := scanner.Bytes()

		existsBatchItem := item{}
		if err := json.Unmarshal(data, &existsBatchItem); err != nil {
			return nil, err
		}
		existsBatchResultItem := BatchResultItem{}
		if err := json.Unmarshal(data, &existsBatchResultItem); err != nil {
			return nil, err
		}

		for i, item := range items {
			if existsBatchItem.FullURL != item.OriginalURL {
				continue
			}
			itemsToSkip[i] = struct{}{}
			existsBatchResultItem.CorrectionID = item.CorrectionID
			res = append(res, existsBatchResultItem)

			break
		}
	}

	writer := bufio.NewWriter(file)
	for i, batchItem := range items {
		_, ok := itemsToSkip[i]
		if ok {
			continue
		}

		newShortURL := r.config.BaseURL + "/" + strconv.Itoa(ln+1)
		newItem := item{
			FullURL:  batchItem.OriginalURL,
			ShortURL: newShortURL,
			User:     token,
			CorrID:   batchItem.CorrectionID,
		}

		data, err := json.Marshal(&newItem)
		if err != nil {
			return nil, err
		}
		if _, err := writer.Write(data); err != nil {
			return nil, err
		}
		if _, err := writer.Write([]byte("\n")); err != nil {
			return nil, err
		}

		ln++

		res = append(res, BatchResultItem{
			CorrectionID: batchItem.CorrectionID,
			ShortURL:     newShortURL,
		})
	}
	if err := writer.Flush(); err != nil {
		return nil, err
	}

	return res, nil
}

func (r *Repository) Store(u string, userToken string) (string, error) {

	for i := 0; i < len(r.items); i++ {
		if r.items[i].FullURL == u {
			return r.config.BaseURL + "/" + strconv.Itoa(i+1), nil
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

func (r *Repository) Batch(items []BatchItem, token string) ([]BatchResultItem, error) {
	var res []BatchResultItem
	itemsToSkip := make(map[int]struct{})
	curLen := len(r.items)

	for i, newItem := range items {
		for _, existsItem := range r.items {
			if existsItem.FullURL == newItem.OriginalURL {
				existsItem.CorrID = newItem.CorrectionID

				res = append(res, BatchResultItem{
					CorrectionID: existsItem.CorrID,
					ShortURL:     existsItem.ShortURL,
				})

				itemsToSkip[i] = struct{}{}
			}
		}
	}

	for i, newItem := range items {
		_, ok := itemsToSkip[i]
		if ok {
			continue
		}

		itemToAdd := item{
			FullURL:  newItem.OriginalURL,
			ShortURL: r.config.BaseURL + "/" + strconv.Itoa(curLen+1),
			User:     token,
			CorrID:   newItem.CorrectionID,
		}
		r.items = append(r.items, itemToAdd)

		res = append(res, BatchResultItem{
			CorrectionID: itemToAdd.CorrID,
			ShortURL:     itemToAdd.ShortURL,
		})
	}

	return res, nil
}
