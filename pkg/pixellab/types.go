package pixellab

import (
	"iter"
	"net/url"
)

// All returns all urls in the order PixelLab's own exports list them: south
// first, then round through east, north and west (south, south-east, east,
// north-east, north, north-west, west, south-west).
// 4-directional characters yield 4 entries; 8-directional characters yield 8.
func (u RotationURLs) All() iter.Seq2[Direction, url.URL] {
	return func(yield func(Direction, url.URL) bool) {
		if !yield(DirectionSouth, u.South) {
			return
		}

		if u.SouthEast != (url.URL{}) {
			if !yield(DirectionSouthEast, u.SouthEast) {
				return
			}
		}

		if !yield(DirectionEast, u.East) {
			return
		}

		if u.NorthEast != (url.URL{}) {
			if !yield(DirectionNorthEast, u.NorthEast) {
				return
			}
		}

		if !yield(DirectionNorth, u.North) {
			return
		}

		if u.NorthWest != (url.URL{}) {
			if !yield(DirectionNorthWest, u.NorthWest) {
				return
			}
		}

		if !yield(DirectionWest, u.West) {
			return
		}

		if u.SouthWest != (url.URL{}) {
			if !yield(DirectionSouthWest, u.SouthWest) {
				return
			}
		}
	}
}

// Length returns the number of rotation URLs present (4 or 8).
func (u RotationURLs) Length() int {
	n := 4

	if u.NorthEast != (url.URL{}) {
		n++
	}

	if u.SouthEast != (url.URL{}) {
		n++
	}

	if u.SouthWest != (url.URL{}) {
		n++
	}

	if u.NorthWest != (url.URL{}) {
		n++
	}

	return n
}

// IsEightDirectional reports whether diagonal rotations are present.
func (u RotationURLs) IsEightDirectional() bool {
	return u.NorthEast != (url.URL{}) ||
		u.SouthEast != (url.URL{}) ||
		u.SouthWest != (url.URL{}) ||
		u.NorthWest != (url.URL{})
}

// Get returns the URL for the given direction. The second return value is
// false if dir is DirectionNone, unknown, or unset (diagonals on a 4-dir set).
func (u RotationURLs) Get(dir Direction) (url.URL, bool) {
	switch dir {
	case DirectionNorth:
		return u.North, true
	case DirectionNorthEast:
		return u.NorthEast, u.NorthEast != (url.URL{})
	case DirectionEast:
		return u.East, true
	case DirectionSouthEast:
		return u.SouthEast, u.SouthEast != (url.URL{})
	case DirectionSouth:
		return u.South, true
	case DirectionSouthWest:
		return u.SouthWest, u.SouthWest != (url.URL{})
	case DirectionWest:
		return u.West, true
	case DirectionNorthWest:
		return u.NorthWest, u.NorthWest != (url.URL{})
	default:
		return url.URL{}, false
	}
}

// Keys returns just the directions in clockwise order.
func (u RotationURLs) Keys() iter.Seq[Direction] {
	return func(yield func(Direction) bool) {
		for d := range u.All() {
			if !yield(d) {
				return
			}
		}
	}
}

// Values returns just the URLs in clockwise order.
func (u RotationURLs) Values() iter.Seq[url.URL] {
	return func(yield func(url.URL) bool) {
		for _, v := range u.All() {
			if !yield(v) {
				return
			}
		}
	}
}
