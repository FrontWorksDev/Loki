## Context

LokiのAPIサーバー（`internal/api/server.go`）はHuma v2 と Chi v5 の組み合わせで構築されており、現在はChiルーターに対してミドルウェアを一切登録していない。`NewServer` で `chi.NewMux()` を生成し、Humaアダプタ経由でハンドラを登録するのみである。

設定は `internal/api.Config` 構造体にハードコードされた `DefaultConfig()` のみが存在し、`configs/default.yaml` には現在CLI関連の設定しか入っていない。`cmd/api/main.go` は `api.DefaultConfig()` をそのまま渡している。

Cloud Run上で公開APIとして運用するために必要な横断的関心事（CORS、構造化ロギング、ボディサイズ制限、レートリミット、ヘルスチェック）を `internal/api/middleware/` パッケージとして整備し、`Server` の組み立て時にChiルーターへ登録する。設定は `viper` 経由で `configs/default.yaml` および環境変数から読み込めるようにする。

依存関係の方向（`cmd/` → `internal/api/` → `internal/api/middleware/` → 標準ライブラリ・サードパーティ）は既存の3層構造を踏襲し、循環依存を作らない。

## Goals / Non-Goals

**Goals:**

- ChiルーターにCORS、構造化ロギング、ボディサイズ制限、レートリミットを登録する
- `log/slog` によるJSON構造化ログを出力する（Cloud Logging互換）
- 設定値（許可オリジン、レート、サイズ上限など）を `configs/default.yaml` および環境変数で変更可能にする
- `GET /api/v1/health` を `api-middleware` capability の一部として仕様化する（実装は既存）
- 各ミドルウェアにテーブル駆動テストを追加し、`go test -race ./...` を維持する
- ミドルウェアの登録順序を明確に定義し、想定外の挙動（ロギング前に413が出てログが残らない等）を防ぐ

**Non-Goals:**

- 認証・認可の実装
- Redis等を用いた分散レートリミット（マルチインスタンス間共有）
- メトリクス・トレーシング
- WAF相当の高度防御
- 既存エンドポイントのレスポンス内容変更

## Decisions

### D1: ミドルウェアパッケージの配置 — `internal/api/middleware/`

**選択**: `internal/api/middleware/` 配下に機能ごとに `cors.go`, `logging.go`, `bodylimit.go`, `ratelimit.go` を作成。各ファイルは `func New<Name>(cfg <Name>Config) func(http.Handler) http.Handler` という Chi 互換のシグネチャを公開する。

**代替案**:
- `pkg/middleware/`（外部公開）→ Lokiのミドルウェアは本プロジェクト固有であり外部公開する理由がないため不採用。
- `internal/middleware/`（API以外でも使う前提）→ 現状TUI/CLIには不要であり、API専用として明示する方が責務が明確。

**理由**: 既存の `internal/api/{server,routes}.go` と並列に置くことで、API層の関心事として閉じ、依存方向が `server.go → middleware/` の単方向になる。

### D2: CORSライブラリ — `github.com/go-chi/cors`

**選択**: Chi公式ファミリーの `go-chi/cors` を使用する。

**代替案**:
- `github.com/rs/cors`：機能は同等だが、Chiとのインターフェース整合性で `go-chi/cors` のほうが取り回しが良い。
- 自前実装：preflight処理が地味に複雑（ベンダー固有ヘッダー、credentials周り）なので車輪の再発明を避ける。

**理由**: Chiエコシステムでデファクト。設定APIが `cors.Options` 構造体ベースで、本プロジェクトの `Config` から素直にマッピングできる。

### D3: レートリミット — `golang.org/x/time/rate` + IPキーのインメモリLRU

**選択**: `x/time/rate.Limiter` をクライアントIPごとに保持する `sync.Map`（または容量制限付きLRU）で管理する。

**代替案**:
- Redisベース分散レートリミット → Non-goalsで除外。
- `github.com/didip/tollbooth` 等のラッパー → 内部実装が `x/time/rate` ベースで、薄いラッパーを書くほうがテストしやすく挙動を制御できる。

