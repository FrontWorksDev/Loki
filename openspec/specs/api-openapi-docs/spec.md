## Purpose

API サーバが Huma v2 経由で自動公開する OpenAPI スペックの品質要件 (Info メタ情報、共通エラーレスポンス、リクエスト/レスポンス例) を、エンドポイント個別ではなく横断的に定義する。

## Requirements

### Requirement: OpenAPI Info メタ情報の設定

API サーバは Huma が生成する OpenAPI スペックの `info` セクションに、最低限 `title`、`version`、`description`、`contact`、`license` を設定しなければならない（MUST）。`description` は API の概要、対応フォーマット (JPEG / PNG / WebP)、認証ポリシー (認証なし)、レート制限の概要を含む複数行のテキストでなければならない（MUST）。

#### Scenario: OpenAPI 生成時に Info メタが含まれる

- **WHEN** サーバが `huma.DefaultConfig` で構成され OpenAPI スペックが生成される
- **THEN** `info.title == "Loki Image API"` であること
- **THEN** `info.version` が非空であること
- **THEN** `info.description` に「画像圧縮」「画像変換」「JPEG」「PNG」「WebP」のいずれの語も含まれること
- **THEN** `info.contact.name` と `info.contact.url` が設定されていること
- **THEN** `info.license.name` と `info.license.url` が設定されていること

### Requirement: 画像エンドポイントの共通エラーレスポンス

システムは画像処理エンドポイント (`POST /api/v1/compress`、`POST /api/v1/convert`) の OpenAPI 定義に、`huma.ErrorModel` (RFC9457 Problem Details 準拠) スキーマを参照する以下の共通エラーレスポンスを明示しなければならない（MUST）: `400 Bad Request`、`413 Payload Too Large`、`422 Unprocessable Entity`、`429 Too Many Requests`、`500 Internal Server Error`。各レスポンスは `application/problem+json` メディアタイプで定義されなければならない（MUST）。

#### Scenario: compress エンドポイントのエラーレスポンス定義

- **WHEN** OpenAPI スペックの `paths."/api/v1/compress".post.responses` を取得する
- **THEN** キーとして `"400"`, `"413"`, `"422"`, `"429"`, `"500"` がすべて存在する
- **THEN** 各エラーレスポンスの `content` キーに `application/problem+json` が含まれる

#### Scenario: convert エンドポイントのエラーレスポンス定義

- **WHEN** OpenAPI スペックの `paths."/api/v1/convert".post.responses` を取得する
- **THEN** キーとして `"400"`, `"413"`, `"422"`, `"429"`, `"500"` がすべて存在する
- **THEN** 各エラーレスポンスの `content` キーに `application/problem+json` が含まれる

#### Scenario: ヘルスチェックは共通エラー対象外

- **WHEN** OpenAPI スペックの `paths."/api/v1/health".get.responses` を取得する
- **THEN** 共通エラーレスポンス（400/413/422/429/500）は含まれない（200 のみ、もしくは Huma の最小デフォルトのみ）

### Requirement: 主要入力フィールドの例 (example) 付与

システムは画像処理エンドポイントの主要入力フィールドに、OpenAPI スキーマの `example` 値を付与しなければならない（MUST）。対象は compress の `quality` / `level`、convert の `format` / `quality` / `level` とする。

#### Scenario: compress の入力フィールドに example が含まれる

- **WHEN** OpenAPI スペックから compress の `requestBody` の multipart スキーマを取得する
- **THEN** `quality` フィールドに数値の `example` が設定されている
- **THEN** `level` フィールドに `"low" | "medium" | "high"` のいずれかの `example` が設定されている

#### Scenario: convert の入力フィールドに example が含まれる

- **WHEN** OpenAPI スペックから convert の `requestBody` の multipart スキーマを取得する
- **THEN** `format` フィールドに `"jpeg" | "png" | "webp"` のいずれかの `example` が設定されている
- **THEN** `quality` フィールドに数値の `example` が設定されている
- **THEN** `level` フィールドに `"low" | "medium" | "high"` のいずれかの `example` が設定されている
