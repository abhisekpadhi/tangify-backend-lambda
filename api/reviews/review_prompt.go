package reviews

import (
	"fmt"
	"math/rand/v2"
	"strings"
)

type promptStyle struct {
	dialectGuide     string
	lengthGuide      string
	openingGuide     string
	flowGuide        string
	toneGuide        string
	middleGuide      string
	endingGuide      string
	foodMentionGuide string
	verbGuide        string
}

func buildPrompt(rating int, menuItemNames []string) string {
	style := randomPromptStyle()
	tone := moodForRating(rating)

	menuSection := "Do not mention any specific dish names — talk about food/service in general only."
	if len(menuItemNames) > 0 {
		menuSection = fmt.Sprintf(
			"Menu items Tangify serves (ONLY use names from this list if mentioning food — never invent dishes):\n%s\n\n%s",
			strings.Join(menuItemNames, "\n"),
			style.foodMentionGuide,
		)
	}

	return fmt.Sprintf(`Write one Google Maps review for Tangify (Odia restaurant, Bhubaneswar area).

Star rating: %d/5
Overall mood: %s

THIS REVIEW MUST FOLLOW THIS STYLE (do not use a generic template):
- Language / dialect: %s
- Length: %s
- How to start: %s
- Flow / structure: %s
- Tone: %s
- What to cover in the middle: %s
- How to end: %s
- Use action verbs naturally (pick a few, don't stack them): %s

%s

Hard rules:
- Follow the language/dialect guide strictly for THIS review — do not default to plain English unless that guide says so
- Sound like one real person typing on phone — uneven, not polished
- Do NOT list multiple dishes like a menu recap — avoid "X and Y and Z, will come again" pattern
- Do NOT always mention 2-3 items — follow the food mention guide above
- Do NOT always end with "next time aasiba / will come again / must try" — often skip a closing line entirely
- No hashtags, emojis, bullet points, star counts in text, or AI-sounding phrases
- Reply with ONLY the review text`,
		rating,
		tone,
		style.dialectGuide,
		style.lengthGuide,
		style.openingGuide,
		style.flowGuide,
		style.toneGuide,
		style.middleGuide,
		style.endingGuide,
		style.verbGuide,
		menuSection,
	)
}

func moodForRating(rating int) string {
	return map[int]string{
		5: "very happy, loved it",
		4: "good experience, mostly pleased",
		3: "mixed — some good some meh",
		2: "disappointed",
		1: "unhappy, would not recommend",
	}[rating]
}

func randomPromptStyle() promptStyle {
	dialect := pickRandom(dialectGuides)
	return promptStyle{
		dialectGuide:     dialect,
		lengthGuide:      pickRandom(lengthGuides),
		openingGuide:     pickRandom(openingGuides),
		flowGuide:        pickRandom(flowGuides),
		toneGuide:        pickRandom(toneGuides),
		middleGuide:      pickRandom(middleGuides),
		endingGuide:      pickRandom(endingGuides),
		foodMentionGuide: pickRandom(foodMentionGuides),
		verbGuide:        pickRandomVerbSet(dialect),
	}
}

func pickRandom(options []string) string {
	return options[rand.IntN(len(options))]
}

func pickRandomVerbSet(dialect string) string {
	pool := verbPoolEnglish
	switch {
	case strings.Contains(dialect, "Odia") || strings.Contains(dialect, "odia"):
		pool = append(append([]string{}, verbPoolOdia...), verbPoolEnglish...)
	case strings.Contains(dialect, "Hindi") || strings.Contains(dialect, "Hinglish") || strings.Contains(dialect, "hinglish"):
		pool = append(append([]string{}, verbPoolHindi...), verbPoolEnglish...)
	default:
		pool = append(append([]string{}, verbPoolEnglish...), verbPoolOdia[:4]...)
	}

	verbs := append([]string{}, pool...)
	rand.Shuffle(len(verbs), func(i, j int) { verbs[i], verbs[j] = verbs[j], verbs[i] })
	count := 5 + rand.IntN(4)
	if count > len(verbs) {
		count = len(verbs)
	}
	return strings.Join(verbs[:count], ", ")
}

var dialectGuides = []string{
	"Plain English only — no Odia or Hindi words at all.",
	"Mostly English with 2–3 Odia words sprinkled in (romanized Odia).",
	"Odia-English mix — roughly half Odia half English, like locals in Bhubaneswar text.",
	"Heavy Odia in Roman script — mostly Odia with occasional English food words.",
	"Hinglish — Hindi + English mix ( yaar, mast, achha, bilkul, thoda, kha liya ).",
	"Mostly Hindi in Roman script — simple Hindi sentences, English only for dish names if needed.",
	"Light Hinglish — English sentences with Hindi filler words and verbs.",
	"Odia opening, English middle, Odia ending — switch dialect mid-review.",
	"English frame but every verb/action in Odia romanized form.",
	"Urban Bhubaneswar chat style — Odia + English + bit of Hindi casually mixed.",
}

