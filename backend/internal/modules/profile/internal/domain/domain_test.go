package domain

import (
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestGeneratedNames(t *testing.T) {
	for attempt, digits := range map[int]int{0: 5, 2: 5, 3: 7, 5: 7, 6: 9, 9: 9} {
		u := GeneratedUsername(attempt)
		if !regexp.MustCompile(`^oyuncu[1-9][0-9]{`+string(rune('0'+digits-1))+`}$`).MatchString(u) || !ValidUsername(u) {
			t.Errorf("attempt %d: %q, want oyuncu + %d digits", attempt, u, digits)
		}
	}
	if d := GeneratedDisplayName("tr"); !regexp.MustCompile(`^Misafir \d{4}$`).MatchString(d) || !ValidDisplayName(d) {
		t.Errorf("tr display name %q", d)
	}
	if d := GeneratedDisplayName("en"); !regexp.MustCompile(`^Guest \d{4}$`).MatchString(d) {
		t.Errorf("en display name %q", d)
	}
	if a := RandomAvatar(); !ValidAvatar(a) {
		t.Errorf("RandomAvatar = %q", a)
	}
}

func TestFieldRules(t *testing.T) {
	for in, ok := range map[string]bool{"ali.veli": true, "ab": false, "has space": false, "Upper": false, "a_b.c9": true, strings.Repeat("a", 21): false} {
		if ValidUsername(in) != ok {
			t.Errorf("ValidUsername(%q) = %v", in, !ok)
		}
	}
	if NormalizeUsername("  Ali.Veli ") != "ali.veli" {
		t.Error("NormalizeUsername")
	}
	for in, ok := range map[string]bool{"Ali": true, "A": false, "Şükrü Öztürk": true, strings.Repeat("ç", 24): true, strings.Repeat("ç", 25): false, "a\u0007b": false} {
		if ValidDisplayName(in) != ok {
			t.Errorf("ValidDisplayName(%q) = %v", in, !ok)
		}
	}
	if NormalizeDisplayName("  Ali   Veli ") != "Ali Veli" {
		t.Error("NormalizeDisplayName")
	}
	if !ValidCountry(NormalizeCountry(" tr ")) || ValidCountry("TUR") || ValidCountry("T1") {
		t.Error("country rules")
	}
}

func TestChangeCooldown(t *testing.T) {
	now := time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)
	if !CanChange(nil, now) || NextChange(nil) != nil {
		t.Fatal("never changed must be changeable now")
	}
	last := now.Add(-29 * 24 * time.Hour)
	if CanChange(&last, now) {
		t.Fatal("29 days after a change must be blocked")
	}
	if next := NextChange(&last); !next.Equal(now.Add(24 * time.Hour)) {
		t.Fatalf("NextChange = %v", next)
	}
	last = now.Add(-30 * 24 * time.Hour)
	if !CanChange(&last, now) {
		t.Fatal("exactly 30 days later must be allowed")
	}
}

func TestWordFilter(t *testing.T) {
	blocked := []string{"amk", "Siktir git", "0r0spu", "f.u.c.k", "FuckYou", "Admin", "brain volt", "Moderatör", "y@rr@k", "Destek Ekibi", "piç", "PİÇ", "s1k", "Yönetici"}
	for _, s := range blocked {
		if !Rejected(s) {
			t.Errorf("Rejected(%q) = false", s)
		}
	}
	allowed := []string{"Misafir 4821", "Musiki Sever", "Gotham", "Picasso", "Aslı Şık", "dickens", "Ali Veli", "oyuncu48213", "Kocaman Kedi", "Şık Oyuncu", "Pıçak", "Göttingen"}
	for _, s := range allowed {
		if Rejected(s) {
			t.Errorf("Rejected(%q) = true (false positive)", s)
		}
	}
}
