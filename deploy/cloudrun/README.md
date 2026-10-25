# Cloud Run deployment (D-096)

BrainVolt runs as **one Cloud Run service**, `brainvolt-api`, built from
`backend/Dockerfile`. The API process also works the background jobs
(`BRAINVOLT_EMBED_WORKER=true`: outbox deliveries such as rewards, ranking
and notifications, plus the periodic jobs), so nothing has to stay awake
while nobody plays. `cmd/worker` still exists for a split setup but is not
deployed.

| Setting | Value | Why |
|---|---|---|
| `minScale` | 0 | No instance and no charge while idle. The first request after idle pays a cold start (startup CPU boost on). |
| `maxScale` | 1 | Rooms, matchmaking lobbies, rematch offers and WebSocket routing live in this process's memory; a second instance would split players. |
| `containerConcurrency` | 100 (`CONCURRENCY`) | Requests per instance. **Every open WebSocket holds one slot for its whole life**, so with one instance this is also the number of players connected at once; connection 101 waits / is refused. Go handles many more, so raise it (up to 1000) when the player count grows — it costs nothing extra by itself. |
| CPU allocation | request-based (`cpu-throttling: true`) | CPU is billed only while a request or WebSocket is open. Background loops (matchmaker, lobby timers, jobs) only matter while players are connected, which is exactly when CPU is allocated. |
| Request timeout | 3600 s | Cloud Run's maximum; a WebSocket is closed after it and the app reconnects and resumes. |
| Resources | 1 vCPU, 512 MiB | Below 1 vCPU Cloud Run allows only concurrency 1. |

Free tier (per month, per billing account): 180 000 vCPU-seconds,
360 000 GiB-seconds and 2 million requests. With request-based billing that
is about **50 hours of "someone is connected" time** per month at 1 vCPU —
the instance is billed for the whole time at least one WebSocket is open,
however many players share it. Idle connections are therefore what decides
the bill (handled in the app and the WebSocket gateway, D-097).

## Deploy

```bash
# 1. Build and push the image (Artifact Registry).
gcloud builds submit backend --tag "$IMAGE" --project "$PROJECT_ID"

# 2. Database URL as a secret (Supabase, see below), once.
printf '%s' "$SUPABASE_URL" | gcloud secrets create brainvolt-postgres-url --data-file=- --project "$PROJECT_ID"

# 3. Deploy the service.
PROJECT_ID=… REGION=europe-west1 IMAGE=… FIREBASE_PROJECT_ID=brainvolt \
POSTGRES_URL_SECRET=brainvolt-postgres-url ./deploy/cloudrun/deploy.sh

# 4. If an older separate worker service exists, delete it.
gcloud run services delete brainvolt-worker --project "$PROJECT_ID" --region "$REGION"
```

The service account needs `roles/secretmanager.secretAccessor` on the
secret and Firebase Cloud Messaging rights (pushes use Application Default
Credentials in prod). Migrations run separately (`/app/migrate`, e.g. as a
Cloud Run job) before a release that needs them.

## Supabase

Use the **Supavisor session-mode** connection string (port **5432**). The
transaction-mode pooler (6543) does not support the session settings each
connection gets (`statement_timeout`, `application_name`) or River's
`LISTEN/NOTIFY`. Pool sizes are set by the backend (D-098): at most 8
connections for the API and 3 for the embedded jobs (11 per instance, one
instance), none kept at rest and idle ones closed after a minute. Override
with `BRAINVOLT_POSTGRES_MAX_CONNS`, `BRAINVOLT_POSTGRES_JOBS_MAX_CONNS`,
`BRAINVOLT_POSTGRES_MAX_CONN_IDLE_TIME` and
`BRAINVOLT_POSTGRES_MAX_CONN_LIFETIME` (env entries in
`api.service.yaml`).

## AdMob server-side verification (D-103)

Rewarded ads are granted only by AdMob's signed callback. In AdMob, open the
`rewarded_main` ad unit → **Server-side verification** and set the callback
URL to

```
https://brainvolt-api-684037840139.europe-west3.run.app/v1/ads/admob/ssv
```

then press **Verify URL** (the service answers 200; the test call carries no
reward). The service fetches AdMob's public keys from
`https://www.gstatic.com/admob/reward/verifier-keys.json` and caches them for
a day; no secret is involved. Without this setting rewarded ads in release
builds play but never grant their reward.

## RevenueCat webhook (D-104)

BrainVolt Plus is granted by RevenueCat's webhook. `finish-deploy.ps1`
creates the Secret Manager secret `brainvolt-revenuecat-webhook` once with a
random value and prints it; the service reads it as
`BRAINVOLT_REVENUECAT_WEBHOOK_SECRET`. (`deploy.sh` expects the secret to
exist.) In RevenueCat → Integrations → Webhooks add:

- URL: `https://brainvolt-api-684037840139.europe-west3.run.app/v1/store/revenuecat/webhook`
- Authorization header value: the printed secret
  (`gcloud secrets versions access latest --secret=brainvolt-revenuecat-webhook`)
- Environment: both production and sandbox; events: all.

Use "Send test event": the service answers 200 (the test user is not a
BrainVolt user, so nothing is granted).
