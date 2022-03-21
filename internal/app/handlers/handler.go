package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go-yandex/internal/app/config"
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

		res, err := repository.Store(string(data))
		if err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			return
		}

		writer.Header().Set("content-type", "application/json")
		writer.WriteHeader(http.StatusCreated)
		writer.Write([]byte(config.BaseURL + "/" + res))
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

		fmt.Println(apiItem)

		res, err := repository.Store(apiItem.FullURL)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			return
		}

		var buf bytes.Buffer
		apiRes := apiResult{ShortURL: config.BaseURL + "/" + res}
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