**理由**: 標準準拠の `x/time/rate` で十分。マルチインスタンス展開時はアプリ側で分散実装に差し替えるための抽象として `RateLimiter interface { Allow(key string) bool }` を切る。

**容量制限**: 古いIPエントリを永続的に保持しないよう、最終アクセス時刻を記録し、定期的（例: 5分間隔のticker、または `Allow` 時のサンプリング）にクリーンアップする。シンプルさを優先し、初版は `sync.Map` + 単純な mutex 保護のクリーンアップループで実装する。

### D4: クライアントIPの取り出し — `X-Forwarded-For` の最左 → `RemoteAddr` フォールバック

**選択**: Cloud Run は `X-Forwarded-For` の最左にクライアントIPを設定する。これをまず参照し、なければ `r.RemoteAddr` を使う。設定で「信頼するプロキシ段数」を将来追加できる余地を残すが、初版は固定。

**理由**: Cloud Run公式仕様に合わせる。ローカル開発時は `RemoteAddr` で動く。

### D5: 構造化ロギング — 標準ライブラリ `log/slog` + JSONハンドラ

**選択**: `slog.New(slog.NewJSONHandler(os.Stdout, ...))` で生成したロガーをミドルウェアに注入する。出力先は `os.Stdout`（Cloud Run の Cloud Logging に自動吸い上げ）。

**ログフィールド**: `time`, `level`, `msg`, `method`, `path`, `status`, `duration_ms`, `remote_ip`, `bytes_out`, `user_agent`, `request_id`。`request_id` はChiの `middleware.RequestID` を使うか自前で `uuid` 生成するが、初版はChi標準の `middleware.RequestID` に乗る。

**代替案**: `zerolog` / `zap` → 高速だが追加依存になる。Go 1.21+ の `slog` で十分。

**理由**: 標準ライブラリ完結。Cloud Loggingが自動でJSONをパースして検索可能フィールドにする。

### D6: ボディサイズ制限の二段構え — Huma の `MaxBodyBytes` + 全体ミドルウェア

**現状**: 既存の `compress` / `convert` operation には `MaxBodyBytes: 50 * 1024 * 1024` が個別に設定されている。

**選択**:
1. ミドルウェア層で `http.MaxBytesReader` をリクエストボディに被せる（デフォルト50MB、設定で上書き可能）。
2. Huma operation の `MaxBodyBytes` は廃止せず、エンドポイント固有のより厳しい上限が必要な場合に上書きできるよう残す。両者の最小値が実効上限になる。

**理由**: ミドルウェアは「サーバー全体の防御」、operationの設定は「APIスキーマの宣言」として責務を分けると、OpenAPIドキュメントとしての明示性も保てる。両者とも維持する。

**413レスポンス**: `http.MaxBytesReader` 自体は読み取り時にエラーを返すだけなので、ミドルウェアで `r.Body = http.MaxBytesError` を捕捉して 413 + JSONエラー（Hostから出る Huma のエラー形式に合わせる）を返す薄いラッパーを実装する。

### D7: ミドルウェア登録順（外側 → 内側）

```
1. middleware.RequestID  (Chi標準)
2. middleware.Recoverer  (Chi標準、panic→500)
3. logging               (RequestIDが取れたあと)
4. cors                  (preflightへの応答)
5. ratelimit             (CORSプリフライト後)
6. bodylimit             (レート許可後にサイズ判定)
7. (route handlers)
```

**根拠**:
- ロギングは `RequestID` の後に置き、`request_id` をログに含められるようにする。
- panicからの復旧は最外周に近い位置に置き、後段ミドルウェアの panic でも 500 を返す。
- レートリミットはCORS preflight (`OPTIONS`) を許可する必要がある。`go-chi/cors` は preflight に対して短絡応答するため、CORSをratelimitの前に置けばOPTIONSはレート対象外にできる。
- ボディサイズ制限はレート許可済みリクエストにのみ適用すれば十分。

### D8: 設定スキーマと読み込み

**`configs/default.yaml` 拡張**:

