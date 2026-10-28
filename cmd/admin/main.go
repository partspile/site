package main

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/partspile/site/internal/backup"
	"github.com/partspile/site/internal/config"
	"github.com/partspile/site/internal/db"
	"github.com/partspile/site/internal/imagestore"
	"github.com/partspile/site/internal/logger"
	"github.com/partspile/site/internal/message"
	"github.com/partspile/site/internal/user"
	"github.com/partspile/site/internal/vector"
	"golang.org/x/term"
)

type screen int

const (
	screenMenu screen = iota
	screenDBMenu
	screenBackupConfirm
	screenBackupRunning
	screenRestorePick
	screenRestoreKey
	screenRestoreConfirm
	screenRestoreRunning
	screenUsersList
	screenUserDeleteConfirm
	screenUserRunning
	screenEmbMenu
	screenEmbRunning
	screenResult
)

type menuItem int

const (
	menuDatabase menuItem = iota
	menuUsers
	menuEmbeddings
	menuQuit
)

type model struct {
	screen        screen
	menuCursor    int
	dbCursor      int
	embCursor     int
	archives      []string
	archiveCursor int
	keyInput      textinput.Model
	users         []user.User
	userCursor    int
	selected      string
	status        string
	err           string
	resultBack    screen
	store         imagestore.Store
	dbTarget      string
}

type doneMsg struct {
	ok  string
	err error
}

type usersLoadedMsg struct {
	users []user.User
	err   error
}

func initialModel(store imagestore.Store, dbTarget string) model {
	ki := textinput.New()
	ki.Placeholder = "BACKUP_DB_ENCRYPTION_KEY (empty = users.csv or DB key)"
	ki.EchoMode = textinput.EchoPassword
	ki.EchoCharacter = '•'
	ki.Width = 60

	return model{
		screen:     screenMenu,
		keyInput:   ki,
		store:      store,
		dbTarget:   dbTarget,
		resultBack: screenMenu,
	}
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case doneMsg:
		if msg.err != nil {
			m.err = msg.err.Error()
			m.status = ""
		} else {
			m.status = msg.ok
			m.err = ""
		}
		m.screen = screenResult
		return m, nil
	case usersLoadedMsg:
		if msg.err != nil {
			m.err = msg.err.Error()
			m.status = ""
			m.resultBack = screenMenu
			m.screen = screenResult
			return m, nil
		}
		m.users = msg.users
		if m.userCursor >= len(m.users) && len(m.users) > 0 {
			m.userCursor = len(m.users) - 1
		}
		if len(m.users) == 0 {
			m.userCursor = 0
		}
		m.screen = screenUsersList
		return m, nil
	case tea.KeyMsg:
		switch m.screen {
		case screenMenu:
			return m.updateMenu(msg)
		case screenDBMenu:
			return m.updateDBMenu(msg)
		case screenBackupConfirm:
			return m.updateBackupConfirm(msg)
		case screenRestorePick:
			return m.updateRestorePick(msg)
		case screenRestoreKey:
			return m.updateRestoreKey(msg)
		case screenRestoreConfirm:
			return m.updateRestoreConfirm(msg)
		case screenUsersList:
			return m.updateUsersList(msg)
		case screenUserDeleteConfirm:
			return m.updateUserDeleteConfirm(msg)
		case screenEmbMenu:
			return m.updateEmbMenu(msg)
		case screenResult:
			if msg.String() == "enter" || msg.String() == "esc" {
				back := m.resultBack
				m.status = ""
				m.err = ""
				switch back {
				case screenUsersList:
					m.screen = screenUserRunning
					return m, loadUsersCmd()
				case screenDBMenu:
					m.screen = screenDBMenu
				case screenEmbMenu:
					m.screen = screenEmbMenu
				default:
					m.screen = screenMenu
				}
			}
			return m, nil
		}
	}

	var cmd tea.Cmd
	if m.screen == screenRestoreKey {
		m.keyInput, cmd = m.keyInput.Update(msg)
	}
	return m, cmd
}

