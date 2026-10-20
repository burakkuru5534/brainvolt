package domain

import "testing"

func TestMask(t *testing.T) {
	for _, c := range []struct {
		in, out      string
		changed, sev bool
	}{
		{"İyi oyundu, tebrikler!", "İyi oyundu, tebrikler!", false, false},
		{"bu soru amk zordu", "bu soru *** zordu", true, false},
		{"Siktir ya, kaçırdım", "****** ya, kaçırdım", true, false},
		{"what the fuck", "what the ****", true, false},
		{"musiki ve şık", "musiki ve şık", false, false},
		{"sen bir orospu", "sen bir orospu", false, true},
	} {
		out, changed, severe := Mask(c.in)
		if out != c.out || changed != c.changed || severe != c.sev {
			t.Errorf("Mask(%q) = %q %v %v", c.in, out, changed, severe)
		}
	}
}
