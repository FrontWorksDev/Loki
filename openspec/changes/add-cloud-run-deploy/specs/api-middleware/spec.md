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