func loadUsersCmd() tea.Cmd {
	return func() tea.Msg {
		users, err := user.GetAllUsers("id", "ASC")
		return usersLoadedMsg{users: users, err: err}
	}
}

func (m model) updateMenu(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "up", "k":
		if m.menuCursor > 0 {
			m.menuCursor--
		}
	case "down", "j":
		if m.menuCursor < int(menuQuit) {
			m.menuCursor++
		}
	case "enter":
		switch menuItem(m.menuCursor) {
		case menuDatabase:
			m.dbCursor = 0
			m.screen = screenDBMenu
		case menuUsers:
			m.screen = screenUserRunning
			return m, loadUsersCmd()
		case menuEmbeddings:
			m.embCursor = 0
			m.screen = screenEmbMenu
		case menuQuit:
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m model) updateDBMenu(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "esc":
		m.screen = screenMenu
	case "up", "k":
		if m.dbCursor > 0 {
			m.dbCursor--
		}
	case "down", "j":
		if m.dbCursor < 2 {
			m.dbCursor++
		}
	case "b", "B":
		m.screen = screenBackupConfirm
	case "r", "R":
		return m.startRestorePick()
	case "enter":
		switch m.dbCursor {
		case 0:
			m.screen = screenBackupConfirm
		case 1:
			return m.startRestorePick()
		case 2:
			m.screen = screenMenu
		}
	}
	return m, nil
}

func (m model) startRestorePick() (tea.Model, tea.Cmd) {
	names, err := backup.ListArchives()
	if err != nil {
		m.err = err.Error()
		m.resultBack = screenDBMenu
		m.screen = screenResult
		return m, nil
	}
	if len(names) == 0 {
		m.err = "no backups in " + backup.DefaultDir + "/"
		m.resultBack = screenDBMenu
		m.screen = screenResult
		return m, nil
	}
	m.archives = names
	m.archiveCursor = 0
	m.screen = screenRestorePick
	return m, nil
}

func (m model) updateEmbMenu(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "esc":
		m.screen = screenMenu
	case "up", "k":
		if m.embCursor > 0 {
			m.embCursor--
		}
	case "down", "j":
		if m.embCursor < 1 {
			m.embCursor++
		}
	case "b", "B":
		return m.startBackfill()
	case "enter":
		if m.embCursor == 0 {
			return m.startBackfill()
		}
		m.screen = screenMenu
	}
	return m, nil
}

func (m model) startBackfill() (tea.Model, tea.Cmd) {
	m.screen = screenEmbRunning
	m.resultBack = screenEmbMenu
	return m, func() tea.Msg {
		if err := vector.InitEmbedder(); err != nil {
			return doneMsg{err: fmt.Errorf("embedder: %w", err)}
		}
		remaining, err := vector.BackfillAllAdsSync()
		if err != nil {
			return doneMsg{err: err}
		}
		if remaining == 0 {
			return doneMsg{ok: "Backfill complete (nothing left missing)"}
		}
		return doneMsg{ok: fmt.Sprintf(
			"Backfill pass done; %d ads still missing", remaining,
		)}
	}
}

func (m model) updateUsersList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "esc":
		m.screen = screenMenu
		return m, nil
	case "up", "k":
		if m.userCursor > 0 {
			m.userCursor--
		}
	case "down", "j":
		if m.userCursor < len(m.users)-1 {
			m.userCursor++
		}
	case "p", "P":
		return m.runUserAction("promote")
	case "d", "D":
		return m.runUserAction("demote")
	case "s", "S":
		return m.runUserAction("phone")
	case "x", "X":
		if len(m.users) == 0 {
			return m, nil
		}
		u := m.users[m.userCursor]
		if u.DeletedAt != nil {
			m.err = u.Name + " is already deleted"
			m.resultBack = screenUsersList
			m.screen = screenResult
			return m, nil
		}
		m.screen = screenUserDeleteConfirm
	}
	return m, nil
}

