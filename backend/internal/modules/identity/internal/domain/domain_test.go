package domain

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/burakkuru5534/brainvolt/backend/internal/platform/firebase"
)

func TestDeriveTier(t *testing.T) {
	anon := LinkedIdentity{Provider: ProviderAnonymous}
	google := LinkedIdentity{Provider: ProviderGoogle}
	apple := LinkedIdentity{Provider: ProviderApple}
	pwUnverified := LinkedIdentity{Provider: ProviderPassword}
	pwVerified := LinkedIdentity{Provider: ProviderPassword, EmailVerified: true}

	tests := []struct {
		name string
		in   []LinkedIdentity
		want Tier
	}{
		{"anonymous only", []LinkedIdentity{anon}, TierGuest},
		{"nothing linked", nil, TierGuest},
		{"password unverified", []LinkedIdentity{pwUnverified}, TierUnverified},
		{"anonymous + password unverified", []LinkedIdentity{anon, pwUnverified}, TierUnverified},
		{"password verified", []LinkedIdentity{pwVerified}, TierFull},
		{"google", []LinkedIdentity{google}, TierFull},
		{"apple", []LinkedIdentity{apple}, TierFull},
		{"google + unverified password", []LinkedIdentity{pwUnverified, google}, TierFull},
		{"apple + anonymous", []LinkedIdentity{anon, apple}, TierFull},
	}
	for _, tc := range tests {
		if got := DeriveTier(tc.in); got != tc.want {
			t.Errorf("%s: DeriveTier = %s, want %s", tc.name, got, tc.want)
		}
	}
	if !TierGuest.Restricted() || !TierUnverified.Restricted() || TierFull.Restricted() {
		t.Error("Restricted() must be true exactly for guest and unverified (D-016, D-017)")
	}
}

func ptr(s string) *string { return &s }

func TestTierFromTokenMatchesAccount(t *testing.T) {
	tests := []struct {
		name    string
		token   firebase.Token
		account firebase.Account
		want    []LinkedIdentity
		tier    Tier
	}{
		{
			name:    "anonymous",
			token:   firebase.Token{Provider: "anonymous", Identities: map[string][]string{}},
			account: firebase.Account{UID: "u"},
			want:    []LinkedIdentity{{Provider: ProviderAnonymous}},
			tier:    TierGuest,
		},
		{
			name:    "email not verified",
			token:   firebase.Token{Provider: "password", Email: "ali@example.com", Identities: map[string][]string{"email": {"ali@example.com"}}},
			account: firebase.Account{Email: "ali@example.com", Providers: []firebase.ProviderInfo{{ProviderID: "password", UID: "ali@example.com", Email: "ali@example.com"}}},
			want:    []LinkedIdentity{{Provider: ProviderPassword, Email: ptr("ali@example.com")}},
			tier:    TierUnverified,
		},
		{
			name:    "email verified",
			token:   firebase.Token{Provider: "password", EmailVerified: true, Identities: map[string][]string{"email": {"ali@example.com"}}},
			account: firebase.Account{EmailVerified: true, Providers: []firebase.ProviderInfo{{ProviderID: "password", Email: "ali@example.com"}}},
			want:    []LinkedIdentity{{Provider: ProviderPassword, Email: ptr("ali@example.com"), EmailVerified: true}},
			tier:    TierFull,
		},
		{
			// A Google token also carries an "email" identity; only the
			// account record shows that no password is linked.
			name: "guest upgraded with Google",
			token: firebase.Token{Provider: "google.com", Email: "ali@gmail.com", EmailVerified: true,
				Identities: map[string][]string{"google.com": {"1098"}, "email": {"ali@gmail.com"}}},
			account: firebase.Account{EmailVerified: true, Providers: []firebase.ProviderInfo{{ProviderID: "google.com", UID: "1098", Email: "ali@gmail.com"}}},
			want:    []LinkedIdentity{{Provider: ProviderGoogle, ProviderUID: ptr("1098"), Email: ptr("ali@gmail.com"), EmailVerified: true}},
			tier:    TierFull,
		},
		{
			name:  "apple relay address plus password",
			token: firebase.Token{Provider: "apple.com", EmailVerified: false, Identities: map[string][]string{"apple.com": {"000123.abc"}, "email": {"a@b.c"}}},
			account: firebase.Account{Providers: []firebase.ProviderInfo{
				{ProviderID: "password", Email: "a@b.c"},
				{ProviderID: "apple.com", UID: "000123.abc", Email: "x1@privaterelay.appleid.com"},
			}},
			want: []LinkedIdentity{
				{Provider: ProviderApple, ProviderUID: ptr("000123.abc"), Email: ptr("x1@privaterelay.appleid.com")},
				{Provider: ProviderPassword, Email: ptr("a@b.c")},
			},
			tier: TierFull,
		},
	}
	for _, tc := range tests {
		got := IdentitiesFromAccount(tc.account)
		if diff := cmp.Diff(tc.want, got); diff != "" {
			t.Errorf("%s (-want +got):\n%s", tc.name, diff)
		}
		if tier := DeriveTier(got); tier != tc.tier {
			t.Errorf("%s: DeriveTier = %s, want %s", tc.name, tier, tc.tier)
		}
		if tier := TierFromToken(tc.token); tier != tc.tier {
			t.Errorf("%s: TierFromToken = %s, want %s", tc.name, tier, tc.tier)
		}
	}
}
