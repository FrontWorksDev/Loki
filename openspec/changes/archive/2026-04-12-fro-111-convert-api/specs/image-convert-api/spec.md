## ADDED Requirements

### Requirement: フォーマット変換エンドポイント
システムは `POST /api/v1/convert` エンドポイントを提供しなければならない（SHALL）。リクエストは `multipart/form-data` 形式で、変換された画像をバイナリレスポンスとして返す。

#### Scenario: JPEG画像をWebPに変換
- **WHEN** JPEG画像ファイルを `file` フィールドに、`format=webp` を指定してPOSTする
- **THEN** WebP形式に変換された画像バイナリがレスポンスボディとして返され、`Content-Type` は `image/webp` である

#### Scenario: PNG画像をJPEGに変換
- **WHEN** PNG画像ファイルを `file` フィールドに、`format=jpeg` を指定してPOSTする
- **THEN** JPEG形式に変換された画像バイナリが返され、`Content-Type` は `image/jpeg` である

#### Scenario: WebP画像をPNGに変換
- **WHEN** WebP画像ファイルを `file` フィールドに、`format=png` を指定してPOSTする
- **THEN** PNG形式に変換された画像バイナリが返され、`Content-Type` は `image/png` である

### Requirement: 出力品質制御
システムは `quality` パラメータ（0-100の整数）および `level` パラメータ（low/medium/high）による出力品質制御を提供しなければならない（MUST）。`quality` が1-100の値で指定された場合は `level` より優先される。`quality=0` または `quality` 未指定の場合はデフォルト値を使用し、`level` が指定されていればその `level` に対応する品質、`level` も未指定であれば `medium` 相当の品質を適用する。

#### Scenario: quality指定での変換
- **WHEN** `quality=80` を指定してフォーマット変換リクエストを送信する
- **THEN** 指定された品質で変換された画像が返される

#### Scenario: quality=0指定での変換
- **WHEN** `quality=0` を指定してフォーマット変換リクエストを送信する
- **THEN** `quality` 未指定時と同様にデフォルトの品質設定が適用される

#### Scenario: level指定での変換
- **WHEN** `level=high` を指定してフォーマット変換リクエストを送信する
- **THEN** 高品質設定で変換された画像が返される

#### Scenario: quality未指定・level未指定
- **WHEN** `quality` も `level` も指定せずにフォーマット変換リクエストを送信する
- **THEN** デフォルトの品質設定（medium相当）で変換された画像が返される

### Requirement: 同一フォーマット変換のフォールバック
入力フォーマットと出力フォーマットが同一の場合、システムは圧縮処理にフォールバックしなければならない（MUST）。

#### Scenario: JPEG→JPEG変換
- **WHEN** JPEG画像を `format=jpeg` で変換リクエストする
- **THEN** 圧縮処理が実行され、圧縮された画像が返される（エラーにはならない）

### Requirement: レスポンスヘッダーにメタデータ付与
システムはレスポンスヘッダーに変換結果のメタデータを付与しなければならない（MUST）。

#### Scenario: 変換成功時のヘッダー
- **WHEN** フォーマット変換が成功する
- **THEN** レスポンスヘッダーに `X-Original-Size`（元サイズ）、`X-Converted-Size`（変換後サイズ）、`X-Original-Format`（元フォーマット）、`X-Output-Format`（出力フォーマット）が含まれる

### Requirement: 必須パラメータのバリデーション
`file` と `format` は必須パラメータであり、未指定の場合はエラーを返さなければならない（MUST）。

#### Scenario: ファイル未指定
- **WHEN** `file` フィールドなしでリクエストを送信する
- **THEN** HTTPステータス 422 が返される

#### Scenario: format未指定
- **WHEN** `format` フィールドなしでリクエストを送信する
- **THEN** HTTPステータス 422 が返される

#### Scenario: 非対応フォーマット指定
- **WHEN** `format=gif` など非対応フォーマットを指定する
- **THEN** HTTPステータス 400 が返される

### Requirement: エラーハンドリング
システムは以下のエラーケースを適切に処理しなければならない（MUST）。

#### Scenario: 非対応MIMEタイプの画像
- **WHEN** GIF等の非対応フォーマットの画像ファイルをアップロードする
- **THEN** HTTPステータス 400 が返される

#### Scenario: 不正な画像データ
- **WHEN** 壊れた画像データをアップロードする
- **THEN** HTTPステータス 400 が返される

#### Scenario: ファイルサイズ超過
- **WHEN** 50MBを超える画像ファイルをアップロードする
- **THEN** HTTPステータス 413 が返される