func (m model) selectedUser() (user.User, bool) {
	if len(m.users) == 0 || m.userCursor < 0 ||
		m.userCursor >= len(m.users) {
		return user.User{}, false
	}
	return m.users[m.userCursor], true
}

func (m model) runUserAction(action string) (tea.Model, tea.Cmd) {
	u, ok := m.selectedUser()
	if !ok {
		return m, nil
	}
	m.screen = screenUserRunning
	m.resultBack = screenUsersList
	store := m.store
	id := u.ID
	name := u.Name
	isAdmin := u.IsAdmin
	deleted := u.DeletedAt != nil
	return m, func() tea.Msg {
		switch action {
		case "promote":
			if deleted {
				return doneMsg{err: fmt.Errorf("%s is deleted", name)}
			}
			if isAdmin {
				return doneMsg{ok: name + " is already an admin"}
			}
			if err := user.PromoteToAdmin(id); err != nil {
				return doneMsg{err: err}
			}
			return doneMsg{ok: "Promoted " + name + " to admin"}
		case "demote":
			if deleted {
				return doneMsg{err: fmt.Errorf("%s is deleted", name)}
			}
			if !isAdmin {
				return doneMsg{ok: name + " is not an admin"}
			}
			if err := user.DemoteFromAdmin(id); err != nil {
				return doneMsg{err: err}
			}
			return doneMsg{ok: "Demoted " + name + " from admin"}
		case "phone":
			full, err := user.GetByIDIncludingDeleted(id)
			if err != nil {
				return doneMsg{err: err}
			}
			opt := "SMS notifications on"
			if full.SMSOptedOut {
				opt = "SMS opted out"
			}
			return doneMsg{ok: fmt.Sprintf(
				"%s\nphone: %s\n%s", full.Name, full.PhoneE64, opt,
			)}
		case "delete":
			if err := user.DeleteUser(id); err != nil {
				return doneMsg{err: err}
			}
			_ = store.DeleteUserAccount(id)
			_, _ = message.CloseConversationsForDeletedAccount(id)
			return doneMsg{ok: "Deleted " + name}
		}
		return doneMsg{err: fmt.Errorf("unknown action")}
	}
}

func (m model) updateUserDeleteConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "n":
		m.screen = screenUsersList
	case "y":
		return m.runUserAction("delete")
	}
	return m, nil
}

func (m model) updateBackupConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.screen = screenDBMenu
	case "y", "enter":
		m.screen = screenBackupRunning
		m.resultBack = screenDBMenu
		store := m.store
		return m, func() tea.Msg {
			path, err := backup.BackupToArchive("", store, false, false)
			if err != nil {
				return doneMsg{err: err}
			}
			return doneMsg{ok: "Backup written: " + path}
		}
	}
	return m, nil
}

func (m model) updateRestorePick(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.screen = screenDBMenu
		return m, nil
	case "up", "k":
		if m.archiveCursor > 0 {
			m.archiveCursor--
		}
	case "down", "j":
		if m.archiveCursor < len(m.archives)-1 {
			m.archiveCursor++
		}
	case "enter":
		if len(m.archives) == 0 {
			return m, nil
		}
		m.selected = m.archives[m.archiveCursor]
		m.keyInput.SetValue("")
		m.keyInput.Focus()
		m.screen = screenRestoreKey
		return m, textinput.Blink
	}
	return m, nil
}

func (m model) updateRestoreKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.screen = screenRestorePick
		return m, nil
	case "enter":
		m.screen = screenRestoreConfirm
		return m, nil
	}
	var cmd tea.Cmd
	m.keyInput, cmd = m.keyInput.Update(msg)
	return m, cmd
}

