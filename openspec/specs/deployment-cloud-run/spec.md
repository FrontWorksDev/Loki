# deployment-cloud-run Specification

## Purpose

Cloud Run へのコンテナデプロイ全体に関する仕様を集約する。コンテナイメージの構成要件 (マルチステージビルド、非 root 実行、CGO ランタイム依存)、Cloud Run サービス設定要件 (PORT 注入、リソース割当、min/max インスタンス、タイムアウト、公開ポリシー、CORS 環境変数注入)、CI/CD パイプライン要件 (Workload Identity Federation 認証、イメージタグ戦略、`concurrency` 制御)、ロールバック手順、三層コスト保護 (Cloud Run 上限 + アプリ側レートリミット + GCP 請求アラート) を含む。

## Requirements
### Requirement: マルチステージ Docker イメージのビルド

リポジトリは API サーバ (`./cmd/api`) を本番運用可能なコンテナイメージとしてビルドできる `Dockerfile` を提供しなければならない（MUST）。`Dockerfile` はマルチステージビルドを採用し、ビルドステージとランタイムステージを分離しなければならない（MUST）。ランタイムイメージにはビルドツール・ソースコード・テストデータを含めてはならない（MUST NOT）。`chai2010/webp` の CGO 依存 (libwebp / glibc) を満たすため、ランタイムイメージは glibc を含むベース (`gcr.io/distroless/base-debian12` 系) を使用しなければならない（MUST）。コンテナは非 root ユーザで実行されなければならない（MUST）。

#### Scenario: ビルドステージとランタイムステージの分離
- **WHEN** `docker build -t loki-api .` を実行する
- **THEN** ビルドが成功し、生成されるイメージのレイヤにビルドツール (Go コンパイラ、`apt` 等) が含まれていないこと

#### Scenario: 非 root 実行の保証
- **WHEN** ビルドしたイメージを `docker run --rm loki-api id` (相当の検証) で起動する
- **THEN** プロセスの uid が 0 (root) ではなく、非 root ユーザ (`nonroot` / 65532 等) であること

#### Scenario: WebP CGO 依存の動作
- **WHEN** ビルドしたコンテナを起動し、`POST /api/v1/convert?format=webp` で実画像を投げる
- **THEN** WebP 形式の画像が返却される (libwebp との CGO リンクが正しく解決されている)

### Requirement: ビルドコンテキストからの除外

リポジトリは `.dockerignore` を提供し、Docker ビルドコンテキストから不要ファイルを除外しなければならない（MUST）。少なくとも以下のパスは除外対象とする（MUST）: `build/`, `.git/`, `.github/`, `testdata/`, `cmd/img-cli/`, `cmd/tui-demo/`, `internal/cli/`, `internal/platform/`, ドキュメント (`docs/`, `README.md`, `LICENSE`)。

#### Scenario: API バイナリのみをイメージに含む
- **WHEN** `docker build .` を実行する
- **THEN** イメージ内に CLI / TUI 関連のソース・テストデータが含まれていないこと

#### Scenario: ビルドコンテキスト送信サイズの最小化
- **WHEN** `docker build .` を実行する
- **THEN** `Sending build context to Docker daemon` のサイズが、リポジトリ全体のサイズより明確に小さいこと (`testdata/` が除外されていることが体感できる規模)

### Requirement: ローカル動作確認用 docker-compose

リポジトリは `docker-compose.yml` を提供し、本番イメージとほぼ同等の環境で API サーバをローカル起動できるようにしなければならない（MUST）。compose は `Dockerfile` を `build` 指定で参照し、ホスト 8080 番ポートにマッピングしなければならない（MUST）。ホットリロードは導入してはならない（MUST NOT、開発用途には `go run ./cmd/api` を使う方針）。

#### Scenario: compose 起動とヘルスチェック
- **WHEN** `docker compose up -d` を実行し、起動後 5 秒以内に `curl http://localhost:8080/api/v1/health` を叩く
- **THEN** ステータス 200 と `{"status":"ok"}` が返ること

#### Scenario: 環境変数による設定上書き
- **WHEN** compose 内で `LOKI_API_LOGGING_LEVEL=debug` を設定して起動する
- **THEN** API サーバの起動ログに debug レベルが反映されていること

### Requirement: Cloud Run の PORT 注入対応

API サーバは Cloud Run が注入する `$PORT` 環境変数に従って listen しなければならない（MUST）。コンテナ起動時に `LOKI_API_PORT` 環境変数で listen ポートを指定可能でなければならない（MUST）。本要件はコード変更ではなく、デプロイ時に `--set-env-vars=LOKI_API_PORT=8080` および `--port=8080` を併せて指定することで満たす。

#### Scenario: 任意のポートで起動
- **WHEN** コンテナを `docker run -e LOKI_API_PORT=9000 -p 9000:9000 loki-api` で起動する
- **THEN** API サーバは 9000 番ポートで listen し、`curl http://localhost:9000/api/v1/health` が 200 を返すこと

#### Scenario: Cloud Run 既定の 8080 番
- **WHEN** Cloud Run にデプロイし、`--port=8080` および `--set-env-vars=LOKI_API_PORT=8080` を指定する
- **THEN** Cloud Run のサービス URL に対するリクエストが API サーバに到達すること

### Requirement: GCP リソースの事前セットアップ手順

リポジトリは Cloud Run デプロイに必要な GCP リソース (Artifact Registry / Service Account / Workload Identity Federation / IAM 権限付与) をセットアップする手順を `docs/deployment/gcp-setup.md` として提供しなければならない（MUST）。手順は `gcloud` コマンドのみで実行可能で、コピペで上から順に実行できなければならない（MUST）。Service Account キー JSON は使用してはならない（MUST NOT）。

