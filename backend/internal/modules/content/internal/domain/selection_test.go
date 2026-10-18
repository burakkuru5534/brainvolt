package domain

import (
	"math/rand/v2"
	"testing"

	"github.com/google/uuid"
)

// pool builds cats×subs×perSub candidates with difficulties 1–5 cycling and
// every tfEvery-th one true/false (0 = none), shuffled with seed.
func pool(seed uint64, cats, subs, perSub, tfEvery int) ([]Candidate, []uuid.UUID) {
	var out []Candidate
	catIDs := make([]uuid.UUID, cats)
	n := 0
	for c := range cats {
		catIDs[c] = uuid.New()
		for range subs {
			sub := uuid.New()
			for range perSub {
				n++
				typ := MultipleChoice
				if tfEvery > 0 && n%tfEvery == 0 {
					typ = TrueFalse
				}
				out = append(out, Candidate{ID: uuid.New(), Type: typ, Difficulty: 1 + n%5, CategoryID: catIDs[c], SubcategoryID: sub})
			}
		}
	}
	r := rand.New(rand.NewPCG(seed, seed))
	r.Shuffle(len(out), func(i, j int) { out[i], out[j] = out[j], out[i] })
	return out, catIDs
}

func checkMix(t *testing.T, got []Candidate, r SelectRules) {
	t.Helper()
	if len(got) != len(r.Difficulties) {
		t.Fatalf("got %d questions, want %d", len(got), len(r.Difficulties))
	}
	seen := map[uuid.UUID]bool{}
	perCat := map[uuid.UUID]int{}
	tf := 0
	for i, c := range got {
		if seen[c.ID] {
			t.Fatalf("question %d repeated", i)
		}
		seen[c.ID] = true
		perCat[c.CategoryID]++
		if i > 0 && got[i-1].SubcategoryID == c.SubcategoryID {
			t.Fatalf("rounds %d and %d share a sub-category", i-1, i)
		}
		if c.Type == TrueFalse {
			tf++
			if i > 0 && got[i-1].Type == TrueFalse {
				t.Fatalf("true/false questions in a row at %d", i)
			}
		}
	}
	if tf > len(got)*r.MaxTrueFalsePercent/100 {
		t.Fatalf("%d true/false of %d exceeds %d%%", tf, len(got), r.MaxTrueFalsePercent)
	}
	if r.MaxPerCategory > 0 && r.HomeCategory == uuid.Nil {
		for _, n := range perCat {
			if n > r.MaxPerCategory {
				t.Fatalf("category cap %d exceeded: %d", r.MaxPerCategory, n)
			}
		}
	}
}

// MT06: the mix rules hold for every seed of a realistic pool.
func TestSelectMixRules(t *testing.T) {
	r := SelectRules{Difficulties: []int{2, 2, 3, 3, 3, 4, 4}, MaxPerCategory: 2, MaxTrueFalsePercent: 30}
	for seed := range uint64(200) {
		p, _ := pool(seed, 8, 3, 6, 3)
		got, ok := Select(p, r)
		if !ok {
			t.Fatalf("seed %d: no selection", seed)
		}
		checkMix(t, got, r)
		for i, c := range got {
			if c.Difficulty != r.Difficulties[i] {
				t.Fatalf("seed %d round %d: difficulty %d, want exact %d", seed, i, c.Difficulty, r.Difficulties[i])
			}
		}
	}
}

// MT07: a missing difficulty falls back to the nearest one, lower first.
func TestSelectDifficultyFallback(t *testing.T) {
	cat, sub1, sub2, sub3 := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	p := []Candidate{
		{ID: uuid.New(), Type: MultipleChoice, Difficulty: 5, CategoryID: cat, SubcategoryID: sub1},
		{ID: uuid.New(), Type: MultipleChoice, Difficulty: 3, CategoryID: cat, SubcategoryID: sub2},
		{ID: uuid.New(), Type: MultipleChoice, Difficulty: 1, CategoryID: cat, SubcategoryID: sub3},
	}
	got, ok := Select(p, SelectRules{Difficulties: []int{4, 4, 4}})
	if !ok {
		t.Fatal("no selection")
	}
	// 3 (lower by one) beats 5 (higher by one); then 5, then 1.
	want := []int{3, 5, 1}
	for i, c := range got {
		if c.Difficulty != want[i] {
			t.Fatalf("round %d: difficulty %d, want %d", i, c.Difficulty, want[i])
		}
	}
}

func TestSelectTrueFalseCap(t *testing.T) {
	// Only true/false questions: 30% of 10 allows 3, so 10 cannot be filled.
	p, _ := pool(1, 4, 4, 4, 1)
	if _, ok := Select(p, SelectRules{Difficulties: make([]int, 10), MaxTrueFalsePercent: 30}); ok {
		t.Fatal("selection of only true/false questions should fail")
	}
	// Multiple choice only (ranked): MaxTrueFalsePercent 0 never picks one.
	p, _ = pool(2, 4, 4, 4, 2)
	got, ok := Select(p, SelectRules{Difficulties: []int{3, 3, 3, 3, 3, 3, 3, 3, 3, 3}, MaxPerCategory: 3})
	if !ok {
		t.Fatal("no selection")
	}
	for _, c := range got {
		if c.Type == TrueFalse {
			t.Fatal("true/false selected with a 0% share")
		}
	}
}

func TestSelectNotEnough(t *testing.T) {
	p, _ := pool(3, 1, 3, 2, 0) // one category, 6 questions
	if _, ok := Select(p, SelectRules{Difficulties: []int{2, 2, 2}, MaxPerCategory: 2}); ok {
		t.Fatal("category cap should make 3 questions impossible")
	}
	if _, ok := Select(nil, SelectRules{Difficulties: []int{1}}); ok {
		t.Fatal("empty pool should fail")
	}
}

// Revenge Match: ⌊n/2⌋ questions from the home category on alternating
// rounds, never more than MaxRun of one category in a row.
func TestSelectRevengeHome(t *testing.T) {
	for _, n := range []int{7, 10} {
		for seed := range uint64(50) {
			p, cats := pool(seed, 4, 3, 6, 0)
			home := cats[0]
			r := SelectRules{Difficulties: make([]int, n), HomeCategory: home, MaxRun: 3, MaxTrueFalsePercent: 30}
			for i := range r.Difficulties {
				r.Difficulties[i] = 3
			}
			got, ok := Select(p, r)
			if !ok {
				t.Fatalf("n=%d seed %d: no selection", n, seed)
			}
			checkMix(t, got, r)
			homeCount, run := 0, 0
			for i, c := range got {
				if c.CategoryID == home {
					homeCount++
				}
				if i > 0 && got[i-1].CategoryID == c.CategoryID {
					run++
				} else {
					run = 1
				}
				if run > 3 {
					t.Fatalf("n=%d seed %d: run of %d", n, seed, run)
				}
			}
			if homeCount != n/2 {
				t.Fatalf("n=%d seed %d: %d home questions, want %d", n, seed, homeCount, n/2)
			}
		}
	}
}
