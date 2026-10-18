# BrainVolt app (Flutter)

Android + iOS client. Architecture: [ADR-0003](../docs/adr/0003-mobile-architecture.md). Flutter **3.47.1** (`.flutter-version`).

## Run

Start the backend first (`make up`, `make migrate`, `make run` in the repository root), then:

```bash
cd mobile
flutter run --dart-define-from-file=env/dev.json     # Android emulator → backend at 10.0.2.2:8080
```

- `env/dev.json` targets the Android emulator, which reaches the host computer as `10.0.2.2`; `env/dev_ios.json` is the same for the iOS simulator (`localhost`). On a physical phone use the computer's LAN address (same Wi-Fi) — copy the file, e.g. `env/dev.local.json`, and pass that one.
- Dev builds sign in against the **Firebase Auth emulator** from `make up` (`AUTH_EMULATOR_HOST`), never the real project; emulator users disappear when the container restarts (the app then starts as a new guest after its next sign-out or reinstall). Verification emails are not sent: the emulator prints the link in `docker compose logs firebase-auth`; open it in a browser, then tap "Doğruladım".
- Plain HTTP to local addresses is allowed only in debug builds (Android `src/debug` network security config, iOS `NSAllowsLocalNetworking`).
- `env/prod.json` targets the deployed API (https/wss only; the app refuses anything else for prod).
- Starting without `--dart-define-from-file` stops with a message saying exactly that.

## Sign-in setup (Firebase project `brainvolt-a9caa`)

- Config: `lib/firebase_options.dart` and `android/app/google-services.json` come from `flutterfire configure` (client keys, safe to commit). Re-run it after adding apps or providers.
- **Google on Android** needs each signing key's SHA-1 in Firebase console → Project settings → Android app: debug key (`cd android && ./gradlew signingReport`) now, Play App Signing key before release. Without it Google sign-in fails with "canceled" right after choosing an account.
- **Apple (iOS, D-057: tested later):** `ios/Runner/Runner.entitlements` enables Sign in with Apple; the App ID needs the same capability and Firebase console → Authentication → Apple must be enabled. Google's iOS callback URL scheme is in `ios/Runner/Info.plist`.
- Enabled providers in Firebase console → Authentication: Anonymous, Google, Email/Password (+ Apple later).

## Flows

Age gate (D-020) → anonymous guest → `POST /v1/session` → home. Guests link Google/Apple/email in Settings → Account (same user, progress kept, D-013a); if the account already exists the app warns (D-015), signs in to it and then abandons the guest. Email accounts verify their address (D-017). Suspended or deletion-pending accounts see only the status screen.

## Windows: Kotlin "Could not close incremental caches"

When the project is on another drive than the pub cache (e.g. project on `D:`, cache in `C:\Users\…\AppData\Local\Pub\Cache`), Kotlin's incremental compilation of plugins such as `package_info_plus` fails. `android/gradle.properties` therefore sets `kotlin.incremental=false`. If a build already failed, clear the broken caches once:

```powershell
cd mobile
flutter clean
cd android; .\gradlew --stop; cd ..
flutter pub get
flutter run --dart-define-from-file=env/dev.json
```

Optional, for faster builds: keep the pub cache on the project's drive (`setx PUB_CACHE D:\pub-cache`, open a new terminal, `flutter pub get`).

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
├── main.dart            builds config, Firebase (+ Auth emulator in dev), shared preferences, app version
├── app/                 app widget, router + sign-in guard, session controller, theme, config (flavors), injected providers
├── core/
│   ├── l10n/            ARB files (tr = template, en), generated localizations, language choice
│   ├── network/         dio client (headers, Firebase token, TOKEN_EXPIRED retry), ApiException, error texts
│   ├── storage/         installation id
│   └── widgets/         shared dialogs
└── features/<feature>/  data / domain / presentation (ADR-0003 rules): auth, home, profile
tool/gen_error_l10n.dart error_<CODE> texts from the backend registry (docs/api/errors.md)
```
