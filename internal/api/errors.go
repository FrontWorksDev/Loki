package api

// commonErrorCodes は画像処理エンドポイント (compress / convert) で
// OpenAPI 仕様に明示する共通エラーレスポンスの HTTP ステータスコード一覧を返す。
//
// Huma は Operation.Errors に列挙されたコードを自動的に
// application/problem+json + huma.ErrorModel スキーマで Responses に追加する
// (huma v2 の defineErrors の挙動)。これにより各 Operation が返し得るエラー
// 形状が OpenAPI 上で自己文書化され、クライアント実装側で型を生成できる。
//
// ヘルスチェックエンドポイントはミドルウェア exempt 対象であり想定外エラーが
// 発生しないため、本リストは適用しない。
func commonErrorCodes() []int {
	return []int{
		400, // Bad Request: 非対応フォーマット、不正な画像データ
		413, // Payload Too Large: ファイルサイズ上限超過
		422, // Unprocessable Entity: 必須フィールド未指定等
		429, // Too Many Requests: レートリミット超過
		500, // Internal Server Error: 想定外のサーバーエラー
	}
}
