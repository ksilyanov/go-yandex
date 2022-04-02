package cookie_manager

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"log"
	"net/http"
)

var cookieName = "token"
var secretKey = []byte("pd15KD$^")

func Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		cookie, err := request.Cookie(GetCookieName())
		if err == nil {
			data, err := hex.DecodeString(cookie.Value)
			if err != nil {
				log.Fatal(err)
			}

			h := hmac.New(sha256.New, secretKey)
			h.Write(data[:16])
			sign := h.Sum(nil)

			if !hmac.Equal(sign, data[16:]) {
				cookie, err = GenerateCookie()
				if err != nil {
					log.Fatal(err)
				}
			}

		} else {
			cookie, err = GenerateCookie()
			if err != nil {
				log.Fatal(err)
			}
		}

		http.SetCookie(writer, cookie)

		ctx := context.WithValue(request.Context(), GetCookieName(), cookie.Value)
		next.ServeHTTP(writer, request.WithContext(ctx))
	})
}

func GenerateCookie() (cookie *http.Cookie, err error) {

	b := make([]byte, 16)
	_, err = rand.Read(b)
	if err != nil {
		return nil, err
	}

	h := hmac.New(sha256.New, secretKey)
	h.Write(b)

	cookie = &http.Cookie{
		Name:  GetCookieName(),
		Value: hex.EncodeToString(b) + hex.EncodeToString(h.Sum(nil)),
		Path:  "/",
	}

	return cookie, nil
}

func GetCookieName() string {
	return cookieName
}
