package domain

import (
	"math"
	"testing"
)

// Glickman's worked example: μ 1500, RD 200, σ 0.06 against three
// opponents → μ' ≈ 1464.06, RD' ≈ 151.52, σ' ≈ 0.05999.
func TestGlickmanExample(t *testing.T) {
	r := Rating{Value: 1500, RD: 200, Volatility: 0.06}.Update([]Opponent{
		{Rating: Rating{Value: 1400, RD: 30}, Score: 1},
		{Rating: Rating{Value: 1550, RD: 100}, Score: 0},
		{Rating: Rating{Value: 1700, RD: 300}, Score: 0},
	})
	near := func(name string, got, want, tol float64) {
		if math.Abs(got-want) > tol {
			t.Errorf("%s = %.5f, want %.5f", name, got, want)
		}
	}
	near("rating", r.Value, 1464.06, 0.01)
	near("rd", r.RD, 151.52, 0.01)
	near("volatility", r.Volatility, 0.05999, 0.00001)
}

func TestRDBoundsAndIdle(t *testing.T) {
	r := Rating{Value: 1500, RD: 50, Volatility: 0.06}
	for range 50 {
		r = r.Update([]Opponent{{Rating: NewRating(), Score: 1}})
	}
	if r.RD < MinRD {
		t.Fatalf("RD %f below the floor", r.RD)
	}
	settled := Rating{Value: 1600, RD: 60, Volatility: 0.06}
	if idle := settled.Idle(30); idle.RD <= 60 || idle.RD > MaxRD || idle.Value != 1600 {
		t.Fatalf("idle %+v", idle)
	}
	if far := settled.Idle(1_000_000); far.RD != MaxRD {
		t.Fatalf("idle cap %+v", far)
	}
}

func TestCompositeOpponent(t *testing.T) {
	c := Composite([]Rating{{Value: 1400, RD: 100}, {Value: 1600, RD: 200}, {Value: 1500, RD: 300}})
	if c.Value != 1500 || math.Abs(c.RD-math.Sqrt((100*100+200*200+300*300)/3.0)) > 1e-9 {
		t.Fatalf("composite %+v", c)
	}
	// A draw against an equal opponent leaves the rating where it was.
	if d := NewRating().Update([]Opponent{{Rating: NewRating(), Score: 0.5}}); math.Abs(d.Value-1500) > 1e-9 {
		t.Fatalf("draw %+v", d)
	}
}
