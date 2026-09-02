// db-internal-appのエントリポイント。セッションごとに独立したProject-D
// インスタンス(dbsession.Session)を持つHTTP APIサーバーを起動する。
package main

import (
	"log"
	"net/http"
	"os"

	"github.com/Moku3956/Project-D/db-internal-app/internal/api"
)

func main() {
	// 本体サーバー(cmd/server)・sql-monsterと同じディレクトリを共有すると
	// 互いのファイルを読み書きしてしまうため、既定値を分けている
	// (sql-monsterの同種のバグ、project_issuesメモリ参照)。
	dataDir := envOr("DATA_DIR", "db-internal-app/data")
	addr := envOr("ADDR", ":8082")

	srv := api.NewServer(dataDir)
	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)

	log.Printf("db-internal-app: %s で起動しました", addr)
	if err := http.ListenAndServe(addr, api.WithCORS(mux)); err != nil {
		log.Fatalf("サーバーの起動に失敗: %v", err)
	}
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
