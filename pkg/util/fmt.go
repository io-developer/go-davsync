package util

import "fmt"

func FormatBytes(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%d B", size)
	}
	rest := size
	mul := uint64(1)
	exp := uint64(0)
	for (rest >> 10) > 0 {
		rest = rest >> 10
		mul = mul << 10
		exp++
	}
	val := float64(size) / float64(mul)
	suffixes := []string{"B", "KiB", "MiB", "GiB", "TiB", "PiB", "EiB"}
	return fmt.Sprintf("%.1f %s", val, suffixes[exp])
}
