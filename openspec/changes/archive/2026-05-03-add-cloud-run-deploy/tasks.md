## 1. ブランチ作成と前提確認

- [x] 1.1 `main` から `feature/fro-114-cloud-run-deploy` ブランチを作成 (現状 `feature/fro-113-api-config-and-openapi-docs` にいる場合は先に `git fetch origin && git checkout main && git pull origin main`)
- [x] 1.2 `lefthook install` 済みであることを確認 (`ls .git/hooks/pre-commit` で確認)
- [x] 1.3 `gcloud --version` および `docker --version` がローカルで利用可能なことを確認 (Docker 29.4.1 / gcloud 566.0.0 確認済)

## 2. フェーズ 1: ボディサイズ既定値の変更

- [x] 2.1 `internal/api/config.go` の定数 `defaultBodyLimitBytes` を `int64(32 * 1024 * 1024)` に変更
- [x] 2.2 `configs/default.yaml` の `api.body_limit_bytes` を `33554432` に変更し、コメントに「Cloud Run HTTP/1 上限と整合」の理由を追記
- [x] 2.3 `internal/api/config_test.go` の既定値検証テスト (`TestDefaultConfig_AllFields` 等) の期待値を `33554432` に追従更新
- [x] 2.4 `internal/api/config_test.go` の YAML / 環境変数読み込みテストで `body_limit_bytes` を扱っているケースの期待値を追従更新 (override 値を直接指定する形のため実コード変更は不要、`TestLoadConfig_DefaultsWhenFileMissing` は `def.BodyLimitBytes` 参照で自動追従)
- [x] 2.5 `internal/api/middleware/bodylimit_test.go` の境界テスト値を 32 MiB 基準に追従更新 (現テストはパラメトリックに maxBytes を指定する形で既定値を直接参照していないため変更不要、境界ロジックは `exact_limit_plus_one` ケースで担保済)
- [x] 2.6 `goimports -w .` および `golangci-lint run ./...` を実行してエラー 0 を確認
- [x] 2.7 `go test -race ./internal/api/...` で全 pass を確認

## 3. フェーズ 1: コンテナ化

- [x] 3.1 リポジトリルートに `Dockerfile` を新規作成 (マルチステージ、`golang:1.25.6-bookworm` ビルド + `gcr.io/distroless/base-debian12:nonroot` ランタイム、BuildKit キャッシュマウント、`CGO_ENABLED=1`、`-trimpath -ldflags="-s -w"`、`configs/default.yaml` 同梱、`USER nonroot:nonroot`、`ENTRYPOINT ["/app/api"]`)
- [x] 3.2 リポジトリルートに `.dockerignore` を新規作成 (`build/`, `.git/`, `.github/`, `.serena/`, `.claude/`, `testdata/`, `cmd/img-cli/`, `cmd/tui-demo/`, `internal/cli/`, `internal/platform/`, `docs/`, `README.md`, `LICENSE`, `openspec/`, IDE/OS 関連を除外)
- [x] 3.3 リポジトリルートに `docker-compose.yml` を新規作成 (`build` で `Dockerfile` 参照、`8080:8080` ポートマッピング、`LOKI_API_PORT=8080` / `LOKI_API_HOST=0.0.0.0` / `LOKI_API_LOGGING_LEVEL=debug` 環境変数、distroless 制約により `healthcheck.disable: true`)
- [x] 3.4 `docker build -t loki-api:local .` が成功することを確認 (約10秒、distroless ランタイムでビルド成功)
- [x] 3.5 `docker run --rm -d --name loki-test -p 8080:8080 loki-api:local && sleep 2 && curl -fsS http://localhost:8080/api/v1/health && docker stop loki-test` で動作確認 (200 OK + `{"status":"ok"}` 返却)
- [x] 3.6 `docker compose up -d` および `curl -fsS http://localhost:8080/api/v1/health` で compose 起動確認、`docker compose down` で停止 (200 OK 確認)
- [x] 3.7 31 MiB ファイル (multipart overhead 考慮で 32 MiB ぴったりだと 413 になり境界が曖昧なため 31 MiB を使用) で `POST /api/v1/compress` → HTTP 422 (画像形式エラー、ボディ上限は通過) を確認
- [x] 3.8 33 MiB ファイルで `POST /api/v1/compress` → HTTP 413 + `{"max_bytes":33554432,"title":"Payload Too Large"}` のエラー応答を確認

## 4. フェーズ 1: ドキュメントとコミット

