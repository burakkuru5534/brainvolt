package service

import (
	"testing"

	"github.com/google/uuid"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/game"
)

// D-093: a smaller format moves members whose seat disappeared to free
// seats; the others keep their place.
func TestRoomReseat(t *testing.T) {
	a, b, c := uuid.New(), uuid.New(), uuid.New()
	r := &room{
		Config: game.Config{Format: "2v2"},
		Members: []member{
			{UserID: a, Side: 1, Seat: 1},
			{UserID: b, Side: 1, Seat: 2},
		},
	}
	r.Config.Format = "1v1"
	r.reseat()
	if m := r.member(a); m.Side != 1 || m.Seat != 1 {
		t.Fatalf("host moved: %+v", m)
	}
	if m := r.member(b); m.Side != 2 || m.Seat != 1 {
		t.Fatalf("second player: %+v", m)
	}

	r = &room{
		Config: game.Config{Format: "3v3"},
		Members: []member{
			{UserID: a, Side: 1, Seat: 1},
			{UserID: b, Side: 2, Seat: 3},
			{UserID: c, Side: 1, Seat: 3},
		},
	}
	r.Config.Format = "2v2"
	r.reseat()
	seen := map[[2]int]bool{}
	for _, m := range r.Members {
		if m.Seat < 1 || m.Seat > 2 || m.Side < 1 || m.Side > 2 || seen[[2]int{m.Side, m.Seat}] {
			t.Fatalf("bad seat %+v in %+v", m, r.Members)
		}
		seen[[2]int{m.Side, m.Seat}] = true
	}
	if m := r.member(a); m.Side != 1 || m.Seat != 1 {
		t.Fatalf("a moved: %+v", m)
	}
}
