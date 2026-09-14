// db-internal-appのエントリポイント。セッションごとに独立したProject-D
// インスタンス(dbsession.Session)を持つHTTP APIサーバーを起動する。
package main

import (
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/Moku3956/Project-D/db-internal-app/internal/api"
)

func main() {
	// 本体サーバー(cmd/server)・sql-monsterと同じディレクトリを共有すると
	// 互いのファイルを読み書きしてしまうため、既定値を分けている
	// (sql-monsterの同種のバグ、project_issuesメモリ参照)。
	dataDir := envOr("DATA_DIR", "db-internal-app/data")
	addr := envOr("ADDR", ":8082")
	// フロントエンドとバックエンドが別オリジンにデプロイされる本番/ステージング
	// 環境ではSECURE_COOKIES=trueを設定する(セッションCookieがクロスオリジンの
	// fetchでも送られるようになる)。ローカル開発はHTTPのままなので既定は無効。
	secureCookies := envOr("SECURE_COOKIES", "false") == "true"
	// フロントエンドを配信しているOriginだけを許可する。本番/ステージングでは
	// CloudFrontのドメインをカンマ区切りで指定する。ローカル開発(Vite)向けに
	// http://localhost:5173をデフォルトで許可しておく。
	allowedOrigins := strings.Split(envOr("ALLOWED_ORIGINS", "http://localhost:5173"), ",")

	srv := api.NewServer(dataDir, secureCookies)
	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)

	log.Printf("db-internal-app: %s で起動しました", addr)
	if err := http.ListenAndServe(addr, api.WithCORS(allowedOrigins)(mux)); err != nil {
		log.Fatalf("サーバーの起動に失敗: %v", err)
	}
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
