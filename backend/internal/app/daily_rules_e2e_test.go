package app

import (
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/game"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/identity"
)

// Daily Challenge rules of category-modes §9: board tie-breakers,
// restricted accounts off the public board until they upgrade (D-016),
// and one official start among concurrent requests.
func TestDailyBoardRulesAndConcurrentStart(t *testing.T) {
	x := newE2E(t)
	tok1, u1, _ := x.fullUser()
	_, u2, _ := x.fullUser()
	_, u3, _ := x.fullUser()
	_, u4, _ := x.fullUser()
	_, guest := x.signedIn()
	x.seedQuestions(u1)
	day := game.DailyDay(time.Now()).Format(time.DateOnly)
	base := time.Now().UTC().Add(-time.Hour)
	run := func(u uuid.UUID, correct, ms int, after time.Duration, tier string) {
		x.deliver(x.mods.ranking.OnDailyRunCompleted, game.EventDailyRunCompleted, game.DailyRunCompleted{
			RunID: uuid.New(), UserID: u, Day: day, CorrectCount: correct, TotalTimeMS: ms, CompletedAt: base.Add(after), TierAtStart: tier,
		})
	}
	run(u1, 8, 40_000, 2*time.Minute, "full") // ties u2 on correct and time, finished later
	run(u2, 8, 40_000, time.Minute, "full")
	run(u3, 9, 140_000, 0, "full") // more correct beats any time
	run(u4, 8, 55_000, 0, "full")
	run(guest, 10, 10_000, 0, "guest") // restricted: not public yet

	board := func() []string {
		r := x.do(http.MethodGet, "/v1/leaderboards/daily-challenge/"+day, tok1, nil)
		if r.code != 200 {
			t.Fatalf("board %s", r.raw)
		}
		var out []string
		for _, e := range r.get("entries").([]any) {
			out = append(out, str(e.(map[string]any)["user_id"]))
		}
		return out
	}
	order := []string{u3.String(), u2.String(), u1.String(), u4.String()}
	if got := board(); len(got) != 4 || got[0] != order[0] || got[1] != order[1] || got[2] != order[2] || got[3] != order[3] {
		t.Fatalf("board order %v, want %v", got, order)
	}
	x.deliver(x.mods.ranking.OnTierChanged, identity.EventAccountTierChanged, map[string]any{"user_id": guest, "from": "guest", "to": "full"})
	if got := board(); len(got) != 5 || got[0] != guest.String() {
		t.Fatalf("after upgrade %v", got)
	}

	// Concurrent official starts: exactly one wins.
	tokC, _, _ := x.fullUser()
	var wg sync.WaitGroup
	codes := make([]int, 5)
	for i := range codes {
		wg.Add(1)
		go func() {
			defer wg.Done()
			codes[i] = x.post("/v1/daily/runs", tokC, map[string]any{"kind": "official"}).code
		}()
	}
	wg.Wait()
	created := 0
	for _, c := range codes {
		switch c {
		case 201:
			created++
		case 409:
		default:
			t.Fatalf("codes %v", codes)
		}
	}
	if created != 1 {
		t.Fatalf("codes %v", codes)
	}
}
