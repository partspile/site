package registrationticket

import (
	"os"
	"reflect"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/partspile/site/internal/config"
	"github.com/partspile/site/internal/logger"
	"github.com/valyala/fasthttp"
)

const testJWTSecret = "test-jwt-secret-key-for-registrationticket-tests"

func TestMain(m *testing.M) {
	if err := logger.Init("error", "text", ""); err != nil {
		panic(err)
	}
	os.Exit(m.Run())
}

func withSecret(t *testing.T) {
	t.Helper()
	orig := append([]byte(nil), config.JWTSecret...)
	reflect.ValueOf(&config.JWTSecret).Elem().Set(
		reflect.ValueOf([]byte(testJWTSecret)))
	t.Cleanup(func() {
		reflect.ValueOf(&config.JWTSecret).Elem().Set(reflect.ValueOf(orig))
	})
}

func newCtx(t *testing.T) *fiber.Ctx {
	t.Helper()
	app := fiber.New()
	fctx := &fasthttp.RequestCtx{}
	c := app.AcquireCtx(fctx)
	t.Cleanup(func() { app.ReleaseCtx(c) })
	return c
}

func setRequestTicket(c *fiber.Ctx, token string) {
	c.Request().Header.SetCookie("reg_ticket", token)
}

func TestIssueAndConsume(t *testing.T) {
	withSecret(t)
	c := newCtx(t)

	const (
		username = "ticketuser"
		phone    = "+15559874001"
	)

	if err := Issue(c, username, phone); err != nil {
		t.Fatalf("issue: %v", err)
	}

	// Issue writes the cookie on the response; put the same token on the
	// request the way the browser would on the next POST.
	tok, err := sign(username, phone)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	setRequestTicket(c, tok)

	if err := Consume(c, username, phone); err != nil {
		t.Fatalf("consume: %v", err)
	}

	c.Request().Header.DelCookie("reg_ticket")
	if err := Consume(c, username, phone); err == nil {
		t.Fatal("ticket must not work after clear")
	}
}

func TestConsumeRejectsWrongBinding(t *testing.T) {
	withSecret(t)
	c := newCtx(t)

	tok, err := sign("alice", "+15559874002")
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	setRequestTicket(c, tok)
	if err := Consume(c, "bob", "+15559874002"); err == nil {
		t.Fatal("wrong username must fail")
	}

	setRequestTicket(c, tok)
	if err := Consume(c, "alice", "+15559874099"); err == nil {
		t.Fatal("wrong phone must fail")
	}

	setRequestTicket(c, tok)
	if err := Consume(c, "alice", "+15559874002"); err != nil {
		t.Fatalf("correct binding: %v", err)
	}
}

func TestParseRejectsTamperedToken(t *testing.T) {
	withSecret(t)

	tok, err := sign("alice", "+15559874003")
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	tok = tok[:len(tok)-4] + "dead"

	if _, err := parse(tok); err == nil {
		t.Fatal("tampered token must fail")
	}
}
