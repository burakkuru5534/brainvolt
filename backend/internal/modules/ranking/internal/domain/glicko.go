// Package domain holds ranking's pure rules (progression-formulas.md
// ruleset 1): Glicko-2, Season Points, leagues and leaderboard points.
package domain

import "math"

// Glicko-2 parameters (progression-formulas §1.1).
const (
	InitialRating     = 1500.0
	InitialRD         = 350.0
	InitialVolatility = 0.06
	Tau               = 0.5
	epsilon           = 0.000001
	MinRD             = 50.0
	MaxRD             = 350.0
	// NewEntityRD is the RD of a new team or couple entity (§1.3).
	NewEntityRD = 200.0
	scale       = 173.7178
)

// Rating is a Glicko-2 rating on the display (1500) scale.
type Rating struct {
	Value      float64
	RD         float64
	Volatility float64
}

// NewRating is a player's first rating.
func NewRating() Rating {
	return Rating{Value: InitialRating, RD: InitialRD, Volatility: InitialVolatility}
}

// Opponent is one game of a rating period: the opponent's rating and the
// score (win 1, draw 0.5, loss 0).
type Opponent struct {
	Rating Rating
	Score  float64
}

func g(phi float64) float64 { return 1 / math.Sqrt(1+3*phi*phi/(math.Pi*math.Pi)) }

func expected(mu, muJ, phiJ float64) float64 {
	return 1 / (1 + math.Exp(-g(phiJ)*(mu-muJ)))
}

// Idle grows RD for idle days as empty rating periods (§1.2).
func (r Rating) Idle(days int) Rating {
	if days <= 0 {
		return r
	}
	phi := r.RD / scale
	phi = math.Sqrt(phi*phi + r.Volatility*r.Volatility*float64(days))
	r.RD = math.Min(phi*scale, MaxRD)
	return r
}

// Update applies one rating period (Glickman, "Example of the Glicko-2
// system", steps 2–8); RD stays within [MinRD, MaxRD].
func (r Rating) Update(games []Opponent) Rating {
	mu := (r.Value - InitialRating) / scale
	phi := r.RD / scale
	if len(games) == 0 {
		phi = math.Sqrt(phi*phi + r.Volatility*r.Volatility)
		return Rating{Value: r.Value, RD: clampRD(phi * scale), Volatility: r.Volatility}
	}
	var vInv, delta float64
	for _, o := range games {
		muJ := (o.Rating.Value - InitialRating) / scale
		phiJ := o.Rating.RD / scale
		e := expected(mu, muJ, phiJ)
		gj := g(phiJ)
		vInv += gj * gj * e * (1 - e)
		delta += gj * (o.Score - e)
	}
	v := 1 / vInv
	delta *= v
	sigma := newVolatility(r.Volatility, phi, v, delta)
	phiStar := math.Sqrt(phi*phi + sigma*sigma)
	phiNew := 1 / math.Sqrt(1/(phiStar*phiStar)+1/v)
	muNew := mu + phiNew*phiNew*(delta/v)
	return Rating{Value: muNew*scale + InitialRating, RD: clampRD(phiNew * scale), Volatility: sigma}
}

func clampRD(rd float64) float64 { return math.Max(MinRD, math.Min(MaxRD, rd)) }

// newVolatility is step 5 (Illinois algorithm).
func newVolatility(sigma, phi, v, delta float64) float64 {
	a := math.Log(sigma * sigma)
	f := func(x float64) float64 {
		ex := math.Exp(x)
		d := phi*phi + v + ex
		return ex*(delta*delta-phi*phi-v-ex)/(2*d*d) - (x-a)/(Tau*Tau)
	}
	A := a
	var B float64
	if delta*delta > phi*phi+v {
		B = math.Log(delta*delta - phi*phi - v)
	} else {
		k := 1.0
		for f(a-k*Tau) < 0 {
			k++
		}
		B = a - k*Tau
	}
	fA, fB := f(A), f(B)
	for math.Abs(B-A) > epsilon {
		C := A + (A-B)*fA/(fB-fA)
		fC := f(C)
		if fC*fB <= 0 {
			A, fA = B, fB
		} else {
			fA /= 2
		}
		B, fB = C, fC
	}
	return math.Exp(A / 2)
}

// Composite is the opposing side of a team match seen as one opponent:
// mean rating and √(mean RD²) (§1.3).
func Composite(rs []Rating) Rating {
	if len(rs) == 0 {
		return NewRating()
	}
	var sum, sq float64
	for _, r := range rs {
		sum += r.Value
		sq += r.RD * r.RD
	}
	n := float64(len(rs))
	return Rating{Value: sum / n, RD: math.Sqrt(sq / n), Volatility: InitialVolatility}
}
