package live

import (
	"encoding/json"
	"maps"

	"github.com/google/uuid"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/content"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/game/internal/domain"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/game/internal/domain/engine"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/profile"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/wsserver"
)

// meta is what rendering needs besides the engine: question texts in the
// match's locales and the participants' public cards. It is written once
// when the owner starts (Redis :state "meta").
type meta struct {
	MatchID uuid.UUID                                     `json:"match_id"`
	Texts   map[uuid.UUID]map[string]content.QuestionText `json:"texts"`
	Cards   map[uuid.UUID]profile.Card                    `json:"cards"`
	Seats   []engine.Seat                                 `json:"seats"`
	// Summary is the configuration shown in match.created.
	Summary map[string]any `json:"summary"`
	// Bots are development practice opponents played by the owner (D-066).
	Bots []uuid.UUID `json:"bots,omitempty"`
	// Categories are the category names per locale (Mixed Battle picks).
	Categories map[uuid.UUID]map[string]string `json:"categories,omitempty"`
}

// addCategories records the category names of questions whose texts are
// loaded.
func (m *meta) addCategories(qs []engine.Question) {
	for _, q := range qs {
		for locale, t := range m.Texts[q.ID] {
			if t.Category == "" {
				continue
			}
			if m.Categories == nil {
				m.Categories = map[uuid.UUID]map[string]string{}
			}
			if m.Categories[q.CategoryID] == nil {
				m.Categories[q.CategoryID] = map[string]string{}
			}
			m.Categories[q.CategoryID][locale] = t.Category
		}
	}
}

// categoryList names categories in the recipient's locale.
func (m *meta) categoryList(ids []any, locale string) []map[string]any {
	out := make([]map[string]any, 0, len(ids))
	for _, v := range ids {
		id, _ := uuid.Parse(str(v))
		names := m.Categories[id]
		name, ok := names[locale]
		if !ok {
			for _, n := range names {
				name = n
				break
			}
		}
		out = append(out, map[string]any{"id": id, "name": name})
	}
	return out
}

// stored is one match-stream event as kept in game:{m}:events.
type stored struct {
	Seq     int             `json:"seq"`
	Type    string          `json:"type"`
	To      engine.Audience `json:"to"`
	TS      int64           `json:"ts"`
	Payload map[string]any  `json:"payload"`
}

// toStored turns an engine event into its stored JSON form, so live
// delivery and replay render the same values.
func toStored(e engine.Event, ts int64) (stored, error) {
	b, err := json.Marshal(e.Payload)
	if err != nil {
		return stored{}, err
	}
	s := stored{Seq: e.Seq, Type: e.Type, To: e.To, TS: ts}
	return s, json.Unmarshal(b, &s.Payload)
}

// recipient is who an event is rendered for.
type recipient struct {
	UserID uuid.UUID
	Side   int
	Locale string
}

func streamOf(match uuid.UUID) string { return "match:" + match.String() }

// text picks the recipient's locale, falling back to any available one.
func (m *meta) text(q uuid.UUID, locale string) content.QuestionText {
	byLocale := m.Texts[q]
	if t, ok := byLocale[locale]; ok {
		return t
	}
	for _, t := range byLocale {
		return t
	}
	return content.QuestionText{}
}

func str(v any) string { s, _ := v.(string); return s }

// render builds the frame one participant receives for a stored event
// (websocket.md §5.2): texts in their language, the own answer only for
// the sender, and events meant for another audience reduced to their
// round number so every stream stays gap-free.
func (m *meta) render(e stored, r recipient) wsserver.Outbound {
	d := maps.Clone(e.Payload)
	if d == nil {
		d = map[string]any{}
	}
	f := wsserver.Outbound{T: e.Type, S: streamOf(m.MatchID), Seq: int64(e.Seq), TS: e.TS, D: d}
	visible := e.To.All || (e.To.Side != 0 && e.To.Side == r.Side) || (e.To.User != uuid.Nil && e.To.User == r.UserID)
	if !visible {
		f.D = map[string]any{"match_id": m.MatchID, "round_no": d["round_no"], "redacted": true}
		return f
	}
	d["match_id"] = m.MatchID
	switch e.Type {
	case "match.created":
		maps.Copy(d, m.Summary)
		d["participants"] = m.participantList()
	case "round.started":
		qid, _ := uuid.Parse(str(d["question_id"]))
		t := m.text(qid, r.Locale)
		q := map[string]any{"id": qid, "prompt": t.Prompt}
		if cat := str(d["category_id"]); cat != "" {
			q["category"] = map[string]any{"id": cat, "name": t.Category}
		}
		ids, _ := d["options"].([]any)
		opts := make([]map[string]any, 0, len(ids))
		for _, v := range ids {
			id, _ := uuid.Parse(str(v))
			opts = append(opts, map[string]any{"id": id, "text": t.Options[id]})
		}
		d["question"], d["options"] = q, opts
		delete(d, "question_id")
		delete(d, "category_id")
		delete(d, "subcategory_id")
	case "round.pre_round":
		ids, _ := d["categories"].([]any)
		d["categories"] = m.categoryList(ids, r.Locale)
	case "round.answer_received":
		if str(d["user_id"]) != r.UserID.String() {
			delete(d, "option_id")
		}
	case "round.revealed":
		qid, _ := uuid.Parse(str(d["question_id"]))
		t := m.text(qid, r.Locale)
		if t.Explanation != "" {
			d["explanation"] = t.Explanation
		}
		d["category"] = map[string]any{"id": d["category_id"], "name": t.Category}
		delete(d, "category_id")
	}
	return f
}

