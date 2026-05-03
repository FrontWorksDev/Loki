## Context

FRO-113 で API サーバの設定管理 (Viper, `LOKI_API_*` 環境変数) と OpenAPI ドキュメントが整い、本番運用の前提条件が揃った。本変更ではこの API サーバを Google Cloud Run にデプロイする経路を整備する。

**現状**: API サーバはローカルで `go run ./cmd/api` でしか起動できない。フロントエンドの `Lugh` (Astro on Cloudflare Pages、`PUBLIC_API_BASE_URL` 経由でブラウザから直接 fetch する完全クライアントサイド構成) から叩ける公開 URL が存在しない。

**主要な技術制約**:
- `chai2010/webp` は libwebp に CGO リンクするため `CGO_ENABLED=1` 必須。`distroless/static` は使えず、glibc を含むランタイムベースが必要。
- Cloud Run は HTTP/1 でリクエストボディ上限 32 MiB を強制する。既存の `body_limit_bytes` 既定値 50 MiB と不整合。
- Cloud Run は `$PORT` 環境変数を注入し、コンテナはこれを listen する必要がある。
- フロントエンドが完全クライアントサイドのため、IAM 認証 (Google ID トークン要求) は実用不可。CORS とアプリ側レートリミットでアクセス制御する。

**ステークホルダー**: Loki API のデプロイ運用者 (= 本リポジトリのコントリビュータ)、`Lugh` フロントエンドの利用者 (= 不特定多数のブラウザ訪問者)。

## Goals / Non-Goals

**Goals:**

- API サーバの本番イメージを Docker でビルドでき、`docker compose up` でローカルでも起動確認できる。
- GCP プロジェクトに対する初回 1 回限りのセットアップ手順 (Artifact Registry / Service Account / Workload Identity Federation) を、コピペ実行可能な `gcloud` コマンドとして文書化する。
- `main` ブランチへの push をトリガに、GitHub Actions が WIF 認証で Cloud Run へ自動デプロイする。
- `Lugh` の dev サーバおよび本番ドメインから CORS で許可された fetch が成功する。
- リクエストボディサイズの既定値を Cloud Run HTTP/1 上限と整合させ、ローカルとデプロイ環境で挙動差を出さない。
- 障害時のロールバックが手動 1 コマンドで実行可能。

**Non-Goals:**

- Terraform / Pulumi 等 IaC によるインフラ管理 (本プロジェクトの規模に対して過剰)。
- 独自ドメイン割当・カスタム HTTPS 化 (Cloud Run 発行 URL を直接利用)。
- `--use-http2` 有効化 (本変更ではボディ上限を 32 MiB に下げて HTTP/1 で運用)。
- Cloud Run ランタイム専用 SA の最小権限化 (本 API は GCP API を叩かないため既定 SA で良い)。
- IAM 認証 / API キーによるアクセス制御 (CORS + レートリミットに委ねる)。
- Cloud Build 等 GCP 側 CI への移行 (GitHub Actions に統一)。

## Decisions

### D1: コンテナベースイメージの選択

**決定**: ビルドステージ `golang:1.25.6-bookworm`、ランタイムステージ `gcr.io/distroless/base-debian12:nonroot` を採用する。

**理由**:
- `chai2010/webp` の CGO 依存により `distroless/static` (libc なし) は使用不可。`distroless/base-debian12` は glibc を含むため動作する。
- ビルドステージとランタイムステージで Debian 12 (bookworm) に揃え、glibc バージョンの不整合を回避する。
- `:nonroot` タグを使うことで既定で UID 65532 で実行され、Cloud Run のセキュリティ推奨に合致。

**代替案**:
- `alpine` (musl libc): `chai2010/webp` の CGO ビルドを musl で通すには追加調整が必要。最小サイズの利点はあるが運用上の利得が小さい。
- `debian:bookworm-slim`: shell やパッケージマネージャを含むため攻撃面が広く、distroless の方が望ましい。
- `distroless/cc`: glibc + libgcc + libstdc++ を含む。今回は libstdc++ 不要のため `base` で足りる。

