// sql-monsterのエントリポイント。現時点ではDBの初期化とモンスター1体の
// 投入まで(HTTP APIは未実装、別途着手する)。
package main

import (
	"log"
	"os"

	"github.com/Moku3956/Project-D/client"
	"github.com/Moku3956/Project-D/sql-monster/internal/game"
)

func main() {
	dir := envOr("DATA_DIR", "data")

	db, err := client.Open(dir)
	if err != nil {
		log.Fatalf("client.Open に失敗: %v", err)
	}
	defer db.Close() //nolint:errcheck

	// NOTE: 2回目以降の起動(dataが既存)ではCREATE TABLEがエラーになる。
	// テーブル存在確認の仕組みがまだないため、現時点では新規のDATA_DIRでのみ動作する。
	if err := game.SetupSchema(db); err != nil {
		log.Fatalf("SetupSchema に失敗: %v", err)
	}

	log.Println("sql-monster: 起動完了")
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
