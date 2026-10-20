package service

import "strings"

// Categories a user can tune (schema-platform §1).
var Categories = []string{"challenge", "friend", "match_found", "team", "chat", "season", "quest", "tournament", "moderation", "product_news"}

// template is the text of a notification kind in each language; {name}
// and the like are filled from the item's params.
type template struct {
	category string
	title    map[string]string
	body     map[string]string
}

var templates = map[string]template{
	"friend.request": {"friend",
		map[string]string{"tr": "Arkadaşlık isteği", "en": "Friend request"},
		map[string]string{"tr": "{name} sana arkadaşlık isteği gönderdi.", "en": "{name} sent you a friend request."}},
	"friend.accepted": {"friend",
		map[string]string{"tr": "Yeni arkadaş", "en": "New friend"},
		map[string]string{"tr": "{name} arkadaşlık isteğini kabul etti.", "en": "{name} accepted your friend request."}},
	"challenge.received": {"challenge",
		map[string]string{"tr": "Meydan okuma!", "en": "Challenge!"},
		map[string]string{"tr": "{name} seni düelloya çağırıyor.", "en": "{name} challenged you to a duel."}},
	"challenge.accepted": {"challenge",
		map[string]string{"tr": "Meydan okuma kabul edildi", "en": "Challenge accepted"},
		map[string]string{"tr": "{name} meydan okumanı kabul etti, maç başlıyor!", "en": "{name} accepted your challenge. The match is on!"}},
	"challenge.revenge": {"challenge",
		map[string]string{"tr": "Rövanş isteği!", "en": "Revenge!"},
		map[string]string{"tr": "{name} rövanş istiyor. Kabul ediyor musun?", "en": "{name} wants revenge. Are you in?"}},
	"chat.team": {"chat",
		map[string]string{"tr": "Takım sohbeti", "en": "Team chat"},
		map[string]string{"tr": "{name} takım sohbetine yazdı.", "en": "{name} wrote in your team chat."}},
	"chat.couple": {"chat",
		map[string]string{"tr": "Yeni mesaj", "en": "New message"},
		map[string]string{"tr": "{name} sana mesaj gönderdi.", "en": "{name} sent you a message."}},
	"couple.request": {"team",
		map[string]string{"tr": "Çift isteği", "en": "Couple request"},
		map[string]string{"tr": "{name} seninle çift olmak istiyor.", "en": "{name} wants to be your couple partner."}},
	"moderation.sanction": {"moderation",
		map[string]string{"tr": "Hesabınla ilgili bir karar", "en": "A decision about your account"},
		map[string]string{"tr": "Hesabına bir kısıtlama uygulandı. Ayrıntılar: Ayarlar → Yaptırımlarım.", "en": "A restriction was applied to your account. See Settings → My sanctions."}},
}

// render fills a kind's title and body in locale (Turkish when unknown).
func render(kind, locale string, params map[string]string) (title, body string) {
	t, ok := templates[kind]
	if !ok {
		return kind, ""
	}
	if locale != "en" {
		locale = "tr"
	}
	title, body = t.title[locale], t.body[locale]
	for k, v := range params {
		title = strings.ReplaceAll(title, "{"+k+"}", v)
		body = strings.ReplaceAll(body, "{"+k+"}", v)
	}
	return title, body
}
