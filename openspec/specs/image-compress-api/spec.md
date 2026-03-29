## ADDED Requirements

### Requirement: 画像圧縮エンドポイントの提供
システムは `POST /api/v1/compress` エンドポイントを提供し、multipart/form-data 形式で画像ファイルを受け取り、圧縮された画像バイナリをレスポンスとして返さなければならない（SHALL）。

#### Scenario: JPEG画像の圧縮（デフォルトオプション）
- **WHEN** JPEG画像ファイルを `file` フィールドでアップロードする
- **THEN** ステータス200で圧縮されたJPEG画像バイナリが返される
- **THEN** Content-Type ヘッダーが `image/jpeg` である

#### Scenario: PNG画像の圧縮
- **WHEN** PNG画像ファイルを `file` フィールドでアップロードする
- **THEN** ステータス200で圧縮されたPNG画像バイナリが返される
- **THEN** Content-Type ヘッダーが `image/png` である

#### Scenario: WebP画像の圧縮
- **WHEN** WebP画像ファイルを `file` フィールドでアップロードする
- **THEN** ステータス200で圧縮されたWebP画像バイナリが返される
- **THEN** Content-Type ヘッダーが `image/webp` である

### Requirement: 圧縮品質の指定
システムは `quality` パラメータ（整数、1-100）で圧縮品質を指定できなければならない（SHALL）。0または未指定の場合は圧縮レベルに基づくデフォルト値を使用する。

#### Scenario: quality パラメータを指定した圧縮
- **WHEN** `quality=50` を指定してJPEG画像をアップロードする
- **THEN** 指定された品質で圧縮された画像が返される

#### Scenario: quality 未指定時のデフォルト動作
- **WHEN** `quality` を指定せずに画像をアップロードする
- **THEN** 圧縮レベル（デフォルト: medium）に基づく品質で圧縮される

### Requirement: 圧縮レベルの指定
システムは `level` パラメータ（文字列: low/medium/high）で圧縮レベルを指定できなければならない（SHALL）。未指定の場合は medium をデフォルトとする。

#### Scenario: level=high を指定した圧縮
- **WHEN** `level=high` を指定して画像をアップロードする
- **THEN** 高圧縮レベルで圧縮された画像が返される

#### Scenario: level 未指定時のデフォルト動作
- **WHEN** `level` を指定せずに画像をアップロードする
- **THEN** medium レベルで圧縮される

### Requirement: 圧縮結果メタデータのレスポンスヘッダー
システムはレスポンスヘッダーに圧縮結果のメタデータを含めなければならない（SHALL）。

#### Scenario: レスポンスヘッダーにメタデータが含まれる
- **WHEN** 画像を正常に圧縮した場合
- **THEN** `X-Original-Size` ヘッダーに元のファイルサイズ（バイト数）が設定される
- **THEN** `X-Compressed-Size` ヘッダーに圧縮後のファイルサイズ（バイト数）が設定される
- **THEN** `X-Compression-Ratio` ヘッダーに圧縮率（パーセンテージ）が設定される

### Requirement: 対応フォーマットのバリデーション
システムは JPEG、PNG、WebP 以外の画像フォーマットに対してエラーを返さなければならない（SHALL）。

#### Scenario: 非対応フォーマットのアップロード
- **WHEN** GIF画像やテキストファイルなど非対応フォーマットをアップロードする
- **THEN** 422 もしくは 400 エラーが返される

### Requirement: ファイルサイズ上限
システムは 50MB を超えるファイルのアップロードを拒否しなければならない（SHALL）。

#### Scenario: 50MBを超えるファイルのアップロード
- **WHEN** 50MB を超えるファイルをアップロードする
- **THEN** エラーレスポンスが返される

### Requirement: ファイル必須バリデーション
システムは `file` フィールドが未指定の場合にエラーを返さなければならない（SHALL）。

#### Scenario: ファイル未指定でのリクエスト
- **WHEN** `file` フィールドなしでリクエストを送信する
- **THEN** 422 バリデーションエラーが返される