### D2: PORT 注入方式

**決定**: コードに変更を加えず、デプロイ時の環境変数で `LOKI_API_PORT=$PORT` 相当をセットする。

**理由**:
- 既存の Viper 設定は `LOKI_API_PORT` で読み込み可能。Cloud Run 側で `--port=8080` を指定し、コンテナ内では `LOKI_API_PORT=8080` 固定で動作させる。
- コードに `os.Getenv("PORT")` 分岐を入れると、ローカル / Cloud Run / 設定ファイルの三系統が混ざり可読性を損ねる。
- 将来 Cloud Run 以外のランタイム (Cloud Functions 等) を使う場合も、デプロイレイヤで環境変数をマッピングするだけで済む。

**代替案**:
- コードで `PORT` 環境変数を見るフォールバックを追加: 小さな複雑性追加だが、Viper の優先順位に新たな層を作ることになり、テスト負担が増える。

### D3: リクエストボディサイズ既定値の変更

**決定**: `defaultBodyLimitBytes` を 50 MiB → 32 MiB (33,554,432 バイト) に下げる。

**理由**:
- Cloud Run HTTP/1 のリクエスト上限は 32 MiB。既定 50 MiB のままだと、ローカルで通るリクエストが Cloud Run で 413 になる挙動差が発生する。
- 画像処理 API としても、JPEG 50 MiB は事実上の上限を超えるユースケース。32 MiB あれば 4K PNG の高画質も収まる。
- 将来 50 MiB 超のリクエストが必要になった場合は `--use-http2` 有効化 + 設定値上書きで対応可能。

**代替案**:
- `--use-http2` で 32 MiB 上限を撤廃: コード側に h2c (cleartext HTTP/2) Listen 設定が必要になり、かつクライアント側の HTTP/2 対応が必要。シンプルさで負ける。
- 環境別設定ファイル (`configs/cloudrun.yaml` 等) で切り替え: 設定の二重管理になり、ローカルとデプロイで挙動が分かれる。

### D4: 認証方式 (GitHub Actions → GCP)

**決定**: Workload Identity Federation (WIF) を採用し、Service Account キー JSON は使わない。

**理由**:
- SA キー JSON はリポジトリ Secrets に保存しても、漏洩時に永続的な認証情報になりリスクが大きい。
- WIF は OIDC トークンを毎回交換する短命認証で、漏洩リスクなし。Google も公式に推奨。
- `attribute-condition` で「`FrontWorksDev/Loki` リポジトリ限定」、`principalSet` で「同リポジトリの GitHub Actions のみ」と二重に絞ることで、トークン悪用経路を最小化できる。

**代替案**:
- SA キー JSON: セットアップが数分で済むが、長期的な運用リスクが上回る。後から WIF へ切り替えるのも手戻りが発生するため、最初から WIF にする。

### D5: API 公開ポリシー

**決定**: `--allow-unauthenticated` で公開し、CORS allowed_origins と既存のレートリミット (30 req/min/IP) でアクセス制御する。

**理由**:
- フロントエンド (`Lugh`) が完全クライアントサイド (Astro 静的生成 + ブラウザから直接 fetch) のため、IAM 認証は visitor 全員に Google ログインを要求することになり実用不可。
- CORS allowed_origins を本番ドメインと dev サーバ (`http://localhost:4321`) に絞ることでブラウザ経由の不正利用を抑止。CORS は browser-only の防御だが、既存レートリミット + `--max-instances=3` でコスト暴走を物理的に防ぐ。
- 将来サーバ間呼び出し (Lugh のバックエンドが介在する構成等) が必要になった場合は、別チケットで IAM 認証を追加可能。

**代替案**:
- IAM 認証必須: フロントエンド構成と矛盾するため不採用。
- API キー方式: フロントエンドに埋め込めば実質公開と変わらず、追加コードで複雑性のみ増える。Lugh のバックエンドが介在する構成になった時点で再検討する。

### D6: デプロイの段階的検証

**決定**: 実装を 3 フェーズに分けて各段階で動作確認する。

