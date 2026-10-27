package handler

import (
	"errors"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/partspile/site/internal/ad"
	"github.com/partspile/site/internal/config"
	"github.com/partspile/site/internal/cookie"
	"github.com/partspile/site/internal/local"
	"github.com/partspile/site/internal/logger"
	"github.com/partspile/site/internal/message"
	"github.com/partspile/site/internal/otplimit"
	"github.com/partspile/site/internal/password"
	"github.com/partspile/site/internal/rock"
	"github.com/partspile/site/internal/service/turnstile"
	"github.com/partspile/site/internal/service/verify"
	"github.com/partspile/site/internal/ui"
	"github.com/partspile/site/internal/user"
	g "maragu.dev/gomponents"
)

const (
	memberSinceLayoutPage    = "January 2, 2006"
	memberSinceLayoutSummary = "Jan 2006"
)

func userProfileData(u user.User, activeAdCount, userRockCount int,
	tz *time.Location, memberSinceLayout string) ui.UserProfileData {
	return ui.UserProfileData{
		ID:                u.ID,
		Name:              u.Name,
		MemberSince:       u.CreatedAt.In(tz).Format(memberSinceLayout),
		ActiveAdCount:     activeAdCount,
		UserRockCount:     userRockCount,
		HasAccountPicture: u.HasAccountPicture,
		AccountPictureURL: u.AccountPictureURL,
	}
}

func loadUserMenuContext(userID int, tz *time.Location) (name, memberSince string,
	hasUnread bool, rockCount, userRockCount int, err error) {
	u, err := user.GetByID(userID)
	if err != nil {
		return "", "", false, 0, 0, err
	}
	hasUnread, _ = message.GetHasUnread(userID)
	rockCount, _ = rock.GetUserRockCount(userID)
	userRockCount, _ = rock.GetRockCountForUser(userID)
	memberSince = u.CreatedAt.In(tz).Format(memberSinceLayoutSummary)
	return u.Name, memberSince, hasUnread, rockCount, userRockCount, nil
}

func UserMenuHandler(c *fiber.Ctx) error {
	userID := local.GetUserID(c)
	tz := cookie.GetTimezone(c)
	name, memberSince, hasUnread, rockCount, userRockCount, err :=
		loadUserMenuContext(userID, tz)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to load user menu")
	}
	isAdmin := local.GetUserIsAdmin(c)
	return render(c, ui.UserMenu(name, memberSince, userID, isAdmin, hasUnread, rockCount, userRockCount))
}

func UserMyAdsHandler(c *fiber.Ctx) error {
	return userMyAdsTabHandler(c, "active", 0)
}

func UserMyAdsTabHandler(c *fiber.Ctx) error {
	tabID := c.Params("tab")
	if tabID != "bookmarked" && tabID != "active" &&
		tabID != "inactive" && tabID != "deleted" {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid tab")
	}
	return userMyAdsTabHandler(c, tabID, 0)
}

func UserMyAdsViewHandler(c *fiber.Ctx) error {
	tabID := c.Params("tab")
	if tabID != "bookmarked" && tabID != "active" &&
		tabID != "inactive" && tabID != "deleted" {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid tab")
	}
	view := ui.ValidateView(c.Params("view"))
	cookie.SetView(c, view)
	return userMyAdsTabHandler(c, tabID, view)
}

func userMyAdsTabHandler(c *fiber.Ctx, activeTab string, view int) error {
	userID := local.GetUserID(c)
	tz := cookie.GetTimezone(c)
	csrfToken := local.GetCSRFToken(c)
	if view == 0 {
		view = ui.ValidateView(cookie.GetView(c))
	}

	adIDs, err := ad.GetUserAdIDs(userID, activeTab)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to load ads")
	}

	ads, err := ad.GetAds(userID, adIDs, tz)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to load ads")
	}

	adNodes := ui.AdNodes(adCardsFrom(ads, userID, tz), userID, view, 1, csrfToken, false)

	if c.Get("HX-Request") != "" {
		return render(c, ui.MyAdsContainer(activeTab, view, adNodes))
	}

	return renderPage(c, "My Ads", ui.MyAdsPage(activeTab, view, adNodes))
}

func UserSettingsHandler(c *fiber.Ctx) error {
	userID := local.GetUserID(c)
	u, err := user.GetByID(userID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "User not found")
	}
	return renderPage(c, "Settings", ui.SettingsPage(
		u.Name, u.PhoneE64, u.SMSOptedOut,
		u.ID, u.HasAccountPicture, u.AccountPictureURL,
	))
}

