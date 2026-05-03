## MODIFIED Requirements

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
