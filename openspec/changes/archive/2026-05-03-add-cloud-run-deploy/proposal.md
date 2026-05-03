## Why

FRO-109〜113 で API サーバ・compress/convert エンドポイント・ミドルウェア・設定管理・OpenAPI ドキュメントが整い、本番運用の前提条件は満たされた。残るは「実際にどう本番にデプロイするか」の手段が未整備である点で、現状はローカルでしか起動できず、フロントエンド (`Lugh`, Astro on Cloudflare Pages) から叩ける公開 URL が存在しない。本変更で API サーバを Google Cloud Run にデプロイする経路 (Dockerfile / GCP セットアップ / GitHub Actions 自動デプロイ) を整備し、フロントエンドからの実利用を可能にする。あわせて Cloud Run の HTTP/1 リクエストサイズ上限 (32 MiB) と既存の `body_limit_bytes` 既定値 (50 MiB) の不整合を解消する。

## What Changes

- **コンテナ化**: マルチステージ `Dockerfile` を新設。ビルドステージは `golang:1.25.6-bookworm`、ランタイムは `gcr.io/distroless/base-debian12:nonroot` を採用し、`chai2010/webp` の CGO 依存 (libwebp / glibc) を満たしつつ最小サイズ・非 root 実行を担保する。BuildKit キャッシュマウントで `go mod download` と `go build` を高速化する。
- **`.dockerignore`**: ビルドコンテキストから `build/`・`testdata/`・`.git/`・CLI 関連 (`cmd/img-cli/`, `cmd/tui-demo/`, `internal/cli/`)・ドキュメントを除外し、API バイナリのみがイメージに入るようにする。
- **`docker-compose.yml`**: 「本番イメージとほぼ同等の環境でローカル起動できる」ことを担保する確認用 compose を追加する。ホットリロードは入れない (開発は引き続き `go run ./cmd/api`)。
- **GCP 事前セットアップ手順**: `docs/deployment/gcp-setup.md` を新設し、Artifact Registry 作成 / デプロイ用 Service Account 作成 / Workload Identity Federation (WIF) 設定 / GitHub Secrets 登録 / 初回手動デプロイの手順をコピペ実行可能な `gcloud` コマンドで網羅する。SA キー JSON は使わず WIF のみで認証する。
- **Cloud Run 運用ドキュメント**: `docs/deployment/cloud-run.md` を新設し、サービス構成 (リージョン `asia-northeast1`、`min=0/max=3`、`memory=512Mi`、`cpu=1`、`concurrency=10`、`timeout=60s`、`--allow-unauthenticated`)、環境変数注入方法、ロールバック手順、ログ閲覧コマンド、コスト保護 (請求アラート) の手順をまとめる。
- **CI 自動デプロイ**: `.github/workflows/deploy.yml` を新設し、`main` ブランチへの push および `workflow_dispatch` をトリガに、WIF 経由で `google-github-actions/auth@v2` → `setup-gcloud@v2` → `docker build/push` → `gcloud run deploy` を実行する。`concurrency` グループで同時デプロイを抑止し、イメージタグは `:${{ github.sha }}` と `:latest` の二系統で運用する。
- **API ボディサイズ既定値変更 (BREAKING な可能性は低いが既定挙動の変更)**: `api.body_limit_bytes` の既定値を 50 MiB → 32 MiB (33,554,432 バイト) に下げる。Cloud Run HTTP/1 のリクエスト上限と整合させ、ローカルとデプロイ環境で挙動差を出さないため。`configs/default.yaml` および `internal/api/config.go` の定数 `defaultBodyLimitBytes`、関連テスト (`internal/api/config_test.go`, `internal/api/middleware/bodylimit_test.go` の境界テスト) を追従更新する。
- **CORS 本番設定**: 本変更はコードを足さず、デプロイ時の環境変数 (`LOKI_API_CORS_ALLOWED_ORIGINS`) で `Lugh` の本番ドメインと dev サーバ (`http://localhost:4321`) のみを許可する形にする (`gcloud run deploy --set-env-vars=^@@^...`)。
- **README**: デプロイ手順の章を追記し、`docs/deployment/*.md` への導線を作る。

## Non-goals