var lengthGuides = []string{
	"One short sentence only, under 60 characters.",
	"One sentence, around 60–100 characters.",
	"2 short sentences in one paragraph.",
	"2–3 sentences in one paragraph, around 120–200 characters.",
	"3 sentences max in a single block — medium length.",
	"Two short paragraphs (separate with a blank line): 2 sentences in first, 1–2 in second.",
	"Two paragraphs: 3–4 sentences total, around 250–350 characters.",
	"One longer paragraph: 3–4 connected sentences, chatty but not essay-like.",
	"Very brief text-message style — under 80 characters, one line.",
	"2 paragraphs if it fits: first about food, second about service or vibe.",
}

var openingGuides = []string{
	"Start with the food you ate — no greeting, no 'visited Tangify'.",
	"Start with service or staff — food comes later or not at all.",
	"Start with the vibe / ambience / seating — not the restaurant name.",
	"Start with when you came ( lunch / dinner / aaji / yesterday ) — keep it casual.",
	"Start mid-thought, like continuing a chat: 'bhala thila...' or 'honestly...'",
	"Start with one strong opinion word: 'Solid.', 'Okay-ish.', 'Surprisingly good.'",
	"Start with value or portion size, not dish names.",
	"Start with a small complaint or praise — whichever fits the rating — no intro fluff.",
}

var flowGuides = []string{
	"Single thread: one topic only (food OR service OR vibe) — don't touch everything.",
	"Food first, then one quick line on something else.",
	"Service first, food only if there's room.",
	"Jump between two ideas without smooth transitions — like a real hurried review.",
	"Match the dialect guide — let language choice drive how choppy or smooth it reads.",
	"No clear structure — one or two flowing blocks of text.",
	"Mention one dish early, one sensory detail ( taste / spice / hot / crispy ) later.",
}

var toneGuides = []string{
	"Excited but not over the top.",
	"Chill and understated.",
	"Matter-of-fact, like reporting to a friend.",
	"Warm and grateful.",
	"Blunt and short.",
	"Chatty, slightly rambly but still short overall.",
	"Slightly humorous or playful.",
	"Tired/end-of-day casual — low energy wording.",
}

var middleGuides = []string{
	"Focus on taste or spice — skip staff and ambience.",
	"Mention waiting time or speed of service.",
	"Mention staff behaviour ( friendly / helpful / rushed ).",
	"Mention portion size or value for money.",
	"Mention one dish by name only — nothing else specific.",
	"Talk about the meal overall without naming any dish.",
	"Mention seating, AC, noise, or cleanliness — not food.",
	"Mention who you went with or the occasion briefly.",
	"Compare to expectations — better or worse than expected.",
	"One small nitpick even if rating is high, OR one redeeming point if rating is low.",
}

var endingGuides = []string{
	"End abruptly — no closing phrase, no 'will come again'.",
	"End with a single word or short fragment: 'Recommended.' / 'Decent.' / 'Okay.'",
	"End with recommendation to others — but don't say 'must try' every time.",
	"End with what you'd order differently next visit — only if it fits naturally.",
	"Trail off without a proper ending punctuation feel.",
	"End mentioning you'd return — but use varied wording, not 'next time aasiba' cliché.",
	"No ending about future visits at all.",
	"End with thanks to staff — only if service was mentioned earlier.",
}

var foodMentionGuides = []string{
	"Mention exactly ONE dish name from the list — do not name anything else.",
	"Do not name any dish — talk about food generically ( curry, thali, lassi, starter ).",
	"Mention one dish and one category word ( e.g. starter or lassi ) — max two food references total.",
	"Only hint at what you ate without exact menu names.",
	"If you name a dish, describe how it tasted in one verb — don't list more dishes.",
	"Pick one item from the list that fits starters/thali/mains/lassi/combo — never stack 3 names.",
}

var verbPoolOdia = []string{
	"khāilu", "khaidili", "taste kala", "lagila", "lagilani", "milila",
	"order kari thili", "mangili", "try kari dekhilu", "recommend karibi",
	"wait kailu", "serve hela", "dele", "asila", "pila", "khāibaku milila",
	"bhala lagila", "jaldi hela", "late hela", "maza āsi", "spicy thila",
	"bhari bhala", "tikiye spicy", "pet puila", "mote bhala lagila",
}

var verbPoolHindi = []string{
	"kha liya", "try kiya", "order kiya", "maza aaya", "pasand aaya",
	"wait kiya", "serve hua", "mil gaya", "recommend karunga", "laga",
	"achha tha", "mast tha", "thoda spicy", "jaldi mila", "late hua",
	"yaad raha", "dubara lunga", "share kiya", "enjoy kiya", "notice kiya",
}

var verbPoolEnglish = []string{
	"ordered", "tried", "loved", "liked", "enjoyed", "finished",
	"shared", "grabbed", "skipped", "returned", "recommended",
	"noticed", "remember", "craved", "reordered", "waited",
	"tasted", "savored", "devoured", "polished off", "went back for",
}
