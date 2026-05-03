## Purpose

API サーバの横断的な HTTP ミドルウェア (CORS、構造化ロギング、リクエストボディサイズ制限、IP ベースレートリミット、ヘルスチェック、ミドルウェア登録順序) と、それらに関連する設定の外部化 (リッスンアドレス・ポート・各種上限値の YAML / 環境変数からの読み込み) に関する仕様を集約する。
## Requirements
### Requirement: CORSミドルウェアの適用

APIサーバーはすべてのHTTPレスポンスに対してCORSヘッダーを付与しなければならない（SHALL）。許可されるオリジン、メソッド、ヘッダーは設定ファイル（`configs/default.yaml` の `api.cors`）および環境変数から読み込み可能でなければならない（MUST）。プリフライトリクエスト（`OPTIONS`）に対しては2xx応答（go-chi/corsライブラリの既定では200）を返し、レートリミットの対象外としなければならない（MUST）。

#### Scenario: 許可オリジンからのGETリクエスト
- **WHEN** クライアントが `Origin: https://example.com` ヘッダー付きで `GET /api/v1/health` を送信し、設定で `https://example.com` が許可されている
- **THEN** レスポンスは200を返し、`Access-Control-Allow-Origin: https://example.com` ヘッダーが含まれる

#### Scenario: 許可されていないオリジンからのリクエスト
- **WHEN** クライアントが `Origin: https://evil.example` ヘッダー付きでリクエストを送信し、設定で当該オリジンが許可されていない
- **THEN** レスポンスは200相当（リクエスト自体は処理される）だが、`Access-Control-Allow-Origin` ヘッダーは付与されない

#### Scenario: プリフライトリクエスト
- **WHEN** クライアントが `OPTIONS /api/v1/compress` を `Origin` ヘッダーおよび `Access-Control-Request-Method: POST` ヘッダー付きで送信する
- **THEN** レスポンスは2xxを返し、`Access-Control-Allow-Methods` と `Access-Control-Allow-Headers` が設定値に従って付与される

#### Scenario: ワイルドカード許可（デフォルト）
- **WHEN** 設定 `api.cors.allowed_origins` が `["*"]` で、任意のオリジンからリクエストが来る
- **THEN** `Access-Control-Allow-Origin: *` が返り、`allow_credentials` が `false` であることが保証される

### Requirement: 構造化リクエストロギング

APIサーバーは受信したすべてのHTTPリクエストに対して、`log/slog` のJSONハンドラを用いた1行のJSONログを出力しなければならない（MUST）。ログは少なくとも以下のフィールドを含む（MUST）: `time`, `level`, `msg`, `method`, `path`, `status`, `duration_ms`, `remote_ip`, `bytes_out`, `request_id`。出力先は標準出力でなければならない（MUST。Cloud Run の Cloud Logging が標準出力を吸い上げるため）。

#### Scenario: 正常レスポンス時のログ出力
- **WHEN** クライアントが `GET /api/v1/health` を送信し、サーバーが200を返す
- **THEN** 標準出力にJSON1行が書き込まれ、`method="GET"`, `path="/api/v1/health"`, `status=200`, `duration_ms` が0以上の数値、`request_id` が空でない文字列であること

#### Scenario: エラーレスポンス時のログ出力
- **WHEN** ハンドラが500を返す（panic からの復旧含む）
- **THEN** ログレベル `error` で、`status=500` を含むJSONログが出力される

#### Scenario: リクエストIDの伝播
- **WHEN** リクエストにミドルウェアチェーンで `RequestID` が付与される
- **THEN** ログ中の `request_id` とレスポンスヘッダー `X-Request-Id` が同一の値である

### Requirement: リクエストボディサイズ制限

APIサーバーは受信リクエストのボディサイズに対して上限を強制しなければならない（MUST）。デフォルトは 32 MiB (33,554,432 バイト) とし、設定 `api.body_limit_bytes` で上書き可能でなければならない（MUST）。上限超過時は HTTP 413 (Payload Too Large) を返し、ハンドラには到達させてはならない（MUST NOT）。デフォルト値は Google Cloud Run の HTTP/1 リクエスト上限 (32 MiB) と整合させ、ローカル開発環境とデプロイ環境で挙動差を生じさせてはならない（MUST）。

