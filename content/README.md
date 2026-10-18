# BrainVolt question content

Questions live here as reviewable files, apart from code, so every question's origin is traceable (D-005, D-006, [schema-content.md](../docs/domain/schema-content.md)). `backend/cmd/seed` loads them into the database.

```
content/
├── taxonomy.yaml          categories and sub-categories (TR + EN names, display order)
├── sources.yaml           every source with its license and attribution (D-006)
├── SOURCES.md             the same, for people
├── brainvolt/*.yaml       the BrainVolt team's own questions, one file per category
└── opentdb/
    ├── raw/*.json         untouched OpenTDB snapshots (`seed opentdb-fetch`)
    └── selected.yaml      OpenTDB questions picked for BrainVolt: sub-category + Turkish text
```

## Commands (run in `backend/`, or `make seed` from the root)

| Command | What it does |
|---|---|
| `go run ./cmd/seed validate` (`make seed SEED=validate`) | Checks every file without a database. Also runs in CI (`TestRepositoryContent`), so a broken file never reaches `main`. |
| `go run ./cmd/seed import` (`make seed`) | Loads taxonomy, sources and questions. New questions start **in review** (`in_review`); questions already in the database (same wording and options) are skipped, so running it again is harmless. |
| `go run ./cmd/seed import -approve-as <user-id>` | **Local development only** (`BRAINVOLT_ENV=dev`): also verifies every question as that user, so local matches have questions. Never used on real data. |
| `go run ./cmd/seed opentdb-list` | Lists the raw OpenTDB questions not yet in `selected.yaml`, with their `ref`, type, difficulty, category and answer. |
| `go run ./cmd/seed opentdb-fetch` | Downloads the whole OpenTDB bank into `opentdb/raw/opentdb-YYYYMMDD.json` (needs internet; OpenTDB allows one request per 5 s, so it takes several minutes). |

Verification is a database state set by moderators (`POST /v1/questions/{id}/review`), never by these files (D-007). Only verified questions with a verified translation in the player's language are used in matches.

## Writing questions (`brainvolt/*.yaml`)

```yaml
source: brainvolt-team            # key from sources.yaml (required, D-006)
questions:
  - ref: bv-science-011           # unique and never reused; becomes source_ref
    subcategory: science/space    # category/sub-category slugs from taxonomy.yaml
    difficulty: 2                 # 1 (easy) … 5 (hard)
    # type: multiple_choice       # default; or true_false (then `answer: true|false`, no options)
    # locale: tr                  # language the question was written in (default tr)
    tr:
      prompt: Güneş Sistemi'nin en büyük gezegeni hangisidir?
      options: [Jüpiter, Satürn, Neptün, Dünya]   # FIRST option = correct answer
      explanation: İsteğe bağlı; cevaptan sonra gösterilir.
    en:
      prompt: What is the largest planet in the Solar System?
      options: [Jupiter, Saturn, Neptune, Earth]  # same order as tr
      # origin: human_translation  # default for the non-original language; or machine_translation
```

Rules (checked by `validate`):
- Multiple choice has exactly 4 options, the **first** is correct (matches shuffle them); options of one question differ from each other.
- Prompt 5–300 characters, option 1–120, explanation ≤ 500.
- Every question needs the text of its original language; add English (`en`) whenever possible (D-002).
- Quote values that contain `,` `:` `#` or start with a quote, e.g. `["300,000 km/s", …]` or `prompt: '"Çalıkuşu" romanının yazarı kimdir?'`.
- A question must be fact-checked, unambiguous, not time-sensitive ("the current champion…") and not offensive; moderators reject anything else.

## OpenTDB questions (`opentdb/`)

1. `seed opentdb-fetch` saves the bank unchanged under `raw/` (commit the file).
2. Pick suitable questions (relevant for a Turkish audience, not time-sensitive, unambiguous) into `selected.yaml` with their sub-category and Turkish text. The English original comes from `raw/`; option order is the correct answer first, then OpenTDB's wrong answers in their order:

   ```yaml
   questions:
     - ref: otdb-1a2b3c4d5e6f      # from `seed opentdb-list` (hash of the English question + answer)
       subcategory: science/physics
       # difficulty: 3             # optional; default easy/medium/hard → 2/3/4
       tr:
         prompt: …
         options: [doğru cevap, yanlış 1, yanlış 2, yanlış 3]
         origin: human_translation # or machine_translation (must be reviewed by a person)
   ```
3. `seed import`. OpenTDB content is **CC BY-SA 4.0**: the Credits screen and `SOURCES.md` credit it, and our Turkish translations are CC BY-SA 4.0 too (schema-content §6).