- **独自ドメインの割り当て**: Cloud Run が発行する `https://loki-api-xxxxx.a.run.app` を直接利用する。Cloudflare DNS でのカスタムドメイン (例: `api.loki.example.com`) は将来チケットで扱う。
- **Terraform / Pulumi 等 IaC によるインフラ管理**: 個人プロジェクトの規模に対して過剰なため、初回 GCP セットアップは `docs/deployment/gcp-setup.md` の手順書ベースとする。
- **HTTP/2 (`--use-http2`) の有効化**: デフォルトの HTTP/1 で運用し、ボディ上限は 32 MiB に下げて整合させる。50 MiB 超のリクエストが必要になった時点で再検討する。
- **Cloud Run ランタイム専用 Service Account の最小権限化**: 既定の Compute Engine SA をランタイムに使う。本 API は GCP 側 API を一切叩かないため現時点で問題ない。GCS 等を使うときに別チケットで分離する。
- **画像処理エンジンの差し替え (libvips 等)**: スコープ外。`chai2010/webp` (CGO + libwebp) の現状構成を維持する。
- **Cloud Build 等 GCP 側 CI への移行**: GitHub Actions に統一する。
- **API 認証 (IAM / API キー)**: 本変更では `--allow-unauthenticated` で公開する。アクセス制御は CORS とアプリ側レートリミット (30 req/min/IP) に委ねる。サーバ間呼び出しが必要になった時点で別チケットで IAM 認証を追加する。
- **タグプッシュ・リリースブランチ等の高度なデプロイ戦略**: `main` への push と `workflow_dispatch` のみをデプロイトリガとする。

## Capabilities

### New Capabilities

- `deployment-cloud-run`: Cloud Run へのコンテナデプロイ全体に関する仕様。コンテナイメージの構成要件 (マルチステージ、非 root 実行、CGO ランタイム依存)、Cloud Run サービス設定要件 (PORT 注入、リソース割当、min/max インスタンス、タイムアウト、公開ポリシー)、CI/CD パイプライン要件 (WIF 認証、イメージタグ戦略、`concurrency` 制御、ロールバック容易性) を集約する。

### Modified Capabilities

- `api-middleware`: 「リクエストボディサイズ制限」要件のデフォルト値を 50 MiB → 32 MiB に変更する。設定キー (`api.body_limit_bytes`) と上書き挙動は維持。Cloud Run HTTP/1 上限 (32 MiB) と整合させ、ローカルとデプロイ環境で挙動差を出さないことを目的とする。

## Impact

- **コード**:
  - `internal/api/config.go`: `defaultBodyLimitBytes` 定数を `32 * 1024 * 1024` に変更
  - `internal/api/config_test.go`: 既定値・YAML 読込テストの期待値追従
  - `internal/api/middleware/bodylimit_test.go`: 境界テスト値の追従 (32 MiB ちょうどは通る、超過は 413)
- **設定**: `configs/default.yaml` の `api.body_limit_bytes` を `33554432` に変更、コメントに Cloud Run 整合の理由を追記
- **新規ファイル**:
  - `Dockerfile`
  - `.dockerignore`
  - `docker-compose.yml`
  - `.github/workflows/deploy.yml`
  - `docs/deployment/gcp-setup.md`
  - `docs/deployment/cloud-run.md`
- **依存**: 追加 Go モジュールなし。CI 側で `google-github-actions/auth@v2`, `google-github-actions/setup-gcloud@v2` を新規利用
- **GCP 側 (リポジトリ外)**: Artifact Registry リポジトリ (`asia-northeast1-docker.pkg.dev/{PROJECT}/loki`)、Service Account (`loki-deployer@...`)、Workload Identity Pool/Provider (`github` / `loki`)、Cloud Run サービス (`loki-api`)
- **GitHub Secrets**: `GCP_PROJECT_ID`, `GCP_WIF_PROVIDER`, `GCP_DEPLOY_SA`, `LOKI_CORS_ORIGINS`
- **後方互換**: `body_limit_bytes` の既定値変更は、明示的に 32〜50 MiB の範囲で `api.body_limit_bytes` 設定 / 環境変数 `LOKI_API_BODY_LIMIT_BYTES` を上書きしているクライアントにのみ影響。CLI には影響なし
- **ドキュメント**: `README.md` に Deployment 章追加、`docs/deployment/` 配下を新設
- **下流チケット**: 本変更完了後、独自ドメイン割当・カスタムドメイン HTTPS 化・運用 SA 最小権限化など派生タスクを別チケットで扱う想定