- [x] 4.1 `README.md` に Deployment 章の見出しを追加 (`docs/deployment/` への導線を残す形、詳細はフェーズ 2 で追記)
- [x] 4.2 `docker-compose.yml` の使い方 (主にローカル動作確認用、開発は `go run ./cmd/api`) を `README.md` の Development セクションに追記
- [x] 4.3 ここまでの変更を 1 コミット ("APIサーバのコンテナ化とボディ上限の Cloud Run 整合 (FRO-114)" 等) でコミット (lefthook の pre-commit が走り fmt/lint 通過を確認) (commit `e2703bf`)

## 5. フェーズ 2: GCP セットアップ手順の文書化

- [x] 5.1 `docs/deployment/` ディレクトリを新規作成
- [x] 5.2 `docs/deployment/gcp-setup.md` を新規作成し、以下を含める: 前提変数 (`GCP_PROJECT_ID` 等)、API 有効化 (`run.googleapis.com`, `artifactregistry.googleapis.com`, `iamcredentials.googleapis.com`, `sts.googleapis.com`)、Artifact Registry 作成、Service Account 作成と IAM ロール付与 (`roles/run.admin`, `roles/artifactregistry.writer`, `roles/iam.serviceAccountUser`)、Workload Identity Pool/Provider 作成 (`attribute-condition` でリポジトリ限定)、`add-iam-policy-binding` で `principalSet` 経由の二重防御、GitHub Secrets 登録手順 (`gh secret set` コマンド例)
- [x] 5.3 `docs/deployment/cloud-run.md` を新規作成し、以下を含める: サービス構成 (リージョン / インスタンス / リソース / タイムアウト / concurrency / 公開設定 / 環境変数)、初回手動デプロイコマンド、ロールバック手順 (`gcloud run services update-traffic --to-revisions=<PREV>=100`)、リビジョン一覧取得、ログ閲覧 (`gcloud logging tail` / `gcloud logging read`)、コスト保護 (請求アラート設定手順)
- [x] 5.4 `README.md` の Deployment 章に `docs/deployment/gcp-setup.md` と `docs/deployment/cloud-run.md` への導線を整備 (Phase 1 で先行追加済み)

## 6. フェーズ 2: GCP 初回セットアップ実施 (リポジトリ外作業)

- [x] 6.1 `docs/deployment/gcp-setup.md` 記載の手順を上から順に実行 (API 有効化、Artifact Registry、Service Account、WIF Pool/Provider、IAM 付与) (ユーザ手動実行済)
- [x] 6.2 GitHub Secrets に `GCP_PROJECT_ID`, `GCP_WIF_PROVIDER`, `GCP_DEPLOY_SA`, `LOKI_CORS_ORIGINS` を登録 (`LOKI_CORS_ORIGINS=https://tool.frontworks.dev,http://localhost:4321`) (ユーザ手動実行済)
- [x] 6.3 ローカルから `gcloud auth configure-docker asia-northeast1-docker.pkg.dev` を実行
- [x] 6.4 ローカルから手動でイメージをビルド・push し (`docker build --platform linux/amd64 -t asia-northeast1-docker.pkg.dev/$GCP_PROJECT_ID/loki/api:bootstrap . && docker push ...`)、`gcloud run deploy loki-api ...` (フルオプション) で初回デプロイを実行 (Apple Silicon の `--platform linux/amd64` 必要に気付いた経緯は commit `2e294ae` で文書化)
- [x] 6.5 `gcloud run services describe loki-api --region=asia-northeast1 --format='value(status.url)'` でサービス URL を取得し、`curl ${URL}/api/v1/health` が 200 を返すことを確認 (status:ok 確認済)
- [x] 6.6 `curl ${URL}/openapi.yaml` で OpenAPI スペックが配信されていることを確認
- [x] 6.7 `cd ../Lugh && echo "PUBLIC_API_BASE_URL=${URL}" > .env.local && bun run dev` で Lugh dev サーバを起動し、ブラウザから実際に画像変換が動作すること (CORS エラーがないこと) を確認 (GCP 側ログで Lugh dev サーバからのアクセスを確認済)
- [x] 6.8 `curl -i -X OPTIONS -H "Origin: http://localhost:4321" -H "Access-Control-Request-Method: POST" ${URL}/api/v1/convert` で CORS プリフライトに `Access-Control-Allow-Origin: http://localhost:4321` が返ることを確認 (CORS env 解析バグ修正後 commit `35dd805` で動作確認)
- [x] 6.9 `curl -i -X OPTIONS -H "Origin: https://tool.frontworks.dev" -H "Access-Control-Request-Method: POST" ${URL}/api/v1/convert` で本番ドメインからのプリフライトにも `Access-Control-Allow-Origin: https://tool.frontworks.dev` が返ることを確認 (同上)

