package ui

import (
	"bytes"
	"strings"
	"testing"

	g "maragu.dev/gomponents"
)

func TestUserProfilePageShareButton(t *testing.T) {
	d := UserProfileData{
		ID:                1,
		Name:              "test",
		MemberSince:       "January 2, 2006",
		AdsViewPathPrefix: "/auth/user/1/view/",
		ShowShare:         true,
		IsOwn:             true,
	}
	var buf bytes.Buffer
	if err := g.Group(UserProfilePage(d, ViewList, nil)).Render(&buf); err != nil {
		t.Fatal(err)
	}
	html := buf.String()
	if !strings.Contains(html, "/auth/user/share") {
		t.Fatal("missing share button")
	}
	if !strings.Contains(html, "/auth/user/1/view/") {
		t.Fatal("missing auth view prefix")
	}
	if !strings.Contains(html, "/auth/user/profile/picture/edit") {
		t.Fatal("own profile should offer picture edit")
	}
	if !strings.Contains(html, "/images/edit.svg") {
		t.Fatal("own profile should show edit icon on picture")
	}
	if !strings.Contains(html, "border-dashed") {
		t.Fatal("empty own profile should show dotted picture placeholder")
	}
	if strings.Contains(html, "Link: ") {
		t.Fatal("should not show link text under picture")
	}
	if !strings.Contains(html, "/js/account-picture.js") {
		t.Fatal("own profile should load account-picture.js")
	}

	d.ShowShare = false
	d.IsOwn = false
	d.AdsViewPathPrefix = "/u/tokentokentokentoke/view/"
	buf.Reset()
	if err := g.Group(UserProfilePage(d, ViewList, nil)).Render(&buf); err != nil {
		t.Fatal(err)
	}
	html = buf.String()
	if strings.Contains(html, "/auth/user/share") {
		t.Fatal("guest page should not share")
	}
	if strings.Contains(html, "/auth/user/profile/picture") {
		t.Fatal("guest page should not edit picture")
	}
	if !strings.Contains(html, "/u/tokentokentokentoke/view/") {
		t.Fatal("missing shared view prefix")
	}
}

func TestAccountPictureEditSectionPaths(t *testing.T) {
	d := UserProfileData{
		ID:                1,
		Name:              "test",
		HasAccountPicture: true,
		AccountPictureURL: "https://example.com",
		IsOwn:             true,
	}
	var buf bytes.Buffer
	if err := AccountPictureEditSection(d, "").Render(&buf); err != nil {
		t.Fatal(err)
	}
	html := buf.String()
	for _, want := range []string{
		`id="account-picture-section"`,
		"/auth/user/profile/picture",
		"/auth/user/profile/picture/remove",
		`id="account-picture-upload-btn"`,
		`id="account-picture-file"`,
		`class="hidden"`,
		"Change picture",
		"Link when picture is clicked",
		"https://example.com",
		`change delay:400ms, blur`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("edit section missing %q", want)
		}
	}
	for _, ban := range []string{
		"Replace picture", "Save link", ">Done<", "Done</",
		"/auth/user/settings/account-picture",
	} {
		if strings.Contains(html, ban) {
			t.Errorf("edit section should not include %q", ban)
		}
	}
}
