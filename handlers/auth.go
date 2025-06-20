package handlers

import (
	"net/http"
	"time"

	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"

	"github.com/parts-pile/site/templates"
	"github.com/parts-pile/site/user"
	"golang.org/x/crypto/bcrypt"
)

func HandleRegister(w http.ResponseWriter, r *http.Request) {
	currentUser, _ := GetCurrentUser(r)
	_ = templates.Page(
		"Register",
		currentUser,
		r.URL.Path,
		[]g.Node{
			templates.PageHeader("Register"),
			templates.ContentContainer(
				templates.FormContainer("registerForm",
					templates.FormGroup("Username", "name",
						templates.TextInput("name", "name", ""),
					),
					templates.FormGroup("Phone Number", "phone",
						templates.TextInput("phone", "phone", ""),
					),
					templates.FormGroup("Password", "password",
						templates.PasswordInput("password", "password"),
					),
					templates.FormGroup("Confirm Password", "password2",
						templates.PasswordInput("password2", "password2"),
					),
					templates.ActionButtons(
						templates.StyledButton("Register", templates.ButtonPrimary,
							hx.Post("/api/register"),
							hx.Target("#result"),
							hx.Indicator("#registerForm"),
						),
					),
					templates.ResultContainer(),
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
		templates.ValidationError("Passwords do not match").Render(w)
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Server error, unable to create your account.", http.StatusInternalServerError)
		return
	}

	if _, err := user.CreateUser(name, phone, string(hashedPassword)); err != nil {
		templates.ValidationError("User already exists or another error occurred.").Render(w)
	} else {
		templates.SuccessMessageWithRedirect("Registration successful!", "/login").Render(w)
	}
}

func HandleLogin(w http.ResponseWriter, r *http.Request) {
	currentUser, _ := GetCurrentUser(r)
	_ = templates.Page(
		"Login",
		currentUser,
		r.URL.Path,
		[]g.Node{
			templates.PageHeader("Login"),
			templates.ContentContainer(
				templates.FormContainer("loginForm",
					templates.FormGroup("Username", "name",
						templates.TextInput("name", "name", ""),
					),
					templates.FormGroup("Password", "password",
						templates.PasswordInput("password", "password"),
					),
					templates.ActionButtons(
						templates.StyledButton("Login", templates.ButtonPrimary,
							hx.Post("/api/login"),
							hx.Target("#result"),
							hx.Indicator("#loginForm"),
						),
					),
					templates.ResultContainer(),
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
		templates.ValidationError("Invalid username or password").Render(w)
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
	if err != nil {
		templates.ValidationError("Invalid username or password").Render(w)
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
		templates.SuccessMessageWithRedirect("Login successful!", "/").Render(w)
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
