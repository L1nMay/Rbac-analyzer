package security

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

type Claims struct {
	Sub   string `json:"sub"` // user id
	Email string `json:"email"`
	Exp   int64  `json:"exp"`
}

func SignJWT(secret []byte, c Claims) (string, error) {
	header := map[string]string{"alg": "HS256", "typ": "JWT"}
	hb, _ := json.Marshal(header)
	cb, _ := json.Marshal(c)

	hEnc := base64.RawURLEncoding.EncodeToString(hb)
	cEnc := base64.RawURLEncoding.EncodeToString(cb)

	msg := hEnc + "." + cEnc
	sig := signHS256(secret, msg)
	return msg + "." + sig, nil
}

func VerifyJWT(secret []byte, token string) (Claims, error) {
	var out Claims
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return out, errors.New("invalid token format")
	}
	msg := parts[0] + "." + parts[1]
	sig := parts[2]

	if !hmac.Equal([]byte(sig), []byte(signHS256(secret, msg))) {
		return out, errors.New("invalid signature")
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return out, errors.New("bad payload encoding")
	}

	if err := json.Unmarshal(payload, &out); err != nil {
		return out, errors.New("bad payload json")
	}

	if out.Exp > 0 && time.Now().Unix() > out.Exp {
		return out, errors.New("token expired")
	}

	return out, nil
}

func signHS256(secret []byte, msg string) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(msg))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