func (m *meta) participantList() []map[string]any {
	out := make([]map[string]any, 0, len(m.Seats))
	for _, s := range m.Seats {
		c := m.Cards[s.UserID]
		out = append(out, map[string]any{
			"user_id": s.UserID, "side_no": s.Side, "seat_no": s.Seat,
			"display_name": c.DisplayName, "avatar_id": c.AvatarID, "country_code": c.CountryCode,
		})
	}
	return out
}

// snapshot is match.snapshot for one participant (websocket.md §6): the
// whole screen state, never the correct option of an open round.
func (m *meta) snapshot(match *engine.Match, r recipient, ts int64) wsserver.Outbound {
	parts := make([]map[string]any, 0, len(match.Participants))
	for _, p := range match.Participants {
		c := m.Cards[p.UserID]
		parts = append(parts, map[string]any{
			"user_id": p.UserID, "side_no": p.Side, "seat_no": p.Seat, "status": p.Status,
			"display_name": c.DisplayName, "avatar_id": c.AvatarID, "country_code": c.CountryCode,
		})
	}
	sides := make([]map[string]any, 2)
	for i := range 2 {
		sides[i] = map[string]any{"side_no": i + 1, "score": match.Sides[i].Score, "correct": match.Sides[i].Correct}
	}
	d := map[string]any{
		"match_id": m.MatchID, "status": match.Status, "phase": match.Phase, "round_no": match.RoundNo,
		"match_type": match.Config.MatchType, "format": match.Config.Format,
		"question_count": match.Derived.QuestionCount, "answer_window_ms": match.Derived.AnswerWindowMS,
		"ready_deadline_at": match.ReadyDeadMS, "participants": parts, "sides": sides,
	}
	if match.StartsAtMS > 0 {
		d["starts_at"] = match.StartsAtMS
	}
	if rd := match.Round; rd != nil && (match.Phase == engine.PhaseQuestion || match.Phase == engine.PhaseReveal) {
		t := m.text(rd.Question.ID, r.Locale)
		opts := make([]map[string]any, 0, len(rd.OptionOrder))
		for _, pos := range rd.OptionOrder {
			id := rd.Question.Options[pos-1]
			opts = append(opts, map[string]any{"id": id, "text": t.Options[id]})
		}
		q := map[string]any{"id": rd.Question.ID, "prompt": t.Prompt}
		if match.Config.CategoryMode != domain.BlindChallenge {
			q["category"] = map[string]any{"id": rd.Question.CategoryID, "name": t.Category}
		}
		round := map[string]any{"round_no": rd.No, "question": q, "options": opts, "show_at": rd.ShowAtMS, "deadline_at": rd.DeadlineMS}
		if opt, ok := match.AnswerOf(r.UserID); ok {
			round["own_option_id"] = opt
		}
		d["round"] = round
	}
	if rd := match.Round; rd != nil && match.Phase == engine.PhasePreRound {
		ids := make([]any, len(match.Config.CategoryIDs))
		for i, c := range match.Config.CategoryIDs {
			ids[i] = c.String()
		}
		d["pre_round"] = map[string]any{
			"round_no": rd.No, "picker_side_no": rd.PickerSide, "picker_user_id": rd.Picker,
			"categories": m.categoryList(ids, r.Locale), "allowed_category_ids": match.AllowedPicks(), "pick_deadline_at": rd.PickDeadMS,
		}
	}
	if res := match.Result(); res != nil {
		d["result"] = map[string]any{
			"end_reason": res.EndReason, "cancel_reason": res.CancelReason, "winner_side_no": res.WinnerSide,
			"is_draw": res.Draw, "tie_break": res.TieBreak,
		}
	}
	return wsserver.Outbound{T: "match.snapshot", S: streamOf(m.MatchID), Seq: int64(match.Seq()), TS: ts, D: d}
}
