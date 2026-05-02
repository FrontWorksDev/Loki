# Cloud Run 運用ドキュメント

Loki API の Cloud Run サービスの構成・初回デプロイ・運用 (ロールバック / ログ閲覧 / コスト保護) をまとめたドキュメント。GCP 側初回セットアップは [`gcp-setup.md`](gcp-setup.md) を参照。

## サービス構成

| 項目 | 値 | 設定理由 |
|---|---|---|
| プロジェクト | `$GCP_PROJECT_ID` (Secrets 参照) | デプロイ先 |
| サービス名 | `loki-api` | |
| リージョン | `asia-northeast1` | 国内利用前提でレイテンシ最小 |
| 公開設定 | `--allow-unauthenticated` | フロントエンド (Astro 静的サイト) からのブラウザ直叩きを許容。アクセス制御は CORS + アプリ側レートリミット |
| ポート | `8080` | コンテナの `LOKI_API_PORT` と整合 |
| メモリ | `512Mi` | 32 MiB リクエスト × 数枚並行処理を想定した最小構成。OOM が出たら 1Gi に上げる |
| CPU | `1` vCPU | 画像処理が CPU バウンドなため。並列度を上げる場合は 2 vCPU 検討 |
| min-instances | `0` | アイドル時の課金ゼロ。Go のコールドスタートは数秒で許容範囲 |
| max-instances | `3` | コスト暴走の物理上限 |
| concurrency | `10` | 1 インスタンスあたり同時処理数。画像処理メモリ負荷を考慮し控えめ |
| timeout | `60s` | 画像処理が長引いたケースに備える保守的な値 |

## 環境変数

CI から `gcloud run deploy --set-env-vars` で注入する。

| 環境変数 | 値 | 用途 |
|---|---|---|
| `LOKI_API_PORT` | `8080` | API サーバの listen ポート (Cloud Run の `--port=8080` と整合) |
| `LOKI_API_HOST` | `0.0.0.0` | リッスンアドレス |
| `LOKI_API_LOGGING_LEVEL` | `info` | ログレベル (debug/info/warn/error) |
| `LOKI_API_CORS_ALLOWED_ORIGINS` | `https://tool.frontworks.dev,http://localhost:4321` | CORS 許可オリジン (本番ドメイン + dev サーバ) |

カンマを含む値の `--set-env-vars` 渡しには `^@@^` セパレータ構文を使う ([`.github/workflows/deploy.yml`](../../.github/workflows/deploy.yml) 参照)。

## 初回手動デプロイ

CI に乗せる前にローカルから 1 回デプロイして疎通確認する。事前に [`gcp-setup.md`](gcp-setup.md) の 1〜5 を完了しておくこと。

```bash
# 0. 環境変数 (gcp-setup.md と同じ)
export GCP_PROJECT_ID="your-project-id"
export GCP_REGION="asia-northeast1"
export AR_REPO="loki"
export SERVICE="loki-api"

# 1. AR への docker push 認証
gcloud auth configure-docker "${GCP_REGION}-docker.pkg.dev"

# 2. ビルド & push
IMAGE_URI="${GCP_REGION}-docker.pkg.dev/${GCP_PROJECT_ID}/${AR_REPO}/api:bootstrap"
docker build -t "$IMAGE_URI" .
docker push "$IMAGE_URI"

# 3. Cloud Run へデプロイ
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

# 4. サービス URL 取得
SERVICE_URL=$(gcloud run services describe "$SERVICE" --region="$GCP_REGION" --format='value(status.url)' --project="$GCP_PROJECT_ID")
echo "Service URL: $SERVICE_URL"

# 5. 疎通確認
curl "${SERVICE_URL}/api/v1/health"
curl "${SERVICE_URL}/openapi.yaml" | head -20

# 6. CORS プリフライト確認
curl -i -X OPTIONS \
  -H "Origin: https://tool.frontworks.dev" \
  -H "Access-Control-Request-Method: POST" \
  "${SERVICE_URL}/api/v1/convert"
```

## 環境変数のみの更新

コード変更なしで CORS や ログレベルを変えたい場合は、再デプロイせず `update` コマンドで反映できる。

