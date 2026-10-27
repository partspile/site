package handler

import (
	"errors"
	"fmt"

	"github.com/partspile/site/internal/config"
	"github.com/partspile/site/internal/service/sms"
	"github.com/partspile/site/internal/service/verify"
)

func verifyStartErrorMessage(err error) string {
	if err == nil {
		return "Unable to send verification code. Please try again."
	}
	if errors.Is(err, verify.ErrBlocked) || errors.Is(err, sms.ErrBlockedNumber) {
		return fmt.Sprintf(
			"This phone number cannot receive verification texts right now. "+
				"If you previously opted out, reply UNSTOP to %s from this "+
				"number, then try again.",
			config.TwilioFromNumber)
	}
	if errors.Is(err, verify.ErrRejected) {
		return "Unable to send a verification code to this number. " +
			"Please try a different US mobile number."
	}
	return "Unable to send verification code. Please try again."
}

func verifyCheckErrorMessage(err error) string {
	if errors.Is(err, verify.ErrInvalidCode) {
		return "Invalid or expired verification code. Please check your code and try again."
	}
	return "Invalid or expired verification code. Please request a new code."
}
