package hash

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

// Header — имя HTTP-заголовка с подписью тела запроса или ответа.
const Header = "HashSHA256"

// Sum считает HMAC-SHA256 от тела и возвращает его в hex.
// Ключ — секрет, переданный агенту и серверу на старте.
func Sum(body []byte, key string) string {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

// Valid сообщает, совпадает ли полученная hex-подпись с HMAC тела.
func Valid(body []byte, key, sum string) bool {
	got, err := hex.DecodeString(sum)
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write(body)
	return hmac.Equal(mac.Sum(nil), got)
}
