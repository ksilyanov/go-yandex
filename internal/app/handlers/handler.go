package handlers

import (
	"go-yandex/internal/app/storage"
	"io"
	"net/http"
	"strings"
)

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

func SaveURL(repository storage.URLRepository) func(writer http.ResponseWriter, request *http.Request) {
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
		writer.Write([]byte("http://localhost:8080/" + res))
	}
}
