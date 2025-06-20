package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/parts-pile/site/templates"
	"github.com/parts-pile/site/user"
)

func AdminRequired(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		currentUser, err := GetCurrentUser(r)
		if err != nil || currentUser == nil || !currentUser.IsAdmin {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		next(w, r)
	}
}

func HandleAdminDashboard(w http.ResponseWriter, r *http.Request) {
	currentUser, _ := GetCurrentUser(r)
	_ = templates.AdminDashboard(currentUser, r.URL.Path).Render(w)
}

func HandleAdminUsers(w http.ResponseWriter, r *http.Request) {
	currentUser, _ := GetCurrentUser(r)
	users, err := user.GetAllUsers()
	if err != nil {
		http.Error(w, "could not get users", http.StatusInternalServerError)
		return
	}
	_ = templates.AdminUsers(currentUser, r.URL.Path, users).Render(w)
}

func HandleSetAdmin(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Error parsing form", http.StatusBadRequest)
		return
	}
	userIDStr := r.FormValue("user_id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}
	isAdmin := r.FormValue("is_admin") == "true"

	if err := user.SetAdmin(userID, isAdmin); err != nil {
		http.Error(w, "could not update user", http.StatusInternalServerError)
		return
	}

	users, err := user.GetAllUsers()
	if err != nil {
		http.Error(w, "could not get users", http.StatusInternalServerError)
		return
	}

	_ = templates.AdminUserTable(users).Render(w)
}

func HandleAdminAds(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Admin Ads")
}

func HandleAdminTransactions(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Admin Transactions")
}

func HandleAdminExport(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Admin Export")
}