func (m model) updateRestoreConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "n":
		m.screen = screenRestoreKey
	case "y":
		m.screen = screenRestoreRunning
		m.resultBack = screenDBMenu
		store := m.store
		selected := m.selected
		keyStr := strings.TrimSpace(m.keyInput.Value())
		return m, func() tea.Msg {
			var backupKey []byte
			if keyStr != "" {
				k, err := base64.StdEncoding.DecodeString(keyStr)
				if err != nil || len(k) != 32 {
					return doneMsg{err: fmt.Errorf(
						"BACKUP_DB_ENCRYPTION_KEY must be 32-byte base64")}
				}
				backupKey = k
			}
			err := backup.RestoreFromArchive(
				selected, store, backupKey, false, false,
			)
			if err != nil {
				return doneMsg{err: err}
			}
			return doneMsg{ok: "Restored from " + selected}
		}
	}
	return m, nil
}

func (m model) View() string {
	title := lipgloss.NewStyle().Bold(true).Render(
		config.ServerName + " Admin")
	if m.dbTarget != "" {
		muted := lipgloss.NewStyle().Faint(true).Render(m.dbTarget)
		title = title + "\n" + muted
	}
	var body string
	switch m.screen {
	case screenMenu:
		body = menuView(m.menuCursor, []string{
			"Database",
			"Users",
			"Embeddings",
			"Quit",
		}, "↑/↓ move · enter select · q quit")
	case screenDBMenu:
		body = menuView(m.dbCursor, []string{
			"[B] Backup database",
			"[R] Restore database",
			"Back",
		}, "↑/↓ or letter · enter select · esc back")
	case screenEmbMenu:
		body = menuView(m.embCursor, []string{
			"[B] Backfill missing embeddings",
			"Back",
		}, "↑/↓ or b · enter select · esc back")
	case screenUsersList:
		body = m.usersListView()
	case screenUserDeleteConfirm:
		name := "?"
		if u, ok := m.selectedUser(); ok {
			name = u.Name
		}
		body = fmt.Sprintf(
			"Delete user %s?\nThis soft-deletes the account.\n\ny confirm · n/esc cancel",
			name,
		)
	case screenBackupConfirm:
		body = "Create backup in " + backup.DefaultDir + "/ ?\n\n" +
			"y/enter confirm · esc cancel"
	case screenBackupRunning, screenRestoreRunning,
		screenUserRunning, screenEmbRunning:
		body = "Working…"
	case screenRestorePick:
		var b strings.Builder
		b.WriteString("Select backup:\n\n")
		for i, name := range m.archives {
			cursor := "  "
			if i == m.archiveCursor {
				cursor = "> "
			}
			b.WriteString(cursor + name + "\n")
		}
		b.WriteString("\n↑/↓ move · enter select · esc cancel")
		body = b.String()
	case screenRestoreKey:
		body = "Archive: " + m.selected + "\n\n" +
			"Source DB encryption key (base64):\n" + m.keyInput.View() +
			"\n\nenter continue · esc back"
	case screenRestoreConfirm:
		body = fmt.Sprintf(
			"Restore %s?\nThis WIPES the current database.\n\ny confirm · n/esc cancel",
			m.selected,
		)
	case screenResult:
		if m.err != "" {
			body = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).
				Render("Error: "+m.err) + "\n\nenter/esc back"
		} else {
			body = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).
				Render(m.status) + "\n\nenter/esc back"
		}
	}
	return title + "\n\n" + body + "\n"
}

func menuView(cursor int, items []string, help string) string {
	var b strings.Builder
	b.WriteString("Choose an action:\n\n")
	for i, item := range items {
		cursorMark := "  "
		if i == cursor {
			cursorMark = "> "
		}
		b.WriteString(cursorMark + item + "\n")
	}
	b.WriteString("\n" + help)
	return b.String()
}