- **フェーズ 1 (コンテナ化)**: `body_limit_bytes` を 32 MiB に修正、`Dockerfile` / `.dockerignore` / `docker-compose.yml` 作成、テスト更新。`docker compose up` でローカル疎通確認。
- **フェーズ 2 (手動デプロイ)**: GCP 側を `docs/deployment/gcp-setup.md` の手順でセットアップし、ローカルから `gcloud run deploy` で 1 回デプロイして Cloud Run URL から疎通確認、Lugh dev サーバから CORS 含めて fetch 確認。
- **フェーズ 3 (CI 自動化)**: `.github/workflows/deploy.yml` を追加し、`workflow_dispatch` で手動実行 → main マージで自動実行を確認、ロールバック手順を 1 回リハーサル。

**理由**: CGO + Cloud Run は初めての組合せのため、CI で初めて動かすとデバッグが「Dockerfile か WIF かワークフローか」の切り分けで難航する。手動デプロイで疎通させてから CI 化することで、CI のデバッグはワークフロー記法に絞れる。

### D7: イメージタグ戦略

**決定**: `:${{ github.sha }}` (固定) と `:latest` (移動タグ) の二系統で push する。

**理由**:
- `:sha` でリビジョン特定可能 (Cloud Run のリビジョン履歴と SHA が紐付く)、ロールバック時に「どの SHA に戻すか」が明確。
- `:latest` は `gcloud run deploy` での参照や、開発時の動作確認に便利。
- 両方を push しておけば、後から「特定 SHA に固定したい」「latest を追従したい」のどちらにも対応可能。

**代替案**:
- `:sha` のみ: シンプルだが、`latest` がないと "今動いている最新" を参照しづらい。
- セマンティックバージョン (`:v1.0.0`) タグ: 個人プロジェクトのデプロイ単位としてはオーバーヘッドが大きい。

### D8: `docker-compose.yml` の用途とホットリロードの不採用

**決定**: compose は「本番イメージとほぼ同等の環境でローカル起動できる」確認用途のみとし、ホットリロード (Air 等) は導入しない。

**理由**:
- 開発時の主な起動方法は `go run ./cmd/api` (依存ツール不要、IDE デバッガ統合容易)。compose を開発主軸にする必要はない。
- ホットリロード導入は依存追加 + 設定維持コストが発生し、現状のニーズに対して過剰。
- compose は「Cloud Run と同じイメージで health 応答が出るか」「環境変数注入が期待通りか」を検証するための用途に限定する。

### D9: distroless でのコンテナヘルスチェック非対応

**決定**: Docker Compose の `healthcheck` は無効化する (`disable: true`)。代わりに手動で `curl http://localhost:8080/api/v1/health` を叩く運用にする。

**理由**:
- distroless には `wget` / `curl` / shell が含まれないため、典型的な `healthcheck.test` が書けない。
- ヘルスチェック用のバイナリを別途同梱するのはイメージ肥大化と複雑化を招く。
- Cloud Run 側は startup probe (TCP 接続) と liveness probe を別レイヤで持つため、Docker レベルの healthcheck は本番には影響しない。

**代替案**:
- `distroless/base-debian12:debug` (busybox 入り) を使う: ローカル確認のためだけにイメージ構成を本番と分けるのは整合性の観点で避ける。

### D10: Cloud Run のリージョンとインスタンス設定

**決定**: リージョン `asia-northeast1` (東京)、`min-instances=0`、`max-instances=3`、`memory=512Mi`、`cpu=1`、`concurrency=10`、`timeout=60s`。

**理由**:
- 国内利用前提のため `asia-northeast1` を採用 (レイテンシ最小)。
- `min=0` でアイドル時の課金ゼロ、Go のコールドスタートは ~1-2 秒で許容範囲。
- `max=3` でコスト暴走の物理上限を担保。請求アラートと併用。
- `memory=512Mi` / `cpu=1` は画像処理 (32 MiB リクエスト × 数枚並行) に最低限の構成。OOM が出たら 1Gi に上げる方針。
- `concurrency=10` は 1 インスタンスあたりの同時処理数。画像処理メモリ負荷を考慮して控えめに。
- `timeout=60s` は画像処理が長引いたケースに備える保守的な値。

