package handlers

import (
	"bytes"
	"encoding/json"
	"go-yandex/internal/app/config"
	"go-yandex/internal/app/middlewares/cookie_manager"
	"go-yandex/internal/app/storage"
	"io"
	"net/http"
	"strings"
)

type apiItem struct {
	FullURL string `json:"url"`
}

type apiResult struct {
	ShortURL string `json:"result"`
}

func GetURL(repository storage.URLRepository) func(writer http.ResponseWriter, request *http.Request) {

	return func(writer http.ResponseWriter, request *http.Request) {
		urlID := strings.TrimPrefix(request.URL.Path, `/`)

		url, err := repository.Find(urlID)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}

		if url != "" {
			http.Redirect(writer, request, url, http.StatusTemporaryRedirect)
			return
		}

		http.Error(writer, "not found :(", http.StatusBadRequest)
	}
}

func SaveURL(repository storage.URLRepository, config config.Config) func(writer http.ResponseWriter, request *http.Request) {
	return func(writer http.ResponseWriter, request *http.Request) {
		data, err := io.ReadAll(request.Body)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}

		res, err := repository.Store(string(data), getUserToken(request))
		if err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			return
		}

		writer.Header().Set("content-type", "application/json")
		writer.WriteHeader(http.StatusCreated)
		writer.Write([]byte(res))
	}
}

func SaveURLJson(repository storage.URLRepository, config config.Config) func(writer http.ResponseWriter, request *http.Request) {
	return func(writer http.ResponseWriter, request *http.Request) {
		var apiItem apiItem
		err := json.NewDecoder(request.Body).Decode(&apiItem)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			return
		}

		res, err := repository.Store(apiItem.FullURL, getUserToken(request))
		if err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			return
		}

		var buf bytes.Buffer
		apiRes := apiResult{ShortURL: res}
		err = json.NewEncoder(&buf).Encode(apiRes)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			return
		}

		writer.Header().Set("content-type", "application/json")
		writer.WriteHeader(http.StatusCreated)
		_, err = writer.Write(buf.Bytes())

		if err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func GetForUser(repository storage.URLRepository, config config.Config) func(writer http.ResponseWriter, request *http.Request) {
	return func(writer http.ResponseWriter, request *http.Request) {

		ctxToken := getUserToken(request)

		items, err := repository.GetByUser(ctxToken)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			return
		}
		if len(items) == 0 {
			http.Error(writer, "", http.StatusNoContent)
			return
		}

		var buf bytes.Buffer
		err = json.NewEncoder(&buf).Encode(items)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			return
		}

		writer.Header().Set("content-type", "application/json")
		writer.WriteHeader(http.StatusOK)
		_, err = writer.Write(buf.Bytes())

		if err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func GetDBStatus(repository storage.URLRepository) func(writer http.ResponseWriter, request *http.Request) {
	return func(writer http.ResponseWriter, request *http.Request) {
		dbStatus := repository.PingDB()
		if dbStatus {
			writer.WriteHeader(http.StatusOK)
		} else {
			writer.WriteHeader(http.StatusInternalServerError)
		}
	}
}

func getUserToken(r *http.Request) string {
	return r.Context().Value(cookie_manager.GetCookieName()).(string)
}
