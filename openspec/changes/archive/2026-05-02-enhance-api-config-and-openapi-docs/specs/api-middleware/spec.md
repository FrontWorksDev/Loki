## MODIFIED Requirements

### Requirement: 設定の外部化

CORS、ロギングレベル、ボディサイズ上限、レートリミット、**リッスンアドレス (`api.host`)、リッスンポート (`api.port`)** の各設定値は `configs/default.yaml` の `api:` セクションおよび環境変数（プレフィックス `LOKI_API_`、ネストはアンダースコアで表現）から読み込み可能でなければならない（MUST）。設定ファイル不在時は安全側のデフォルト（**ホスト `"0.0.0.0"`**、**ポート `8080`**、オリジン `["*"]`、ボディ50MB、30req/分・バースト10、ログレベル `info`）が適用されなければならない（MUST）。**HTTP サーバの listen アドレスは `net.JoinHostPort(host, port)` 形式で構築され、IPv6 アドレス (例: `"::"`) を `host` に指定可能でなければならない（MUST）。**

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
