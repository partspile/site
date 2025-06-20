package handlers

import (
	"net/http"
	"time"

	"github.com/parts-pile/site/components"
	"github.com/parts-pile/site/user"
	"golang.org/x/crypto/bcrypt"
)

func HandleSettings(w http.ResponseWriter, r *http.Request) {
	currentUser, err := GetCurrentUser(r)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	_ = components.SettingsPage(currentUser, r.URL.Path).Render(w)
}

func HandleChangePassword(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	currentPassword := r.FormValue("currentPassword")
	newPassword := r.FormValue("newPassword")
	confirmNewPassword := r.FormValue("confirmNewPassword")

	if newPassword != confirmNewPassword {
		components.ValidationError("New passwords do not match").Render(w)
		return
	}

	currentUser, err := GetCurrentUser(r)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(currentUser.PasswordHash), []byte(currentPassword))
	if err != nil {
		components.ValidationError("Invalid current password").Render(w)
		return
	}

	newHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Server error, unable to update password.", http.StatusInternalServerError)
		return
	}

	if _, err := user.UpdateUserPassword(currentUser.ID, string(newHash)); err != nil {
		components.ValidationError("Failed to update password").Render(w)
	} else {
		components.SuccessMessage("Password changed successfully", "").Render(w)
	}
}

func HandleDeleteAccount(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	password := r.FormValue("password")

	currentUser, err := GetCurrentUser(r)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(currentUser.PasswordHash), []byte(password))
	if err != nil {
		components.ValidationError("Invalid password").Render(w)
		return
	}

	if err := user.DeleteUser(currentUser.ID); err != nil {
		components.ValidationError("Failed to delete account").Render(w)
	} else {
		// Clear session cookie
		cookie := &http.Cookie{
			Name:     "session_token",
			Value:    "",
			Expires:  time.Unix(0, 0),
			HttpOnly: true,
		}
		http.SetCookie(w, cookie)
		w.Header().Set("HX-Redirect", "/")
	}
}
