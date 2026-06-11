package render

import "testing"

func TestVisibleRangeSlidesWithOneItemAboveSelection(t *testing.T) {
	offset, visible := visibleRange(20, 10, 7)
	if offset != 9 || visible != 7 {
		t.Fatalf("visibleRange(20, 10, 7) = (%d, %d), want (9, 7)", offset, visible)
	}

	offset, visible = visibleRange(20, 9, 7)
	if offset != 8 || visible != 7 {
		t.Fatalf("visibleRange(20, 9, 7) = (%d, %d), want (8, 7)", offset, visible)
	}
}

func TestVisibleRangeClampsAtStartAndEnd(t *testing.T) {
	offset, visible := visibleRange(20, 0, 7)
	if offset != 0 || visible != 7 {
		t.Fatalf("visibleRange(20, 0, 7) = (%d, %d), want (0, 7)", offset, visible)
	}

	offset, visible = visibleRange(20, 19, 7)
	if offset != 13 || visible != 7 {
		t.Fatalf("visibleRange(20, 19, 7) = (%d, %d), want (13, 7)", offset, visible)
	}
}

func TestVisibleRangeHandlesShortLists(t *testing.T) {
	offset, visible := visibleRange(5, 4, 7)
	if offset != 0 || visible != 5 {
		t.Fatalf("visibleRange(5, 4, 7) = (%d, %d), want (0, 5)", offset, visible)
	}
}
