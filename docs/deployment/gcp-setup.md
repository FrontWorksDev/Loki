# GCP 初回セットアップ手順

Loki API を Google Cloud Run にデプロイするための GCP 側セットアップを 1 回限りで完結させる手順書。CI 自動デプロイ ([`.github/workflows/deploy.yml`](../../.github/workflows/deploy.yml)) と運用 ([`cloud-run.md`](cloud-run.md)) の前提条件を整える。

## 前提

- `gcloud` CLI がインストール済み (`gcloud --version` で確認)
- `gh` CLI がインストール済み (`gh --version` で確認)
- 操作対象の GCP プロジェクトの **オーナーまたはプロジェクト IAM 管理者** ロールを持っていること
- リポジトリへの **Settings 権限** (Secrets 登録のため)

## 0. 前提変数の定義

シェルセッションの先頭で実行。後続のコマンドはこれらを参照する。

```bash
export GCP_PROJECT_ID="your-project-id"
export GCP_PROJECT_NUM=$(gcloud projects describe "$GCP_PROJECT_ID" --format='value(projectNumber)')
export GCP_REGION="asia-northeast1"
export GITHUB_OWNER="FrontWorksDev"
export GITHUB_REPO="Loki"
export AR_REPO="loki"
export SERVICE="loki-api"
export DEPLOY_SA="loki-deployer"
```

## 1. 必要 API の有効化

Cloud Run / Artifact Registry / IAM Credentials / STS の 4 つを有効化する。

```bash
gcloud services enable \
  run.googleapis.com \
  artifactregistry.googleapis.com \
  iamcredentials.googleapis.com \
  sts.googleapis.com \
  --project="$GCP_PROJECT_ID"
```

## 2. Artifact Registry リポジトリ作成

Docker イメージの保管先を `asia-northeast1-docker.pkg.dev/$GCP_PROJECT_ID/loki` として作成。

```bash
gcloud artifacts repositories create "$AR_REPO" \
  --repository-format=docker \
  --location="$GCP_REGION" \
  --description="Loki Docker images" \
  --project="$GCP_PROJECT_ID"
```

## 3. デプロイ用 Service Account の作成と最小権限付与

CI から impersonate する SA を作成し、デプロイに必要な最小権限のみ付与する。

```bash
# SA 作成
gcloud iam service-accounts create "$DEPLOY_SA" \
  --display-name="Loki Cloud Run Deployer" \
  --project="$GCP_PROJECT_ID"

DEPLOY_SA_EMAIL="${DEPLOY_SA}@${GCP_PROJECT_ID}.iam.gserviceaccount.com"

# 必要な最小ロールを付与
# - roles/run.admin: Cloud Run サービスのデプロイ・更新
# - roles/artifactregistry.writer: Docker イメージの push
# - roles/iam.serviceAccountUser: Cloud Run のランタイム SA を actAs できるように
for ROLE in roles/run.admin roles/artifactregistry.writer roles/iam.serviceAccountUser; do
  gcloud projects add-iam-policy-binding "$GCP_PROJECT_ID" \
    --member="serviceAccount:${DEPLOY_SA_EMAIL}" \
    --role="$ROLE"
done
```

`iam.serviceAccountUser` は、Cloud Run サービスのランタイムが使う Compute Engine デフォルト SA (`${GCP_PROJECT_NUM}-compute@developer.gserviceaccount.com`) に対して `actAs` する権限。本変更時点では Cloud Run のランタイム SA を分離していないため、この粒度で付与している。

## 4. Workload Identity Federation セットアップ

GitHub Actions の OIDC トークンを GCP の短命認証情報に交換する仕組み。**Service Account キー JSON の発行・配布は行わない**。

### 4.1 Workload Identity Pool 作成

```bash
gcloud iam workload-identity-pools create "github" \
  --location="global" \
  --display-name="GitHub Actions Pool" \
  --project="$GCP_PROJECT_ID"
```

### 4.2 OIDC プロバイダ作成

`attribute-condition` でリポジトリを `FrontWorksDev/Loki` に限定する。他リポジトリからのトークン交換は GCP 側で拒否される。

```bash
gcloud iam workload-identity-pools providers create-oidc "loki" \
  --location="global" \
  --workload-identity-pool="github" \
  --display-name="GitHub OIDC Provider for Loki" \
  --issuer-uri="https://token.actions.githubusercontent.com" \
  --attribute-mapping="google.subject=assertion.sub,attribute.repository=assertion.repository,attribute.ref=assertion.ref" \
  --attribute-condition="assertion.repository == '${GITHUB_OWNER}/${GITHUB_REPO}'" \
  --project="$GCP_PROJECT_ID"
```

### 4.3 SA を impersonate できるリポジトリを限定

`principalSet` でリポジトリを `FrontWorksDev/Loki` に絞る。**4.2 の attribute-condition と合わせて二重防御**。

```bash
gcloud iam service-accounts add-iam-policy-binding "$DEPLOY_SA_EMAIL" \
  --role="roles/iam.workloadIdentityUser" \
  --member="principalSet://iam.googleapis.com/projects/${GCP_PROJECT_NUM}/locations/global/workloadIdentityPools/github/attribute.repository/${GITHUB_OWNER}/${GITHUB_REPO}" \
  --project="$GCP_PROJECT_ID"
```

将来 `refs/heads/main` のみに絞りたい場合は `principalSet://.../attribute.ref/refs/heads/main` に変更可能。本構成では `workflow_dispatch` 経由の手動デプロイ (任意ブランチ) も許容するため `attribute.repository` で絞っている。

