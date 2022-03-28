package cookieManager

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"log"
	"net/http"
)

var CookieName = "token"
var SecretKey = []byte("pd15KD$^")

func Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		cookie, err := request.Cookie(CookieName)
		println("cookie from request: " + cookie.String())
		if err == nil {
			data, err := hex.DecodeString(cookie.Value)
			if err != nil {
				log.Fatal(err)
			}

			h := hmac.New(sha256.New, SecretKey)
			h.Write(data[:16])
			sign := h.Sum(nil)

			if !hmac.Equal(sign, data[16:]) {
				println("not match, generate")
				cookie, err = generateCookie(SecretKey)
				if err != nil {
					log.Fatal(err)
				}
			}

		} else {
			cookie, err = generateCookie(SecretKey)
			if err != nil {
				log.Fatal(err)
			}
		}

		http.SetCookie(writer, cookie)
		next.ServeHTTP(writer, request)
	})
}

func generateCookie(key []byte) (cookie *http.Cookie, err error) {

	b := make([]byte, 16)
	_, err = rand.Read(b)
	if err != nil {
		return nil, err
	}

	h := hmac.New(sha256.New, key)
	h.Write(b)

	cookie = &http.Cookie{
		Name:  CookieName,
		Value: hex.EncodeToString(b) + hex.EncodeToString(h.Sum(nil)),
	}
	println("NEW COOKIE WOWOWO: " + cookie.Value)
	return cookie, nil
}
