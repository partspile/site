package imagestore

import (
	"bytes"
	"testing"
)

func TestValidateReorderOrder(t *testing.T) {
	if err := ValidateReorderOrder([]int{2, 1, 3}, 3); err != nil {
		t.Fatal(err)
	}
	if err := ValidateReorderOrder([]int{1, 2}, 3); err == nil {
		t.Fatal("expected length error")
	}
	if err := ValidateReorderOrder([]int{1, 1, 2}, 3); err == nil {
		t.Fatal("expected duplicate error")
	}
	if err := ValidateReorderOrder([]int{0, 1, 2}, 3); err == nil {
		t.Fatal("expected range error")
	}
}

func TestReorderAdImages(t *testing.T) {
	dir := t.TempDir()
	s := NewLocal(dir)
	adID := 42

	for i := 1; i <= 3; i++ {
		for _, size := range ImageSizes {
			data := []byte{byte(i), size[0]}
			if err := s.Put(adID, i, size, data); err != nil {
				t.Fatal(err)
			}
		}
	}

	// New order: former 3,1,2 -> positions 1,2,3
	order := []int{3, 1, 2}
	if err := ReorderAdImages(s, adID, order); err != nil {
		t.Fatal(err)
	}

	wantAt := []byte{3, 1, 2}
	for pos, old := range wantAt {
		got, err := s.Get(adID, pos+1, "160w")
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, []byte{old, '1'}) {
			t.Fatalf("pos %d: got %v want marker %d", pos+1, got, old)
		}
	}

	for i := 4; i <= 6; i++ {
		ok, err := s.Stat(adID, i, "160w")
		if err != nil {
			t.Fatal(err)
		}
		if ok {
			t.Fatalf("temp index %d should be cleaned up", i)
		}
	}
}

func TestReorderAdImagesIdentityNoop(t *testing.T) {
	dir := t.TempDir()
	s := NewLocal(dir)
	if err := s.Put(1, 1, "160w", []byte{1}); err != nil {
		t.Fatal(err)
	}
	if err := ReorderAdImages(s, 1, []int{1}); err != nil {
		t.Fatal(err)
	}
}
