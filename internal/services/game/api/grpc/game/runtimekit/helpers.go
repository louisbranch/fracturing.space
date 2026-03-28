package runtimekit

import (
	"strconv"
	"time"
)

// FixedClock returns a deterministic clock function for tests.
func FixedClock(t time.Time) func() time.Time {
	return func() time.Time {
		return t
	}
}

// FixedIDGenerator returns an ID generator that always yields the same ID.
func FixedIDGenerator(id string) func() (string, error) {
	return func() (string, error) {
		return id, nil
	}
}

// FixedSequenceIDGenerator returns IDs in order and then repeats the last ID.
func FixedSequenceIDGenerator(ids ...string) func() (string, error) {
	if len(ids) == 0 {
		panic("runtimekit.FixedSequenceIDGenerator requires at least one id")
	}
	index := 0
	return func() (string, error) {
		if index >= len(ids) {
			return ids[len(ids)-1], nil
		}
		id := ids[index]
		index++
		return id, nil
	}
}

// SequentialIDGenerator returns IDs with an incrementing numeric suffix.
func SequentialIDGenerator(prefix string) func() (string, error) {
	counter := 0
	return func() (string, error) {
		counter++
		return prefix + "-" + strconv.Itoa(counter), nil
	}
}
