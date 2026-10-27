package businesscard

import (
	"testing"

	"github.com/skip2/go-qrcode"
)

func TestQRModuleCounts(t *testing.T) {
	cats, err := LoadCategories("")
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range cats {
		url := CategoryURL("parts-pile.com", c.ID)
		for _, level := range []qrcode.RecoveryLevel{qrcode.Low, qrcode.Medium} {
			code, err := qrcode.New(url, level)
			if err != nil {
				t.Fatal(err)
			}
			n := len(code.Bitmap())
			t.Logf("%s level=%d modules=%dx%d", c.Name, level, n, n)
		}
	}
}
