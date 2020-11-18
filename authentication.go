package main

import (
	"crypto/md5"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
)

// ExpiresTime of JWT token
var ExpiresTime int64

// SigningKey is secret key used for signing JWT token
var SigningKey = "Welcome to my personal arcserver"

func init() {
	duration, _ := time.ParseDuration("240h")
	ExpiresTime = int64(duration.Seconds())
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	authToken := r.Header.Get("Authorization")
	user, pwd, err := verifyBasicAuth(authToken)
	if err != nil {
		log.Println(err)
		return
	}
	hash := fmt.Sprintf("%x", md5.Sum([]byte(pwd)))
	var (
		userID  int
		pwdHash string
	)
	err = db.QueryRow(sqlStmtQueryLoginInfo, user).Scan(&userID, &pwdHash)
	if err == sql.ErrNoRows || hash != pwdHash {
		http.Error(
			w, `{"success": false, "error_code": 104}`,
			http.StatusForbidden,
		)
		return
	} else if err != nil {
		log.Println(err)
		return
	}

	token := LoginToken{genJWT(userID), "Bearer", true, 0}
	if res, err := json.Marshal(token); err != nil {
		log.Println("Error occured while generating JSON for login token.")
		log.Println(err)
	} else {
		w.Write(res)
	}
}

func verifyBasicAuth(authToken string) (string, string, error) {
	if !strings.HasPrefix(authToken, "Basic ") {
		return "", "", fmt.Errorf("invalid token string: `%s`", authToken)
	}
	authToken = authToken[6:]
	tDec, err := base64.StdEncoding.DecodeString(authToken)
	if err != nil {
		log.Println(err)
	}
	parts := strings.Split(string(tDec), ":")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid token, token was decoded into: `%s`", tDec)
	}
	user, pwd := parts[0], parts[1]
	return user, pwd, nil
}

func genJWT(userID int) string {
	claims := userClaims{
		userID,
		jwt.StandardClaims{
			ExpiresAt: time.Now().Unix() + ExpiresTime,
			Issuer:    "Zrcaea",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(SigningKey))
	if err != nil {
		log.Println(err)
		return ""
	}
	return signedToken
}

func verifyBearerAuth(authToken string) (int, error) {
	if !strings.HasPrefix(authToken, "Bearer ") {
		return 0, fmt.Errorf("invalid token string: `%s`", authToken)
	}
	authToken = authToken[7:]
	token, err := jwt.ParseWithClaims(
		authToken,
		&userClaims{},
		func(token *jwt.Token) (interface{}, error) {
			return []byte(SigningKey), nil
		},
	)
	if err != nil {
		log.Printf("Failed on verifying token `%s`\n", authToken)
		return 0, err
	}
	claims, ok := token.Claims.(*userClaims)
	if !ok {
		return 0, errors.New("Couldn't parse claims")
	} else if claims.ExpiresAt < time.Now().UTC().Unix() {
		return 0, errors.New("JWT is expired")
	}
	return claims.UserID, nil
}