#### Scenario: 上限以内のリクエスト
- **WHEN** クライアントが31MiBのボディで `POST /api/v1/compress` を送信する
- **THEN** リクエストはハンドラに到達し、通常通り処理される

#### Scenario: 上限ちょうどのリクエスト
- **WHEN** クライアントが32MiB (33,554,432 バイト) ちょうどのボディで `POST /api/v1/compress` を送信する
- **THEN** リクエストはハンドラに到達し、通常通り処理される

#### Scenario: 上限超過リクエスト
- **WHEN** クライアントが33MiBのボディで `POST /api/v1/compress` を送信する
- **THEN** ハンドラには到達せず、レスポンスは413であること、ボディはJSON形式のエラー（Humaのエラー形式に準拠）であること

#### Scenario: 設定値での上書き
- **WHEN** `api.body_limit_bytes: 1048576`（1MiB）が設定され、2MiBのボディが送信される
- **THEN** レスポンスは413であること

#### Scenario: Cloud Run HTTP/1 上限との整合
- **WHEN** デフォルト設定でデプロイされた Cloud Run サービスに 32 MiB のリクエストを送る
- **THEN** Cloud Run 側でも API サーバ側でも 413 とならず処理される (両者の上限が一致しているため)

### Requirement: IPベースレートリミット

APIサーバーはクライアントIPごとにリクエストレートを制限しなければならない（MUST）。デフォルトは 1分あたり30リクエスト、バースト10とする。レート超過時は HTTP 429 (Too Many Requests) を返さなければならない（MUST）。`OPTIONS` リクエスト（CORSプリフライト）はレートリミットの対象外でなければならない（MUST NOT count）。クライアントIPは `X-Forwarded-For` の最左の値を優先し、不在の場合は `RemoteAddr` を用いる（MUST）。

#### Scenario: レート以内のリクエスト
- **WHEN** クライアントIPが1分間に20リクエストを送信する（デフォルト30/分）
- **THEN** すべてのリクエストが許可される（少なくともレートリミット由来の429は返らない）

#### Scenario: レート超過
- **WHEN** クライアントIPが1分間に40リクエストを送信し、デフォルト設定（30/分・バースト10）が有効
- **THEN** 制限超過後のリクエストに対して429が返り、`Retry-After` ヘッダーが付与される

#### Scenario: 異なるIPは独立にカウント
- **WHEN** IP A と IP B がそれぞれ独立にリクエストを送信する
- **THEN** 一方の超過が他方に影響してはならない

#### Scenario: プリフライトの除外
- **WHEN** クライアントIPがレート上限に達した状態で `OPTIONS` リクエストを送信する
- **THEN** プリフライトは429を返さず、CORSミドルウェアによる2xx応答が返る

#### Scenario: X-Forwarded-Forによるクライアント識別
- **WHEN** リクエストヘッダーが `X-Forwarded-For: 203.0.113.10, 10.0.0.1` を含む
- **THEN** レートリミットは `203.0.113.10` をキーとして集計する

### Requirement: ヘルスチェックエンドポイント

APIサーバーは `GET /api/v1/health` でサーバーの稼働状態を返さなければならない（MUST）。本エンドポイントは認証・レートリミット・サイズ制限のいずれにも左右されず、サーバープロセスが稼働している限り常に 200 を返す（MUST）。レスポンスボディは `{"status":"ok"}` 固定のJSONとする（MUST）。

#### Scenario: 通常時の応答
- **WHEN** クライアントが `GET /api/v1/health` を送信する
- **THEN** ステータス200、`Content-Type: application/json`、ボディは `{"status":"ok"}` であること

#### Scenario: レート上限到達後でも200
- **WHEN** クライアントが他エンドポイントでレート上限に達した直後に `GET /api/v1/health` を送信する
- **THEN** ヘルスチェックは200を返す（運用要件のためレート制限は適用しない）

### Requirement: ミドルウェアの登録順序

