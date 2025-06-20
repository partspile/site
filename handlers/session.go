package handlers

import (
	"net/http"

	"github.com/parts-pile/site/user"
)

func GetCurrentUserID(r *http.Request) (int, error) {
	c, err := r.Cookie("session_token")
	if err != nil {
		if err == http.ErrNoCookie {
			return 0, nil
		}
		return 0, err
	}
	return user.VerifySession(c.Value)
}

func GetCurrentUser(r *http.Request) (*user.User, error) {
	userID, err := GetCurrentUserID(r)
	if err != nil || userID == 0 {
		return nil, err
	}
	u, err := user.GetUserByID(userID)
	if err != nil {
		return nil, err
	}
	return &u, nil
}