func (m model) usersListView() string {
	var b strings.Builder
	b.WriteString("Users (no phones until [S]):\n\n")
	b.WriteString(fmt.Sprintf(
		"  %-4s %-18s %-5s %-19s %s\n",
		"ID", "Name", "Admin", "Created", "Deleted",
	))
	if len(m.users) == 0 {
		b.WriteString("  (no users)\n")
	}
	for i, u := range m.users {
		cursor := "  "
		if i == m.userCursor {
			cursor = "> "
		}
		admin := "no"
		if u.IsAdmin {
			admin = "yes"
		}
		deleted := "—"
		if u.DeletedAt != nil {
			deleted = u.DeletedAt.Format(time.DateTime)
		}
		name := u.Name
		if len(name) > 18 {
			name = name[:15] + "…"
		}
		b.WriteString(fmt.Sprintf(
			"%s%-4d %-18s %-5s %-19s %s\n",
			cursor, u.ID, name, admin,
			u.CreatedAt.Format(time.DateTime), deleted,
		))
	}
	b.WriteString("\n↑/↓ select · [P]romote [D]emote [S]how phone [X] delete · esc back")
	return b.String()
}

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "-h", "--help", "help":
			fmt.Print(`Usage:
  admin    Interactive TUI (Database, Users, Embeddings)
`)
			return
		default:
			fmt.Fprintf(os.Stderr, "unknown argument %q (try -h)\n", os.Args[1])
			os.Exit(1)
		}
	}
	runTUI()
}

func runTUI() {
	if !term.IsTerminal(int(os.Stdin.Fd())) ||
		!term.IsTerminal(int(os.Stdout.Fd())) {
		fmt.Fprintln(os.Stderr,
			"admin requires a TTY (use ssh -t, or docker compose exec -it)")
		os.Exit(1)
	}

	mustInit(true)
	defer db.Close()

	host, database := db.ConnectionTarget(config.DatabaseURL)
	dbTarget := database
	if host != "" {
		dbTarget = database + " @ " + host
	}
	fmt.Fprintf(os.Stderr, "%s Admin — connected to %s\n",
		config.ServerName, dbTarget)

	store, err := imagestore.NewDefault()
	if err != nil {
		fmt.Fprintf(os.Stderr, "image store: %v\n", err)
		os.Exit(1)
	}

	p := tea.NewProgram(
		initialModel(store, dbTarget), tea.WithAltScreen(),
	)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "admin: %v\n", err)
		os.Exit(1)
	}
}

func mustInit(quiet bool) {
	logOut := ""
	if quiet {
		logOut = os.DevNull
	}
	if err := logger.Init("info", "text", logOut); err != nil {
		fmt.Fprintf(os.Stderr, "logger: %v\n", err)
		os.Exit(1)
	}
	if config.DatabaseURL == "" {
		fmt.Fprintln(os.Stderr, "DATABASE_URL must be set")
		os.Exit(1)
	}
	if len(config.DBEncryptionKey) != 32 {
		fmt.Fprintln(os.Stderr,
			"DB_ENCRYPTION_KEY must be set (32-byte base64)")
		os.Exit(1)
	}
	if len(config.DBHashPepper) != 32 {
		fmt.Fprintln(os.Stderr,
			"DB_HASH_PEPPER must be set (32-byte base64, "+
				"distinct from DB_ENCRYPTION_KEY)")
		os.Exit(1)
	}
	if config.ServerName == "" {
		fmt.Fprintln(os.Stderr, "APP_NAME must be set")
		os.Exit(1)
	}
	if config.ContactEmail == "" {
		fmt.Fprintln(os.Stderr, "CONTACT_EMAIL must be set")
		os.Exit(1)
	}
	db.SetHashPepper(config.DBHashPepper)
	if err := db.Init(config.DatabaseURL); err != nil {
		fmt.Fprintf(os.Stderr, "database: %v\n", err)
		os.Exit(1)
	}
}
