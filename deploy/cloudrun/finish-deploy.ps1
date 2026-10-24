# Finishes the BrainVolt Cloud Run deployment (D-096) from PowerShell,
# after the image was pushed:
#   1. Secret Manager: brainvolt-postgres-url, taken from the database URL the
#      running service already has (no password is typed or printed)
#   2. read access for the service account
#   3. deploy deploy/cloudrun/api.service.yaml (min 0, max 1, embedded worker)
#   4. public access, health checks, leftover worker service
#
# Run from the repository root:
#   powershell -ExecutionPolicy Bypass -File deploy\cloudrun\finish-deploy.ps1
# Optional: -Tag <image tag> (default: the current git commit), -Concurrency <n>.
param(
    [string]$ProjectId = "brainvolt-a9caa",
    [string]$Region = "europe-west3",
    [string]$Repo = "europe-west3-docker.pkg.dev/brainvolt-a9caa/brainvolt-repo/backend",
    [string]$Tag = "",
    [string]$FirebaseProjectId = "brainvolt-a9caa",
    [string]$Secret = "brainvolt-postgres-url",
    [int]$Concurrency = 250
)
# Native gcloud errors are checked with $LASTEXITCODE (Check); "Stop" would
# turn harmless stderr output into errors in Windows PowerShell 5.1.
$ErrorActionPreference = "Continue"
# Never wait for a gcloud prompt (they go to stderr and look like a hang).
$env:CLOUDSDK_CORE_DISABLE_PROMPTS = "1"

function Step($text) { Write-Host "`n==> $text" -ForegroundColor Cyan }
function Check($what) { if ($LASTEXITCODE -ne 0) { throw "$what failed (exit $LASTEXITCODE)" } }

$root = (Resolve-Path (Join-Path $PSScriptRoot "..\..")).Path
Set-Location $root
if (-not $Tag) { $Tag = (git rev-parse --short HEAD).Trim() }
$Image = "${Repo}:${Tag}"

Step "Project $ProjectId, region $Region, image $Image"
gcloud config set project $ProjectId | Out-Null; Check "gcloud config"
gcloud artifacts docker images describe $Image --format="value(image_summary.digest)" | Out-Null
if ($LASTEXITCODE -ne 0) {
    throw "Image $Image not found in Artifact Registry. Push it first (docker build/push with this tag) or pass -Tag <the pushed tag>."
}
gcloud services enable run.googleapis.com secretmanager.googleapis.com artifactregistry.googleapis.com; Check "enable APIs"

Step "Secret $Secret"
$found = gcloud secrets list --filter="name ~ /$Secret$" --format="value(name)"
Check "list secrets"
$secretExists = [bool]$found
$svcJson = gcloud run services describe brainvolt-api --region $Region --format=json
Check "describe brainvolt-api"
$svc = $svcJson | ConvertFrom-Json
$envs = $svc.spec.template.spec.containers[0].env
$pg = ($envs | Where-Object { $_.name -eq "BRAINVOLT_POSTGRES_URL" -and $_.value }).value
if ($pg) {
    $tmp = Join-Path $env:TEMP ("pg-" + [guid]::NewGuid().ToString() + ".txt")
    try {
        [System.IO.File]::WriteAllText($tmp, $pg)  # UTF-8, no BOM, no newline
        if ($secretExists) {
            gcloud secrets versions add $Secret --data-file="$tmp" | Out-Null; Check "secret version"
            Write-Host "added a new version to $Secret (from the running service)"
        } else {
            gcloud secrets create $Secret --replication-policy=automatic --data-file="$tmp" | Out-Null; Check "secret create"
            Write-Host "created $Secret (from the running service)"
        }
    } finally {
        Remove-Item $tmp -ErrorAction SilentlyContinue
        Remove-Variable pg
    }
} elseif ($secretExists) {
    Write-Host "the service already reads the URL from a secret; keeping $Secret as it is"
} else {
    throw "No BRAINVOLT_POSTGRES_URL on the service and no secret ${Secret}: create the secret first."
}

Step "Secret access for the service account"
$sa = $svc.spec.template.spec.serviceAccountName
if (-not $sa) {
    $num = (gcloud projects describe $ProjectId --format="value(projectNumber)").Trim()
    $sa = "$num-compute@developer.gserviceaccount.com"
}
Write-Host "service account: $sa"
gcloud secrets add-iam-policy-binding $Secret --member="serviceAccount:$sa" --role="roles/secretmanager.secretAccessor" | Out-Null
Check "secret IAM"

Step "Deploy brainvolt-api (min 0, max 1, concurrency $Concurrency, embedded worker)"
$yaml = Get-Content -Raw (Join-Path $root "deploy\cloudrun\api.service.yaml")
$yaml = $yaml.Replace('${REGION}', $Region).Replace('${IMAGE}', $Image).Replace('${CONCURRENCY}', "$Concurrency")
$yaml = $yaml.Replace('${FIREBASE_PROJECT_ID}', $FirebaseProjectId).Replace('${POSTGRES_URL_SECRET}', $Secret)
$rendered = Join-Path $env:TEMP "brainvolt-api.service.yaml"
[System.IO.File]::WriteAllText($rendered, $yaml)
try {
    gcloud run services replace $rendered --region $Region; Check "deploy"
} finally {
    Remove-Item $rendered -ErrorAction SilentlyContinue
}

Step "Public access"
gcloud run services add-iam-policy-binding brainvolt-api --region $Region --member=allUsers --role=roles/run.invoker | Out-Null
Check "public access"

Step "Checks"
$url = (gcloud run services describe brainvolt-api --region $Region --format="value(status.url)").Trim()
gcloud run services describe brainvolt-api --region $Region --format="yaml(spec.template.metadata.annotations,spec.template.spec.containerConcurrency)"
foreach ($path in "/healthz", "/readyz") {
    try {
        $r = Invoke-WebRequest -UseBasicParsing -TimeoutSec 60 "$url$path"
        Write-Host "$path -> $($r.StatusCode) $($r.Content)"
    } catch {
        Write-Host "$path -> FAILED: $($_.Exception.Message)" -ForegroundColor Yellow
    }
}

Step "Leftover worker service"
$services = gcloud run services list --region $Region --format="value(metadata.name)"
if ($services -contains "brainvolt-worker") {
    Write-Host "brainvolt-worker still exists. The API now runs its jobs, so delete it to stop its charges:" -ForegroundColor Yellow
    Write-Host "  gcloud run services delete brainvolt-worker --region $Region"
} else {
    Write-Host "no separate worker service"
}

Step "Done: $url"
Write-Host "Recent logs: gcloud run services logs read brainvolt-api --region $Region --limit 50"