#### Scenario: WIF プロバイダ作成
- **WHEN** ドキュメント記載の `gcloud iam workload-identity-pools providers create-oidc` を実行する
- **THEN** `attribute-condition` で `assertion.repository == 'FrontWorksDev/Loki'` がセットされ、他リポジトリからのトークン交換が拒否されること

#### Scenario: Service Account への最小権限付与
- **WHEN** デプロイ用 SA `loki-deployer` に IAM ロールを付与する
- **THEN** 付与されるロールが `roles/run.admin`, `roles/artifactregistry.writer`, `roles/iam.serviceAccountUser` に限定されていること (Owner / Editor 等の広範ロールを含まない)

#### Scenario: 二重防御の WIF バインディング
- **WHEN** SA への `roles/iam.workloadIdentityUser` 付与で `principalSet://...` を `attribute.repository/FrontWorksDev/Loki` に絞る
- **THEN** 他リポジトリの GitHub Actions が同 WIF Provider を経由しても、当該 SA を impersonate できないこと

### Requirement: Cloud Run サービスのデプロイ構成

`docs/deployment/cloud-run.md` および `gcloud run deploy` コマンドは、本番 Cloud Run サービスを以下の既定構成でデプロイしなければならない（MUST）: リージョン `asia-northeast1`、`--allow-unauthenticated`、`--port=8080`、`--memory=512Mi`、`--cpu=1`、`--min-instances=0`、`--max-instances=3`、`--concurrency=10`、`--timeout=60s`。CORS allowed_origins は環境変数 `LOKI_API_CORS_ALLOWED_ORIGINS` で本番ドメインと dev サーバを許可しなければならない（MUST）。

#### Scenario: コスト上限の物理的担保
- **WHEN** Cloud Run サービスを上記構成でデプロイし、突発的な大量リクエストが到来する
- **THEN** インスタンス数は `max-instances=3` を超えず、それ以上はキューイング/拒否される

#### Scenario: アイドル時の課金ゼロ
- **WHEN** トラフィックが一定時間ない状態が続く
- **THEN** Cloud Run はインスタンスをゼロにスケールダウンし、コンピュート課金が発生しないこと (`min-instances=0` の効果)

#### Scenario: CORS 制限の本番反映
- **WHEN** `LOKI_API_CORS_ALLOWED_ORIGINS=https://tool.frontworks.dev,http://localhost:4321` を環境変数として注入してデプロイする
- **THEN** `Origin: https://other.example` からのプリフライトには `Access-Control-Allow-Origin` が付与されないこと、`Origin: https://tool.frontworks.dev` および `Origin: http://localhost:4321` には許可ヘッダーが付与されること

### Requirement: GitHub Actions による自動デプロイ

リポジトリは `.github/workflows/deploy.yml` を提供し、`main` ブランチへの push および `workflow_dispatch` をトリガに、WIF 経由で認証して Cloud Run へ自動デプロイしなければならない（MUST）。同時実行は `concurrency` グループで抑止されなければならない（MUST）。デプロイステップは冪等でなければならない（MUST）。Service Account キー JSON は workflow から参照してはならない（MUST NOT）。

#### Scenario: main マージで自動デプロイ
- **WHEN** PR を main にマージし、`deploy.yml` がトリガされる
- **THEN** WIF 認証 → Docker ビルド → Artifact Registry へ push → `gcloud run deploy` の各ステップが順次成功し、新しい Cloud Run リビジョンが昇格すること

#### Scenario: 同時デプロイの抑止
- **WHEN** 短時間に 2 回連続で main にマージが入る
- **THEN** 2 件目のワークフロー実行は 1 件目の完了を待ってから開始される (`concurrency.group=deploy-cloud-run` の効果)

#### Scenario: イメージタグの二系統運用
- **WHEN** `deploy.yml` がイメージを push する
- **THEN** Artifact Registry に `:${{ github.sha }}` タグと `:latest` タグの両方が存在すること

### Requirement: ロールバック手順の提供

`docs/deployment/cloud-run.md` は、Cloud Run の特定リビジョンへ即時に 100% トラフィックを戻すロールバック手順を提供しなければならない（MUST）。手順は単一の `gcloud run services update-traffic` コマンドで完結しなければならない（MUST）。

#### Scenario: 直前リビジョンへの即時切り戻し
- **WHEN** 新リビジョンに不具合があり `gcloud run services update-traffic loki-api --to-revisions=<PREV>=100 --region=asia-northeast1` を実行する
- **THEN** トラフィックが 100% 旧リビジョンに戻り、サービス URL が以前の挙動に復帰すること

#### Scenario: 最新リビジョンへの復帰
- **WHEN** ロールバック後、修正版をデプロイした後に `--to-latest` で戻す
- **THEN** 最新リビジョンに 100% トラフィックが戻ること

### Requirement: コスト保護の三層防御

本変更は以下の三層でコスト暴走を防がなければならない（MUST）: (1) Cloud Run 側の `--max-instances`, `--concurrency`, `--timeout`、(2) アプリ側の IP ベースレートリミット (既存 30 req/min/IP)、(3) GCP 請求アラート設定手順の文書化。請求アラートの設定手順は `docs/deployment/cloud-run.md` に記載しなければならない（MUST）。

#### Scenario: 全層が機能した場合の上限
- **WHEN** 単一 IP から大量のリクエストが投げられる
- **THEN** アプリ側レートリミットで 30 req/min を超えるリクエストが 429 で拒否され、Cloud Run のインスタンス数も `--max-instances=3` を超えないこと

#### Scenario: 請求アラートの設定手順記載
- **WHEN** `docs/deployment/cloud-run.md` を参照する
- **THEN** GCP コンソールでの予算アラート設定手順 (Billing → Budgets & alerts) が記載されていること