### 4.4 GitHub Actions に渡す値を出力

```bash
echo "GCP_PROJECT_ID=${GCP_PROJECT_ID}"
echo "GCP_WIF_PROVIDER=projects/${GCP_PROJECT_NUM}/locations/global/workloadIdentityPools/github/providers/loki"
echo "GCP_DEPLOY_SA=${DEPLOY_SA_EMAIL}"
```

## 5. GitHub Secrets 登録

リポジトリのルートで `gh` CLI を使って登録 (Web UI なら Settings → Secrets and variables → Actions)。

```bash
gh secret set GCP_PROJECT_ID --body "$GCP_PROJECT_ID"
gh secret set GCP_WIF_PROVIDER --body "projects/${GCP_PROJECT_NUM}/locations/global/workloadIdentityPools/github/providers/loki"
gh secret set GCP_DEPLOY_SA --body "$DEPLOY_SA_EMAIL"
gh secret set LOKI_CORS_ORIGINS --body "https://tool.frontworks.dev,http://localhost:4321"
```

| Secret | 値の意味 |
|---|---|
| `GCP_PROJECT_ID` | デプロイ先の GCP プロジェクト ID |
| `GCP_WIF_PROVIDER` | OIDC プロバイダのフルリソース名 |
| `GCP_DEPLOY_SA` | impersonate 対象の Service Account メール |
| `LOKI_CORS_ORIGINS` | API 側の CORS 許可オリジン (Lugh 本番ドメイン + dev サーバ) |

## 6. 初回手動デプロイ (疎通確認)

CI に乗せる前にローカルから 1 回デプロイし、後述の URL から `/api/v1/health` が叩けることを確認する。詳細手順は [`cloud-run.md`](cloud-run.md#初回手動デプロイ) を参照。

```bash
# AR への docker push 認証
gcloud auth configure-docker "${GCP_REGION}-docker.pkg.dev"

# ビルド & push
# ⚠️ Apple Silicon Mac の場合は --platform linux/amd64 必須 (Cloud Run は amd64 のみ)
IMAGE_URI="${GCP_REGION}-docker.pkg.dev/${GCP_PROJECT_ID}/${AR_REPO}/api:bootstrap"
docker build --platform linux/amd64 -t "$IMAGE_URI" .
docker push "$IMAGE_URI"

# Cloud Run へデプロイ (本番と同じオプション)
gcloud run deploy "$SERVICE" \
  --image="$IMAGE_URI" \
  --region="$GCP_REGION" \
  --allow-unauthenticated \
  --port=8080 \
  --memory=512Mi --cpu=1 \
  --min-instances=0 --max-instances=3 \
  --concurrency=10 --timeout=60s \
  --set-env-vars=LOKI_API_PORT=8080,LOKI_API_HOST=0.0.0.0,LOKI_API_LOGGING_LEVEL=info \
  --set-env-vars="^@@^LOKI_API_CORS_ALLOWED_ORIGINS=https://tool.frontworks.dev,http://localhost:4321" \
  --project="$GCP_PROJECT_ID"

# サービス URL 取得 + health チェック
SERVICE_URL=$(gcloud run services describe "$SERVICE" --region="$GCP_REGION" --format='value(status.url)' --project="$GCP_PROJECT_ID")
curl "${SERVICE_URL}/api/v1/health"
```

## 7. 既知の制約と注意点

- **Cloud Run のランタイム SA**: 何も指定しないと Compute Engine デフォルト SA が使われる。本 API は GCP API を一切叩かないため現時点で問題ないが、GCS 等を使うときには専用 SA を作って `--service-account` で指定すること。
- **コスト監視**: `--max-instances=3` で物理上限を設定済みだが、追加の安全網として GCP 請求アラートを設定することを推奨。手順は [`cloud-run.md`](cloud-run.md#コスト保護請求アラート) を参照。
- **WIF Provider の作り直し**: `--issuer-uri` などを変えたい場合、既存 Provider を削除する必要がある。誤って削除すると CI からのデプロイが即座に止まる点に注意。

## トラブルシュート

| 症状 | 原因の見立て | 対処 |
|---|---|---|
| `gcloud iam workload-identity-pools providers create-oidc` で `INVALID_ARGUMENT` | `attribute-condition` の構文ミス、Pool が未作成 | Pool 作成 (4.1) を先に実行、condition 文字列のクォートを再確認 |
| GitHub Actions の `auth` ステップで `Permission 'iam.serviceAccounts.getAccessToken' denied` | `roles/iam.workloadIdentityUser` の `principalSet` がリポジトリと不一致 | 4.3 の `add-iam-policy-binding` を再確認 (`${GITHUB_OWNER}/${GITHUB_REPO}` が正しいか) |
| `gcloud run deploy` で `User does not have permission` | `roles/run.admin` または `roles/iam.serviceAccountUser` 未付与 | 3 のロール付与を再確認 |
| `docker push` で `denied: Permission` | `roles/artifactregistry.writer` 未付与、または `gcloud auth configure-docker` 未実行 | 6 のコマンド再実行 |
| `gcloud run deploy` で `Container manifest type ... must support amd64/linux` | Apple Silicon Mac でビルドしたマルチプラットフォーム OCI image index に amd64 が含まれない | `docker build --platform linux/amd64 -t ... .` で再ビルド・push (詳細: [`cloud-run.md`](cloud-run.md#トラブルシュート)) |
