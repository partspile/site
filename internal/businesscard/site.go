package businesscard

import "fmt"

// SiteURL returns the homepage URL used on printed QR codes.
func SiteURL(host string) string {
	return fmt.Sprintf("https://%s/", host)
}
