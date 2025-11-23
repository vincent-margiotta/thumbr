package ui

import (
	"math/rand"
	"slices"
	"testing"
)

func sampleRandomSequence(m Model, count int) ([]int, Model) {
	seq := make([]int, 0, count)
	for i := 0; i < count; i++ {
		m = m.randomCursor()
		seq = append(seq, m.cursor)
	}
	return seq, m
}

func TestRandomCursor_UsesInjectedRNG(t *testing.T) {
	const (
		totalCards = 8
		seed       = 42
		draws      = 5
	)

	m1 := newTestModel(totalCards, 0)
	m1.rng = rand.New(rand.NewSource(seed))
	seq1, m1 := sampleRandomSequence(m1, draws)

	m2 := newTestModel(totalCards, 0)
	m2.rng = rand.New(rand.NewSource(seed))
	seq2, _ := sampleRandomSequence(m2, draws)

	if !slices.Equal(seq1, seq2) {
		t.Fatalf("expected deterministic random cursor sequence with same seed, got %v vs %v", seq1, seq2)
	}

	// With a different seed, sequences should differ.
	m3 := newTestModel(totalCards, 0)
	m3.rng = rand.New(rand.NewSource(seed + 1))
	seq3, _ := sampleRandomSequence(m3, draws)

	if slices.Equal(seq1, seq3) {
		t.Fatalf("expected different seed to produce different sequence, but both were %v", seq1)
	}
}
