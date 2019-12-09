package main

import "strings"

import "strconv"

import "time"

type suffixMultiplier struct {
	suffix     string
	multiplier float64
}

var sizeSuffixes []suffixMultiplier = []suffixMultiplier{
	{"kib", 1024}, {"kb", 1024}, {"ki", 1024}, {"k", 1024},
	{"mib", 1024 * 1024}, {"mb", 1024 * 1024}, {"mi", 1024 * 1024}, {"m", 1024 * 1024},
	{"gib", 1024 * 1024 * 1024}, {"gb", 1024 * 1024 * 1024}, {"gi", 1024 * 1024 * 1024}, {"g", 1024 * 1024 * 1024},
	{"tib", 1024 * 1024 * 1024 * 1024}, {"tb", 1024 * 1024 * 1024 * 1024}, {"ti", 1024 * 1024 * 1024 * 1024}, {"t", 1024 * 1024 * 1024 * 1024},
	{"pib", 1024 * 1024 * 1024 * 1024 * 1024}, {"pb", 1024 * 1024 * 1024 * 1024 * 1024}, {"pi", 1024 * 1024 * 1024 * 1024 * 1024}, {"p", 1024 * 1024 * 1024 * 1024 * 1024},
}

var timeSuffixes []suffixMultiplier = []suffixMultiplier{
	{"mks", float64(time.Microsecond)}, {"microsecond", float64(time.Microsecond)}, {"microseconds", float64(time.Microsecond)},
	{"ms", float64(time.Millisecond)}, {"millisecond", float64(time.Millisecond)}, {"milliseconds", float64(time.Millisecond)},
	{"s", float64(time.Second)}, {"second", float64(time.Second)}, {"seconds", float64(time.Second)},
	{"m", float64(time.Minute)}, {"minute", float64(time.Minute)}, {"minutes", float64(time.Minute)},
	{"h", float64(time.Hour)}, {"hour", float64(time.Hour)}, {"hours", float64(time.Hour)},
	{"d", float64(24 * time.Hour)}, {"day", float64(24 * time.Hour)}, {"days", float64(24 * time.Hour)},
}

func parseSuffixed(value string, suffixes []suffixMultiplier) (int64, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, nil
	}
	value = strings.ToLower(value)

	var multiplier float64 = 1
	for _, v := range suffixes {
		if strings.HasSuffix(value, v.suffix) {
			value = strings.TrimSpace(value[0 : len(value)-len(v.suffix)])
			multiplier = v.multiplier
			break
		}
	}
	v, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, err
	}
	return int64(v * multiplier), nil
}

func parseSizeString(size string) (int64, error) {
	return parseSuffixed(size, sizeSuffixes)
}

func parseTimeDuration(duration string) (time.Duration, error) {
	v, err := parseSuffixed(duration, timeSuffixes)
	return time.Duration(v), err
}