## 7. フェーズ 3: GitHub Actions ワークフロー追加

- [x] 7.1 `.github/workflows/deploy.yml` を新規作成 (`on: push: branches: [main]` + `workflow_dispatch`、`concurrency: deploy-cloud-run`、`permissions: id-token: write contents: read`、`google-github-actions/auth@v2` で WIF 認証、`google-github-actions/setup-gcloud@v2`、`gcloud auth configure-docker`、`docker build/push` で `:${{ github.sha }}` と `:latest` の二系統タグ、`gcloud run deploy` フルオプション、CORS 環境変数は `--set-env-vars=^@@^LOKI_API_CORS_ALLOWED_ORIGINS=...` セパレータ構文で渡す。LOKI_CORS_ORIGINS は `env:` 経由で受けてシェルインジェクション耐性を確保)
- [x] 7.2 ローカルで `gh workflow view deploy.yml` (push 後) または `actionlint` 等でワークフローの構文確認 (Ruby YAML.load_file で構文 OK)
- [x] 7.3 ここまでの変更 (`docs/deployment/*`, `.github/workflows/deploy.yml`, README 更新) を 1 コミットでコミット ("Cloud Run デプロイワークフローと運用ドキュメント追加 (FRO-114)" 等) (commit `8bb9e42`)
- [ ] 7.4 リモートに push し、GitHub Actions の `Deploy to Cloud Run` ワークフローが他のワークフロー (test/build) と競合せず一覧に出ることを確認

## 8. フェーズ 3: CI 経由デプロイの動作確認

- [ ] 8.1 PR を作成し、main にマージ (またはマージ前に `gh workflow run deploy.yml --ref feature/fro-114-cloud-run-deploy` で `workflow_dispatch` 経由のドライラン)
- [ ] 8.2 `gh run watch` で Actions の進行を監視し、すべてのステップが緑になることを確認
- [ ] 8.3 `gcloud run revisions list --service=loki-api --region=asia-northeast1 --limit=3` で新リビジョンが生成されたことを確認
- [ ] 8.4 サービス URL に対する `/api/v1/health` および Lugh dev サーバからの fetch が依然として動作することを確認

## 9. フェーズ 3: ロールバックリハーサル

- [ ] 9.1 直前の正常リビジョン名を取得 (`gcloud run revisions list --service=loki-api --region=asia-northeast1 --format='value(metadata.name)' --limit=2`)
- [ ] 9.2 `gcloud run services update-traffic loki-api --to-revisions=<PREV>=100 --region=asia-northeast1` で旧リビジョンにトラフィックを 100% 戻す
- [ ] 9.3 サービス URL が依然として 200 を返すことを確認
- [ ] 9.4 `gcloud run services update-traffic loki-api --to-latest --region=asia-northeast1` で最新リビジョンに戻す
- [ ] 9.5 リハーサルで気付いた手順の不備があれば `docs/deployment/cloud-run.md` を補足

## 10. 最終検証とコミット・PR

- [x] 10.1 `golangci-lint run ./...` でエラー 0 を確認
- [x] 10.2 `go test -race ./...` で全 pass を確認 (CGO 由来の macOS リンカ警告は既知の表示で実害なし)
- [x] 10.3 `go build ./...` で全パッケージビルド成功を確認
- [x] 10.4 `openspec validate add-cloud-run-deploy --strict` (もし利用可能なら) で OpenSpec 整合性を確認
- [x] 10.5 不要な変更ファイル (`.serena/project.yml`, `coverage.out` 等) が含まれていないか `git status` で確認 (`.serena/project.yml` は自動生成ファイルとして 10.6 のコミットに含める方針)
- [x] 10.6 (リハーサル等で追加コミットがある場合) 残りの変更を日本語メッセージでコミット (commit `e6e3c22`)
- [ ] 10.7 リモートに push し、`gh pr create` で PR を作成 (タイトル例: "Cloud Run デプロイ整備 (FRO-114)"、本文に `docs/deployment/gcp-setup.md` の前提条件・GitHub Secrets 設定が完了済みであることを明記)
- [ ] 10.8 PR の Actions ジョブ (test, build, deploy) がすべて緑であることを確認

## 11. アーカイブ準備

- [ ] 11.1 PR マージ後、`/opsx:archive` 相当のフローで本変更を `openspec/changes/archive/<date>-add-cloud-run-deploy/` へ移動 (specs を `openspec/specs/deployment-cloud-run/spec.md` 新規作成、`openspec/specs/api-middleware/spec.md` の `body_limit_bytes` 既定値を 32 MiB 反映)
- [ ] 11.2 Linear FRO-114 を Done に変更