func NotificationsToggleHandler(c *fiber.Ctx) error {
	userID := local.GetUserID(c)
	enabled := c.FormValue("enabled") == "true"
	if err := user.SetSMSOptOut(userID, !enabled); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to update notification preferences")
	}
	return render(c, ui.NotificationsSection(!enabled))
}

func ChangePasswordHandler(c *fiber.Ctx) error {
	userID := local.GetUserID(c)
	currentPassword := c.FormValue("current_password")
	newPassword := c.FormValue("new_password")
	confirmPassword := c.FormValue("confirm_password")

	if err := password.ValidatePasswordChange(currentPassword, newPassword, confirmPassword); err != nil {
		return showErrorTo(c, ui.SettingsChangePasswordErrorID, err.Error())
	}

	u, err := user.GetByID(userID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "User not found")
	}

	if !password.VerifyPassword(currentPassword, u.PasswordHash, u.PasswordSalt) {
		return showErrorTo(c, ui.SettingsChangePasswordErrorID, "Invalid current password")
	}

	newHash, newSalt, err := password.HashPassword(newPassword)
	if err != nil {
		logger.Error("Failed to hash password", "error", err)
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to change password")
	}

	if err := user.UpdatePassword(userID, newHash, newSalt, "argon2id"); err != nil {
		logger.Error("Failed to update password", "error", err)
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to change password")
	}

	logout(c)
	c.Set("HX-Redirect", "/login")
	return c.SendStatus(fiber.StatusOK)
}

func ChangePhoneRequestHandler(c *fiber.Ctx) error {
	userID := local.GetUserID(c)
	currentPassword := c.FormValue("current_password")
	phone := c.FormValue("phone")

	if currentPassword == "" {
		return showErrorTo(c, ui.SettingsChangePhoneErrorID,
			"Current password is required")
	}

	u, err := user.GetByID(userID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "User not found")
	}

	if !password.VerifyPassword(currentPassword, u.PasswordHash, u.PasswordSalt) {
		return showErrorTo(c, ui.SettingsChangePhoneErrorID, "Invalid current password")
	}

	phoneE64, err := validatePhone(phone)
	if err != nil {
		return showErrorTo(c, ui.SettingsChangePhoneErrorID, err.Error())
	}

	if phoneE64 == u.PhoneE64 {
		return showErrorTo(c, ui.SettingsChangePhoneErrorID,
			"That is already your current phone number")
	}

	avail, err := user.CheckPhoneAvailability(phoneE64, userID)
	if err != nil {
		logger.Error("Failed to check phone availability", "error", err)
		return fiber.NewError(fiber.StatusInternalServerError,
			"Failed to check phone number")
	}
	switch avail.Status {
	case user.PhoneActive:
		return showErrorTo(c, ui.SettingsChangePhoneErrorID,
			"That phone number is already in use")
	case user.PhoneHeld:
		return showErrorTo(c, ui.SettingsChangePhoneErrorID,
			(&user.PhoneHoldError{DaysRemaining: avail.DaysRemaining}).Error())
	}

	if allowTestRegistration(phoneE64) {
		return render(c, ui.ChangePhoneVerifySection(phoneE64))
	}

	ip := VisitorIP(c)
	if err := turnstile.Verify(c.FormValue("cf-turnstile-response"), ip); err != nil {
		logger.Warn("Turnstile failed on change-phone", "error", err, "ip", ip)
		return showErrorTo(c, ui.SettingsChangePhoneErrorID,
			"Please complete the verification challenge and try again.")
	}

	if err := otplimit.AllowStart(phoneE64); err != nil {
		return showErrorTo(c, ui.SettingsChangePhoneErrorID, err.Error())
	}

	if err := verify.StartSMS(phoneE64, verify.PurposeChangePhone); err != nil {
		logger.Error("Failed to start Verify SMS for change-phone",
			"error", err, "phone", phoneE64)
		return showErrorTo(c, ui.SettingsChangePhoneErrorID,
			verifyStartErrorMessage(err))
	}

	return render(c, ui.ChangePhoneVerifySection(phoneE64))
}