```yaml
api:
  port: 8080
  cors:
    allowed_origins: ["*"]
    allowed_methods: ["GET", "POST", "OPTIONS"]
    allowed_headers: ["Content-Type", "Authorization"]
    allow_credentials: false
    max_age: 300
  body_limit_bytes: 52428800   # 50MB
  rate_limit:
    requests_per_minute: 30
    burst: 10
  logging:
    level: "info"             # debug/info/warn/error
```

`internal/api/config.go` を新設し、`api.Config` を拡張する。`cmd/api/main.go` で `viper` を使って `configs/default.yaml` を読み込み（環境変数 `LOKI_API_*` で上書き可能）、`api.Config` に詰めて `NewServer` に渡す。CLI側の既存 `viper` 利用パターン（`internal/cli/config.go`）に揃える。

### D9: ヘルスチェックの仕様化

既に `routes.go` で `GET /api/v1/health` を実装済み。本変更では「`api-middleware` capability の要件として仕様化」のみ行い、コード変更は不要。Cloud Run のヘルスチェック互換のため `200` を返し、ボディは `{"status":"ok"}` 固定。

## Risks / Trade-offs

- **インメモリレートリミットはマルチインスタンスで不正確** → Cloud Run はインスタンスがスケールアウトするため、各インスタンスが独立に30req/min を許可する。実効上限はインスタンス数×30。Non-goalsで明示済み。将来的に `RateLimiter interface` を Redis 実装に差し替える設計で逃げる。
- **`sync.Map` 上のクライアントIPエントリが無限増殖する** → 定期クリーンアップ（例: 10分以上アクセスのないキーを削除）で対処。クリーンアップロジックはテストでカバーする。
- **`X-Forwarded-For` の偽装** → Cloud Run はGoogleのフロントエンドが書き換えるので最左を信頼してよい。直接アクセス（`RemoteAddr`）も最終フォールバック。ローカル開発・将来の他環境では誤動作の余地があるためコメントで明記。
- **CORS `allowed_origins: ["*"]` がデフォルト** → 開発容易性のためのデフォルト。本番デプロイ時は環境変数で具体ドメインに上書きすることを README / コメントで明記する。
- **`http.MaxBytesReader` のエラーは Huma 層から見えにくい** → ミドルウェアで先に413を返してハンドラに到達させない設計にする。
- **ロギングミドルウェアによるレスポンスバッファリングのコスト** → ステータスコード捕捉のため `httptest.ResponseRecorder` 風の wrapper を使う。レスポンスボディは記録しない（バイト数のみ）ことで、画像バイナリの転送性能を維持する。
- **`go-chi/cors` の追加依存** → 軽量で他に共依存をほぼ持たないため許容。`golang.org/x/time/rate` は `golang.org/x/sys` 経由で既に間接依存しているエコシステム。

## Migration Plan

破壊的変更ではないため段階移行は不要。以下の順序で導入する。

1. `internal/api/middleware/` を新設して各ミドルウェアを実装＋テスト。
2. `internal/api/config.go` で Config を拡張。`DefaultConfig()` は後方互換を保つデフォルト値を返す。
3. `internal/api/server.go` で `chi.Use(...)` によりミドルウェアを登録。
4. `configs/default.yaml` に `api:` セクションを追加。
5. `cmd/api/main.go` で viper 経由の読み込みに切り替え（既存の `DefaultConfig()` パスは残す）。
6. README に CORS・レート制限・サイズ制限・環境変数の節を追記。

ロールバック: PR単位でrevertすれば元に戻る。データマイグレーションなし。

## Open Questions

- レートリミット超過時のエラーボディ形式は Huma のエラー形式（RFC 7807相当）に合わせるか、シンプルなJSONにするか → 既存の `compress` / `convert` がHumaのエラー形式を返すため、整合のためHuma形式に揃える方針で進める。実装時に確認。
- `request_id` を Huma レスポンスヘッダー（例: `X-Request-Id`）にも返すか → Cloud Run のログ追跡で便利なので返す方向で実装。Open Questionとしては解決済み扱い。
- ロギングレベルの動的変更（SIGHUPで再読み込み等）→ Non-goalsとし、本変更では起動時固定。
