package handlers

import (
	"net/http"
	"time"

	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"

	"github.com/parts-pile/site/components"
	"github.com/parts-pile/site/user"
	"golang.org/x/crypto/bcrypt"
)

func HandleRegister(w http.ResponseWriter, r *http.Request) {
	currentUser, _ := GetCurrentUser(r)
	_ = components.Page(
		"Register",
		currentUser,
		r.URL.Path,
		[]g.Node{
			components.PageHeader("Register"),
			components.ContentContainer(
				components.FormContainer("registerForm",
					components.FormGroup("Username", "name",
						components.TextInput("name", "name", ""),
					),
					components.FormGroup("Phone Number", "phone",
						components.TextInput("phone", "phone", ""),
					),
					components.FormGroup("Password", "password",
						components.PasswordInput("password", "password"),
					),
					components.FormGroup("Confirm Password", "password2",
						components.PasswordInput("password2", "password2"),
					),
					components.ActionButtons(
						components.StyledButton("Register", components.ButtonPrimary,
							hx.Post("/api/register"),
							hx.Target("#result"),
							hx.Indicator("#registerForm"),
						),
					),
					components.ResultContainer(),
				),
			),
		},
	).Render(w)
}

func HandleRegisterSubmission(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	name := r.FormValue("name")
	phone := r.FormValue("phone")
	password := r.FormValue("password")
	password2 := r.FormValue("password2")

	if password != password2 {
		components.ValidationError("Passwords do not match").Render(w)
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Server error, unable to create your account.", http.StatusInternalServerError)
		return
	}

	if _, err := user.CreateUser(name, phone, string(hashedPassword)); err != nil {
		components.ValidationError("User already exists or another error occurred.").Render(w)
	} else {
		components.SuccessMessageWithRedirect("Registration successful!", "/login").Render(w)
	}
}

func HandleLogin(w http.ResponseWriter, r *http.Request) {
	currentUser, _ := GetCurrentUser(r)
	_ = components.Page(
		"Login",
		currentUser,
		r.URL.Path,
		[]g.Node{
			components.PageHeader("Login"),
			components.ContentContainer(
				components.FormContainer("loginForm",
					components.FormGroup("Username", "name",
						components.TextInput("name", "name", ""),
					),
					components.FormGroup("Password", "password",
						components.PasswordInput("password", "password"),
					),
					components.ActionButtons(
						components.StyledButton("Login", components.ButtonPrimary,
							hx.Post("/api/login"),
							hx.Target("#result"),
							hx.Indicator("#loginForm"),
						),
					),
					components.ResultContainer(),
				),
			),
		},
	).Render(w)
}

func HandleLoginSubmission(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	name := r.FormValue("name")
	password := r.FormValue("password")

	u, err := user.GetUserByName(name)
	if err != nil {
		components.ValidationError("Invalid username or password").Render(w)
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
	if err != nil {
		components.ValidationError("Invalid username or password").Render(w)
		return
	}

	sessionToken, err := user.CreateSession(u.ID)
	if err != nil {
		http.Error(w, "Server error, unable to log you in.", http.StatusInternalServerError)
	} else {
		cookie := &http.Cookie{
			Name:     "session_token",
			Value:    sessionToken,
			Expires:  time.Now().Add(24 * time.Hour),
			HttpOnly: true,
			Path:     "/",
		}
		http.SetCookie(w, cookie)
		components.SuccessMessageWithRedirect("Login successful!", "/").Render(w)
	}
}

func HandleLogout(w http.ResponseWriter, r *http.Request) {
	cookie := &http.Cookie{
		Name:     "session_token",
		Value:    "",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
	}
	http.SetCookie(w, cookie)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