APIサーバーはミドルウェアを以下の順序（外側から内側）でChiルーターに登録しなければならない（MUST）: RequestID → Recoverer → Logging → CORS → RateLimit → BodyLimit → ハンドラ。これによりリクエストID付与・panic復旧・ロギング・CORSプリフライト処理がレートリミット判定より前に実行されることが保証される。

#### Scenario: ハンドラ内panicでも500とログ
- **WHEN** 任意のハンドラ内で panic が発生する
- **THEN** Recovererが500を返し、Loggingがそのリクエストの `status=500` をJSONログに出力する

#### Scenario: CORSプリフライトはレート対象外
- **WHEN** クライアントIPがレート上限に達した状態で `OPTIONS` リクエストが到来する
- **THEN** CORSミドルウェアが先に短絡応答するため、RateLimitに到達せず2xxが返る

### Requirement: 設定の外部化

CORS、ロギングレベル、ボディサイズ上限、レートリミット、**リッスンアドレス (`api.host`)、リッスンポート (`api.port`)** の各設定値は `configs/default.yaml` の `api:` セクションおよび環境変数（プレフィックス `LOKI_API_`、ネストはアンダースコアで表現）から読み込み可能でなければならない（MUST）。設定ファイル不在時は安全側のデフォルト（**ホスト `"0.0.0.0"`**、**ポート `8080`**、オリジン `["*"]`、**ボディ32MiB (Cloud Run HTTP/1 上限と整合)**、30req/分・バースト10、ログレベル `info`）が適用されなければならない（MUST）。**HTTP サーバの listen アドレスは `net.JoinHostPort(host, port)` 形式で構築され、IPv6 アドレス (例: `"::"`) を `host` に指定可能でなければならない（MUST）。** **CORS の `allowed_origins` / `allowed_methods` / `allowed_headers` のようなスライス型設定を環境変数で指定する場合はカンマ区切りで列挙でき、サーバー側でカンマで分割し各要素を trim してリストとして解釈しなければならない（MUST）。**

#### Scenario: デフォルト設定での起動

- **WHEN** `configs/default.yaml` に `api:` セクションが存在しない状態でサーバーを起動する
- **THEN** デフォルト値が適用され、サーバーが正常に起動する
- **THEN** ホストは `"0.0.0.0"`、ポートは `8080` で listen する

#### Scenario: 環境変数による上書き

- **WHEN** 環境変数 `LOKI_API_BODY_LIMIT_BYTES=1048576` を設定して起動する
- **THEN** ボディサイズ上限が 1 MiB として動作し、`configs/default.yaml` の値より優先される

#### Scenario: YAML設定の反映

- **WHEN** `configs/default.yaml` の `api.rate_limit.requests_per_minute: 60` を設定して起動する
- **THEN** レートリミットは1分あたり60リクエストとして動作する

#### Scenario: ホストの環境変数オーバーライド

- **WHEN** 環境変数 `LOKI_API_HOST=127.0.0.1` を設定して起動する
- **THEN** サーバは `127.0.0.1:<port>` のみで listen し、外部インタフェースからは到達できない

#### Scenario: IPv6 ホストの指定

- **WHEN** 環境変数 `LOKI_API_HOST=::` を設定して起動する
- **THEN** サーバは `[::]:<port>` で listen する（`net.JoinHostPort` がブラケットを付与する）

#### Scenario: スライス型設定のカンマ区切り環境変数による上書き

- **WHEN** 環境変数 `LOKI_API_CORS_ALLOWED_ORIGINS=https://tool.frontworks.dev,http://localhost:4321` を設定して起動する
- **THEN** `allowed_origins` は `["https://tool.frontworks.dev", "http://localhost:4321"]` の 2 要素として解釈され、両オリジンからのリクエストに `Access-Control-Allow-Origin` ヘッダーが返る

#### Scenario: 空白を含むスライス型環境変数

- **WHEN** 環境変数 `LOKI_API_CORS_ALLOWED_METHODS="GET, POST, OPTIONS"` を設定して起動する
- **THEN** 各要素から空白が trim され、`["GET", "POST", "OPTIONS"]` の 3 要素として解釈される

