package db

import "encoding/binary"

// itob returns an 8-byte big-endian representation of an uint64.
func itob(i uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, i)
	return b
}

// sitosb returns a slice of 8-byte big-endian representation of the slice of uint64.
func sitosb(si []uint64) [][]byte {
	sb := make([][]byte, len(si))
	for i, v := range si {
		b := itob(v)
		sb[i] = make([]byte, len(b))
		_ = copy(sb[i], b)
	}
	return sb
}

// 8-byte big-endian returns an uint64 representation of an 8-byte big-endian.
func btoi(b []byte) uint64 {
	return binary.BigEndian.Uint64(b)
}

// If found, removes i in the slice and returns the slice with no error.
func remove(s []uint64, i uint64) ([]uint64, error) {
	for k, v := range s {
		if v == i {
			return append(s[:k], s[k+1:]...), nil
		}
	}
	return s, ErrNotFound
}
