// sql-monsterのエントリポイント。Project-Dのclientパッケージ経由でDBを開き、
// フロントエンド(React)向けのHTTP APIを提供する。
package main

import (
	"log"
	"net/http"
	"os"

	"github.com/Moku3956/Project-D/client"
	"github.com/Moku3956/Project-D/sql-monster/internal/api"
	"github.com/Moku3956/Project-D/sql-monster/internal/game"
)

func main() {
	// Project-D本体のサーバー(cmd/server)が data/ を使うため、既定値を分けている。
	// 同じディレクトリを共有すると、片方が書いたファイルをもう片方が読んでしまう。
	dir := envOr("DATA_DIR", "sql-monster/data")
	addr := envOr("ADDR", ":8081")

	db, err := client.Open(dir)
	if err != nil {
		log.Fatalf("client.Open に失敗: %v", err)
	}
	defer db.Close() //nolint:errcheck

	if err := game.SetupSchema(db); err != nil {
		log.Fatalf("SetupSchema に失敗: %v", err)
	}
	if err := game.SeedAll(db); err != nil {
		log.Fatalf("SeedAll に失敗: %v", err)
	}

	mux := http.NewServeMux()
	api.NewServer(db).RegisterRoutes(mux)

	log.Printf("sql-monster: %s で起動しました", addr)
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