```bash
# 単一の値を追加・変更
gcloud run services update "$SERVICE" \
  --region="$GCP_REGION" \
  --update-env-vars="LOKI_API_LOGGING_LEVEL=debug" \
  --project="$GCP_PROJECT_ID"

# CORS 許可オリジンの差し替え (カンマ含むためセパレータ指定)
gcloud run services update "$SERVICE" \
  --region="$GCP_REGION" \
  --update-env-vars="^@@^LOKI_API_CORS_ALLOWED_ORIGINS=https://tool.frontworks.dev,http://localhost:4321,https://staging.frontworks.dev" \
  --project="$GCP_PROJECT_ID"
```

## 🚨 緊急時のロールバック

新リビジョンに不具合が出た場合、以下のコマンド 1 発で直前の正常リビジョンに戻す。

### 1. 直近のリビジョン一覧を取得

```bash
gcloud run revisions list \
  --service=loki-api --region=asia-northeast1 \
  --format='table(metadata.name,status.conditions[0].status,metadata.creationTimestamp)' \
  --project="$GCP_PROJECT_ID" \
  --limit=5
```

### 2. 直前のリビジョンに 100% 戻す

```bash
gcloud run services update-traffic loki-api \
  --to-revisions=<PREVIOUS_REVISION_NAME>=100 \
  --region=asia-northeast1 \
  --project="$GCP_PROJECT_ID"
```

### 3. 復帰確認後、最新リビジョンに戻す (修正版デプロイ後)

```bash
gcloud run services update-traffic loki-api \
  --to-latest \
  --region=asia-northeast1 \
  --project="$GCP_PROJECT_ID"
```

## ログ閲覧

slog の JSON 構造化ログは Cloud Logging に自動収集される。

```bash
# 直近のログをストリーミング
gcloud logging tail \
  "resource.type=cloud_run_revision AND resource.labels.service_name=loki-api" \
  --project="$GCP_PROJECT_ID"

# エラーログのみ最新 50 件
gcloud logging read \
  'resource.type=cloud_run_revision AND resource.labels.service_name=loki-api AND severity>=ERROR' \
  --limit=50 --project="$GCP_PROJECT_ID"

# 特定のリビジョンに絞る
gcloud logging read \
  'resource.type=cloud_run_revision AND resource.labels.revision_name=loki-api-00042-xyz' \
  --limit=100 --project="$GCP_PROJECT_ID"
```

## コスト保護 (請求アラート)

Cloud Run 側の `--max-instances=3` + `--concurrency=10` + アプリ側レートリミット (30 req/min/IP) に加え、**GCP 請求アラート** を必ず設定する。

### 1. 予算アラート設定 (GCP コンソール)

1. https://console.cloud.google.com/billing → 該当 Billing Account を選択
2. 左メニュー **Budgets & alerts** → **CREATE BUDGET**
3. Scope: 該当プロジェクトを選択
4. Amount: 月額の上限を設定 (個人プロジェクトなら **$5〜$10** 推奨)
5. Actions: 通知メールアドレス + 50% / 90% / 100% で通知
6. (任意) Pub/Sub トピックを設定すれば自動停止スクリプトと連動可能

### 2. CLI で参照する場合

```bash
# 直近 1 ヶ月の Cloud Run コスト概算 (Billing API が必要、要権限)
gcloud billing budgets list --billing-account=<ACCOUNT_ID> 2>&1 | head
```

請求アラートは `gcloud` よりも **コンソール UI のほうが確実かつ簡単**。

## トラブルシュート

| 症状 | 原因の見立て | 対処 |
|---|---|---|
| デプロイ後に 503 | コンテナ起動失敗 (PORT 不整合 / クラッシュ) | `gcloud run revisions describe <REV>` でログ確認、`/api/v1/health` を直接叩く |
| 502 が散発 | OOM (画像サイズ × 並列数がメモリを超過) | `--memory=1Gi` に上げる、または `--concurrency=5` に下げる |
| CORS エラー (ブラウザ) | `LOKI_API_CORS_ALLOWED_ORIGINS` がドメイン不一致 | `gcloud run services update --update-env-vars=^@@^...` で即時修正 |
| 429 が頻出 | アプリ側レートリミット (既定 30 req/min/IP) に引っかかっている | クライアント側の呼び出し間隔を見直すか、`LOKI_API_RATE_LIMIT_REQUESTS_PER_MINUTE` で緩和 |
| コールドスタート遅延 | `min-instances=0` の設計通り | 許容できなければ `min-instances=1` (常時 1 インスタンス課金が発生) |
| デプロイが進まない | 同時デプロイがキューイングされている | `concurrency: deploy-cloud-run` の前ジョブ完了を待つ |
