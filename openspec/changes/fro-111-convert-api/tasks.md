## 1. セットアップ

- [x] 1.1 `feature/fro-111-convert-api` ブランチを作成する

## 2. ConvertHandler実装

- [x] 2.1 `internal/handler/convert.go` を作成し、ConvertHandler（ConvertFormData, ConvertInput, parseImageFormat, Handle）を実装する
- [x] 2.2 同一フォーマット変換時の圧縮フォールバックロジックを実装する

## 3. ルート登録

- [x] 3.1 `internal/api/routes.go` に `POST /api/v1/convert` のルート登録とSwagger定義を追加する
- [x] 3.2 `internal/api/server.go` に ConvertHandler のインスタンス生成と RegisterRoutes への受け渡しを追加する

## 4. テスト

- [x] 4.1 `internal/handler/convert_test.go` を作成し、フォーマット変換・品質制御・バリデーション・エラーハンドリングのテストを実装する

## 5. 検証・コミット

- [x] 5.1 `golangci-lint run ./...` でlintエラーがないことを確認する
- [x] 5.2 `go test -v ./...` で全テストが通ることを確認する
- [ ] 5.3 変更をコミットする
