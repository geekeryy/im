package jwt

import (
	"errors"
	"fmt"
	"maps"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
)

var (
	secret = []byte("im-gateway-secret")
)

func GenerateToken(uuid string,  expire int64, extraClaims jwt.MapClaims) (string, time.Time, error) {
	now := time.Now()
	exp := now.Add(time.Duration(expire) * time.Second)
	claims := map[string]interface{}{
		"exp": exp.Unix(),
		"sub": uuid,
		"iat": now.Unix(),
		"nbf": now.Unix(),
		"iss": "im-gateway",
		"aud": "member",
	}
	maps.Copy(claims, extraClaims)
	tokenOption := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims(claims))
	token, err := tokenOption.SignedString(secret)
	if err != nil {
		return "", time.Time{}, err
	}
	return token, exp, nil
}

func ValidateToken(tokenStr string) (jwt.MapClaims, error) {
	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

func MapClaimsParseString(claims map[string]interface{}, key string) (string, error) {
	var (
		ok  bool
		raw interface{}
		s   string
	)
	raw, ok = claims[key]
	if !ok {
		return "", nil
	}

	s, ok = raw.(string)
	if !ok {
		return "", fmt.Errorf("%s is invalid", key)
	}

	return s, nil
}