func ChangePhoneVerifyHandler(c *fiber.Ctx) error {
	userID := local.GetUserID(c)
	phone := c.FormValue("phone")
	code := c.FormValue("code")

	phoneE64, err := validatePhone(phone)
	if err != nil {
		return showErrorTo(c, ui.SettingsChangePhoneErrorID, err.Error())
	}

	if code == "" {
		return showErrorTo(c, ui.SettingsChangePhoneErrorID,
			"Please enter the verification code")
	}

	u, err := user.GetByID(userID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "User not found")
	}

	if !allowTestRegistration(phoneE64) {
		if err := verify.Check(phoneE64, code); err != nil {
			logger.Warn("Change-phone Verify check failed",
				"error", err, "phone", phoneE64)
			return showErrorTo(c, ui.SettingsChangePhoneErrorID,
				verifyCheckErrorMessage(err))
		}
	}

	if err := user.UpdatePhone(userID, phoneE64); err != nil {
		var holdErr *user.PhoneHoldError
		if errors.As(err, &holdErr) {
			return showErrorTo(c, ui.SettingsChangePhoneErrorID, holdErr.Error())
		}
		if errors.Is(err, user.ErrPhoneSame) {
			return showErrorTo(c, ui.SettingsChangePhoneErrorID,
				"That is already your current phone number")
		}
		if errors.Is(err, user.ErrUserAlreadyExists) {
			return showErrorTo(c, ui.SettingsChangePhoneErrorID,
				"That phone number is already in use")
		}
		logger.Error("Failed to update phone", "error", err, "userID", userID)
		return fiber.NewError(fiber.StatusInternalServerError,
			"Failed to update phone number")
	}

	cookie.SetDistanceUnitForUser(c, phoneE64)
	return render(c, ui.ChangePhoneSuccess(u.Name, phoneE64))
}

func DeleteAccountConfirmModalHandler(c *fiber.Ctx) error {
	csrfToken := local.GetCSRFToken(c)
	return render(c, ui.DeleteAccountConfirmModal(csrfToken))
}

func DeleteAccountHandler(c *fiber.Ctx) error {
	userID := local.GetUserID(c)
	passwd := c.FormValue("password")
	if passwd == "" {
		return showErrorTo(c, ui.SettingsDeleteAccountErrorID, "Password is required")
	}

	u, err := user.GetByID(userID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "User not found")
	}

	if !password.VerifyPassword(passwd, u.PasswordHash, u.PasswordSalt) {
		return showErrorTo(c, ui.SettingsDeleteAccountErrorID, "Invalid password")
	}

	if err := user.DeleteUser(userID); err != nil {
		logger.Error("Failed to delete account", "error", err, "userID", userID)
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to delete account")
	}
	deleteUserAccountPicture(userID)

	convs, err := message.CloseConversationsForDeletedAccount(userID)
	if err != nil {
		logger.Error("Failed to close conversations for deleted account",
			"error", err, "userID", userID)
	} else {
		NotifyConversationsClosed(convs, userID)
	}

	logout(c)
	c.Set("HX-Redirect", "/")
	return c.SendStatus(fiber.StatusOK)
}

func AboutHandler(c *fiber.Ctx) error {
	return renderPage(c, "About", ui.AboutPage())
}

func DonateHandler(c *fiber.Ctx) error {
	return renderPage(c, "Donate", ui.DonatePage())
}

func FunStatsHandler(c *fiber.Ctx) error {
	rows, err := ad.MonthlyFunStats()
	if err != nil {
		return err
	}
	months := make([]ui.FunStatsMonth, len(rows))
	for i, row := range rows {
		months[i] = ui.FunStatsMonth{
			Label:              row.Month.UTC().Format("Jan 2006"),
			RegisteredUsers:    row.RegisteredUsers,
			UsersWithActiveAds: row.UsersWithActiveAds,
			ActiveAds:          row.ActiveAds,
		}
	}
	return renderPage(c, "Fun Stats",
		ui.FunStatsPage(ui.FunStatsData{Months: months}))
}

func PrivacyHandler(c *fiber.Ctx) error {
	return renderPage(c, "Privacy Policy", ui.PrivacyPolicyPage())
}

func TermsHandler(c *fiber.Ctx) error {
	return renderPage(c, "Terms of Service", ui.TermsOfServicePage())
}

func FAQHandler(c *fiber.Ctx) error {
	section := c.Params("section")
	if section != "" && !ui.ValidFAQSection(section) {
		return fiber.NewError(fiber.StatusNotFound)
	}
	return renderPage(c, "FAQ", ui.FAQPage(section))
}

func UserProfileHandler(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid user ID")
	}
	prefix := fmt.Sprintf("/auth/user/%d/view/", id)
	showShare := local.GetUserID(c) == id
	return renderUserProfile(c, id, prefix, showShare)
}

