package imagestore

import "fmt"

// IdentityOrder reports whether order is 1..n in ascending sequence.
func IdentityOrder(order []int) bool {
	for i, v := range order {
		if v != i+1 {
			return false
		}
	}
	return true
}

// ValidateReorderOrder checks that order is a permutation of 1..n.
func ValidateReorderOrder(order []int, n int) error {
	if n < 0 {
		return fmt.Errorf("invalid image count")
	}
	if len(order) != n {
		return fmt.Errorf("image order must include all %d images", n)
	}
	seen := make([]bool, n+1)
	for _, idx := range order {
		if idx < 1 || idx > n || seen[idx] {
			return fmt.Errorf("invalid image order")
		}
		seen[idx] = true
	}
	return nil
}

// ReorderAdImages remaps stored image files so position i holds former
// order[i-1]. Uses temporary high indexes to avoid overwrite collisions.
func ReorderAdImages(store Store, adID int, order []int) error {
	n := len(order)
	if n == 0 || IdentityOrder(order) {
		return nil
	}
	if err := ValidateReorderOrder(order, n); err != nil {
		return err
	}

	tempBase := n + 1
	for i, from := range order {
		toTemp := tempBase + i
		for _, size := range ImageSizes {
			if err := store.Copy(adID, from, toTemp, size); err != nil {
				return fmt.Errorf(
					"copy image %d to temp %d (%s): %w",
					from, toTemp, size, err,
				)
			}
		}
	}

	for i := 1; i <= n; i++ {
		for _, size := range ImageSizes {
			if err := store.Delete(adID, i, size); err != nil {
				return fmt.Errorf(
					"delete image %d (%s): %w", i, size, err,
				)
			}
		}
	}

	for i := 0; i < n; i++ {
		fromTemp := tempBase + i
		to := i + 1
		for _, size := range ImageSizes {
			if err := store.Copy(adID, fromTemp, to, size); err != nil {
				return fmt.Errorf(
					"copy temp %d to image %d (%s): %w",
					fromTemp, to, size, err,
				)
			}
		}
	}

	for i := 0; i < n; i++ {
		fromTemp := tempBase + i
		for _, size := range ImageSizes {
			if err := store.Delete(adID, fromTemp, size); err != nil {
				return fmt.Errorf(
					"delete temp %d (%s): %w", fromTemp, size, err,
				)
			}
		}
	}
	return nil
}
