package ad_test

import (
	"testing"
	"time"

	"github.com/partspile/site/internal/ad"
	"github.com/partspile/site/internal/db"
	"github.com/partspile/site/internal/db/testdb"
	"github.com/partspile/site/internal/user"
)

func resetFunStatsDB(t *testing.T) {
	t.Helper()
	if err := testdb.InitSchema(); err != nil {
		t.Fatalf("init schema: %v", err)
	}
}

func monthUTC(now time.Time, delta int) time.Time {
	n := time.Date(now.Year(), now.Month(), 1, 12, 0, 0, 0, time.UTC)
	return n.AddDate(0, delta, 0)
}

func insertTestCategory(t *testing.T) int {
	t.Helper()
	var id int
	err := db.QueryRow(
		`INSERT INTO categories (name, facets) VALUES ('Test', '[]')
		 RETURNING id`,
	).Scan(&id)
	if err != nil {
		t.Fatalf("insert category: %v", err)
	}
	return id
}

func insertDatedAd(t *testing.T, categoryID, userID int,
	created, expires time.Time) int {
	t.Helper()
	var id int
	err := db.QueryRow(
		`INSERT INTO ads (category_id, title, description, user_id,
		 expires_at, created_at)
		 VALUES ($1, 'title', 'desc', $2, $3, $4)
		 RETURNING id`,
		categoryID, userID, expires, created,
	).Scan(&id)
	if err != nil {
		t.Fatalf("insert ad: %v", err)
	}
	return id
}

func TestMonthlyFunStatsEmpty(t *testing.T) {
	resetFunStatsDB(t)

	rows, err := ad.MonthlyFunStats()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("empty site: got %d months, want 1", len(rows))
	}
	r := rows[0]
	if r.RegisteredUsers != 0 || r.UsersWithActiveAds != 0 ||
		r.ActiveAds != 0 {
		t.Fatalf("empty counts: %+v", r)
	}
}

func TestMonthlyFunStatsSnapshots(t *testing.T) {
	resetFunStatsDB(t)
	now := time.Now().UTC()
	catID := insertTestCategory(t)

	alice, err := user.CreateUser("alice", "+15559873001", "password1")
	if err != nil {
		t.Fatalf("alice: %v", err)
	}
	bob, err := user.CreateUser("bob", "+15559873002", "password1")
	if err != nil {
		t.Fatalf("bob: %v", err)
	}

	aliceAt := monthUTC(now, -3)
	bobAt := monthUTC(now, -1)
	ad1At := monthUTC(now, -2)
	ad2At := now.Add(-time.Minute)
	expires := now.AddDate(0, 6, 0)

	_, err = db.Exec(`UPDATE users SET created_at = $1 WHERE id = $2`,
		aliceAt, alice.ID)
	if err != nil {
		t.Fatalf("backdate alice: %v", err)
	}
	_, err = db.Exec(`UPDATE users SET created_at = $1 WHERE id = $2`,
		bobAt, bob.ID)
	if err != nil {
		t.Fatalf("backdate bob: %v", err)
	}

	ad1 := insertDatedAd(t, catID, alice.ID, ad1At, expires)
	insertDatedAd(t, catID, bob.ID, ad2At, expires)

	rows, err := ad.MonthlyFunStats()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 4 {
		t.Fatalf("got %d months, want 4", len(rows))
	}

	want := []struct {
		users, withAds, ads int
	}{
		{1, 0, 0},
		{1, 1, 1},
		{2, 1, 1},
		{2, 2, 2},
	}
	for i, w := range want {
		r := rows[i]
		if r.RegisteredUsers != w.users ||
			r.UsersWithActiveAds != w.withAds ||
			r.ActiveAds != w.ads {
			t.Errorf("month %d (%s): users=%d withAds=%d ads=%d, want %+v",
				i, r.Month.UTC().Format("2006-01"),
				r.RegisteredUsers, r.UsersWithActiveAds, r.ActiveAds, w)
		}
	}

	_, err = db.Exec(
		`UPDATE ads SET inactive_at =
		 CURRENT_TIMESTAMP - interval '1 minute'
		 WHERE id = $1`, ad1)
	if err != nil {
		t.Fatalf("inactivate ad1: %v", err)
	}

	rows, err = ad.MonthlyFunStats()
	if err != nil {
		t.Fatal(err)
	}
	last := rows[len(rows)-1]
	if last.ActiveAds != 1 || last.UsersWithActiveAds != 1 {
		t.Fatalf("after inactivate: ads=%d withAds=%d, want 1/1",
			last.ActiveAds, last.UsersWithActiveAds)
	}
	prev := rows[len(rows)-2]
	if prev.ActiveAds != 1 || prev.UsersWithActiveAds != 1 {
		t.Fatalf("month before inactivate: ads=%d withAds=%d, want 1/1",
			prev.ActiveAds, prev.UsersWithActiveAds)
	}
}