func SharedUserProfileHandler(c *fiber.Ctx) error {
	id, err := sharedProfileUserID(c)
	if err != nil {
		return err
	}
	token := c.Params("token")
	prefix := "/u/" + token + "/view/"
	showShare := local.GetUserID(c) == id
	return renderUserProfile(c, id, prefix, showShare)
}

func renderUserProfile(c *fiber.Ctx, id int, adsViewPrefix string,
	showShare bool) error {
	u, err := user.GetByID(id)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "User not found")
	}
	activeAdCount, _ := ad.CountActiveAdsByUser(id)
	userRockCount, _ := rock.GetRockCountForUser(id)
	tz := cookie.GetTimezone(c)
	view := ui.ValidateView(cookie.GetView(c))
	adNodes, err := userActiveAdNodes(c, id, view)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to load ads")
	}
	d := userProfileData(u, activeAdCount, userRockCount, tz, memberSinceLayoutPage)
	d.ShowShare = showShare
	d.AdsViewPathPrefix = adsViewPrefix
	return renderPage(c, u.Name, ui.UserProfilePage(d, view, adNodes))
}

func UserProfileViewHandler(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid user ID")
	}
	if _, err := user.GetByID(id); err != nil {
		return fiber.NewError(fiber.StatusNotFound, "User not found")
	}
	prefix := fmt.Sprintf("/auth/user/%d/view/", id)
	return renderUserProfileAds(c, id, prefix)
}

func SharedUserProfileViewHandler(c *fiber.Ctx) error {
	id, err := sharedProfileUserID(c)
	if err != nil {
		return err
	}
	prefix := "/u/" + c.Params("token") + "/view/"
	return renderUserProfileAds(c, id, prefix)
}

func renderUserProfileAds(c *fiber.Ctx, id int, adsViewPrefix string) error {
	view := ui.ValidateView(c.Params("view"))
	cookie.SetView(c, view)
	adNodes, err := userActiveAdNodes(c, id, view)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to load ads")
	}
	return render(c, ui.UserProfileAds(adsViewPrefix, view, adNodes))
}

func sharedProfileUserID(c *fiber.Ctx) (int, error) {
	id, err := user.GetIDByShareToken(c.Params("token"))
	if err != nil {
		return 0, fiber.NewError(fiber.StatusNotFound, "User not found")
	}
	return id, nil
}

func UserShareHandler(c *fiber.Ctx) error {
	token, err := user.ShareToken(local.GetUserID(c))
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError,
			"Failed to create share link")
	}
	return render(c, ui.UserShareModal(config.CanonicalURL("/u/"+token)))
}

func userActiveAdNodes(c *fiber.Ctx, profileUserID, view int) ([]g.Node, error) {
	viewerID := local.GetUserID(c)
	tz := cookie.GetTimezone(c)
	csrfToken := local.GetCSRFToken(c)
	adIDs, err := ad.GetUserAdIDs(profileUserID, "active")
	if err != nil {
		return nil, err
	}
	ads, err := ad.GetAds(viewerID, adIDs, tz)
	if err != nil {
		return nil, err
	}
	return ui.AdNodes(adCardsFrom(ads, viewerID, tz), viewerID, view, 1,
		csrfToken, false), nil
}

func UserSummaryHandler(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid user ID")
	}
	u, err := user.GetByID(id)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "User not found")
	}
	activeAdCount, _ := ad.CountActiveAdsByUser(id)
	userRockCount, _ := rock.GetRockCountForUser(id)
	tz := cookie.GetTimezone(c)
	d := userProfileData(u, activeAdCount, userRockCount, tz, memberSinceLayoutSummary)
	return render(c, ui.UserSummaryFragment(d))
}

func UserRockConversationHandler(c *fiber.Ctx) error {
	userID, err := c.ParamsInt("id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid user ID")
	}

	ordinal, err := c.ParamsInt("ordinal")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid rock ordinal")
	}

	currentUserID := local.GetUserID(c)
	tz := cookie.GetTimezone(c)
	csrfToken := local.GetCSRFToken(c)

	conversationID, err := rock.GetConversationIDForUserRockByOrdinal(userID, ordinal)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Rock conversation not found")
	}

	conv, err := message.GetConversationByID(conversationID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Conversation not found")
	}
	if conv.RockThrowerID == nil {
		return fiber.NewError(fiber.StatusNotFound, "No dispute assessment")
	}

	if message.IsParticipant(conv, currentUserID) {
		return renderConversationModal(c, conv, currentUserID, tz, csrfToken)
	}
	return renderRockOpinionModal(c, conv, currentUserID, tz)
}
