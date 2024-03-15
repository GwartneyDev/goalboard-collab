package auth

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"

	"github.com/benzend/goalboard/utils"
	"github.com/golang-jwt/jwt/v5"
)

type user struct {
	ID string `json:"id"`
	Username string `json:"username"`
}

func Authorize(ctx context.Context, w http.ResponseWriter, req *http.Request) (user user, err error) {
	    // Parse and validate the JWT token from the cookie
	sessionInfo, err := req.Cookie("jwt_token")
	if err != nil {
		http.Error(w, "no cookie", http.StatusUnauthorized)
		return
	}

	tokenString := sessionInfo.Value

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// Validate the signing method
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
			}
			// Return the byte array representation of the secret key
			return utils.GetJwtSecret(), nil
	})

	if err != nil {
		log.Println("failed to parse token", err)
		http.Error(w, "failed to parse token", http.StatusInternalServerError)
		return
	}

	// Extract claims from the token
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		// Token is invalid or claims couldn't be extracted
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		err = fmt.Errorf("invalid token")
		return
	}

	userID, ok := claims["user_id"]

	if !ok {
		http.Error(w, "no user ID", http.StatusUnauthorized)
		err = fmt.Errorf("no user ID")
		return
	}

	db, ok := ctx.Value(utils.CTX_KEY_DB).(*sql.DB)

	if !ok {
		http.Error(w, "failed to get db", http.StatusInternalServerError)
		err = fmt.Errorf("failed to get db")
		return
	}

	query := "SELECT (id, username) FROM user_ WHERE id = $1"

	err = db.QueryRow(query, userID).Scan(&user)

	if err != nil {
		http.Error(w, "failed to get user", http.StatusInternalServerError)
		return
	}

	return
}