「最小から始めて足りなくなったら上げる」方針 (FRO-114 チケット記載通り)。

## Risks / Trade-offs

- **[CGO 依存により glibc バージョン不整合のリスク]** → ビルドステージとランタイムステージを同じ Debian 12 (bookworm) に揃えることで回避。`alpine` への切替時は再検証が必要。
- **[`body_limit_bytes` 既定値変更による既存挙動の差]** → コード側既定とドキュメントを同時更新。32〜50 MiB の範囲で明示設定しているクライアントには影響あり。`README.md` と `proposal.md` Impact 節で周知。
- **[CORS だけでは非ブラウザクライアント (curl 等) からの直接アクセスは防げない]** → 本変更のスコープ。アプリ側レートリミット (30 req/min/IP) と `--max-instances=3` で乱用とコスト暴走を抑制。本格的な防御が必要になった時点で IAM 認証 / API キー方式を別チケットで導入。
- **[コールドスタートによる初回レイテンシ]** → `min=0` でコスト最小化を優先。Go のコールドスタートは数秒程度で許容範囲。Lugh 側でローディング表現を入れて UX を補完。
- **[GCP 請求の暴走リスク]** → 三層防御: (1) Cloud Run `--max-instances=3` / `--concurrency=10` / `--timeout=60s`、(2) アプリ側レートリミット、(3) GCP 請求アラート (運用ドキュメントに設定手順記載)。
- **[WIF 設定ミス時のデプロイ不能]** → `docs/deployment/gcp-setup.md` に手順を網羅、初回手動デプロイで検証してから CI に乗せる段階的アプローチで早期に発見可能。
- **[Compute Engine 既定 SA をランタイムに使うことによる権限過多]** → 本 API は GCP API を一切叩かないため実害なし。GCS 等を使うときに専用 SA を作って差し替える (別チケット)。
- **[新リビジョンが応答しない場合のサービス停止]** → Cloud Run は全リビジョンを保持するため、`gcloud run services update-traffic --to-revisions=<PREV>=100` で即時ロールバック可能。手順を `docs/deployment/cloud-run.md` に明記し、フェーズ 3 完了時に 1 回リハーサルする。

## Migration Plan

本変更は新機能追加が中心で、既存挙動の変更は `body_limit_bytes` 既定値のみ。

**ロールアウト手順**:

1. フェーズ 1 のコミットで `body_limit_bytes` 既定値変更を含む。テストを通して問題ないことを確認。
2. フェーズ 2 で初回手動デプロイ実施 (CI 経由ではない)。Cloud Run リビジョン #1 が立ち上がる。
3. フェーズ 3 で CI 自動デプロイを有効化。次回 main マージ時にリビジョン #2 が自動生成される。

**ロールバック戦略**:

- **コードレベル**: `git revert` で当該 commit を打ち消し、PR 経由で main に戻す。次の自動デプロイで Cloud Run も追従。
- **デプロイレベル (緊急時)**: Cloud Run のリビジョン履歴を使い、`gcloud run services update-traffic loki-api --to-revisions=<PREV>=100 --region=asia-northeast1` で即時に直前リビジョンに戻す。コードを後から修正できる。
- **GCP セットアップレベル**: WIF / Service Account / Artifact Registry はリポジトリ外のリソース。誤設定時は `docs/deployment/gcp-setup.md` の手順を再実行 (冪等な部分は二重作成エラー、それ以外は `update` コマンド)。

## Open Questions

- **Cloud Run リビジョンの保持数**: 既定値 (各リビジョンは削除されない) で運用開始。ストレージ料金が無視できる規模になるまで何もしない。気になり始めたら `gcloud run revisions delete` で手動掃除、または別チケットで自動掃除を検討。

## Resolved

- **`Lugh` の本番ドメイン**: `https://tool.frontworks.dev` に確定 (2026-05-03)。`LOKI_CORS_ORIGINS` Secret には `https://tool.frontworks.dev,http://localhost:4321` をセットする。
