# BrainVolt app (Flutter)

Android + iOS client. Architecture: [ADR-0003](../docs/adr/0003-mobile-architecture.md). Flutter **3.47.1** (`.flutter-version`).

## Run

Start the backend first (`make up`, `make migrate`, `make run` in the repository root), then:

```bash
cd mobile
flutter run --dart-define-from-file=env/dev.json     # Android emulator → backend at 10.0.2.2:8080
```

- `env/dev.json` targets the Android emulator, which reaches the host computer as `10.0.2.2`. On the iOS simulator use `localhost`; on a physical phone use the computer's LAN address (same Wi-Fi) — copy the file, e.g. `env/dev.local.json`, and pass that one.
- Plain HTTP to local addresses is allowed only in debug builds (Android `src/debug` network security config, iOS `NSAllowsLocalNetworking`).
- `env/prod.json` targets the deployed API (https/wss only; the app refuses anything else for prod).
- Starting without `--dart-define-from-file` stops with a message saying exactly that.

## Develop

| Task | Command (or `make …` from the repository root) |
|---|---|
| Tests | `flutter test` |
| Lint (incl. riverpod_lint) | `dart analyze --fatal-infos` — `flutter analyze` does **not** run the riverpod_lint plugin |
| Code generation | `dart run tool/gen_error_l10n.dart && flutter gen-l10n && dart run build_runner build -d` (`make gen`) |

Generated files (`*.g.dart`, `*.freezed.dart`, `lib/core/l10n/generated/`, `lib/core/network/error_messages.g.dart`) are committed; `make check-gen` fails when they are stale.

## Layout

```
lib/
├── main.dart            builds config, shared preferences, app version; starts the app
├── app/                 app widget, router (go_router), theme, config (flavors), injected providers
├── core/
│   ├── l10n/            ARB files (tr = template, en), generated localizations, language choice
│   └── network/         dio client (headers, Firebase token, TOKEN_EXPIRED retry), ApiException, error texts
└── features/<feature>/  data / domain / presentation (ADR-0003 rules)
tool/gen_error_l10n.dart error_<CODE> texts from the backend registry (docs/api/errors.md)
```
