package main

import (
	"log"
	"net/http"
	"os"

	"github.com/Moku3956/Project-D/api"
	"github.com/Moku3956/Project-D/catalog"
	"github.com/Moku3956/Project-D/executor"
	"github.com/Moku3956/Project-D/infrastructure"
	"github.com/Moku3956/Project-D/sql/planner"
	"github.com/Moku3956/Project-D/storage/page"
	"github.com/Moku3956/Project-D/storage/wal"
)

func main() {
	dbPath := envOr("DB_PATH", "data/mydb.db")
	walPath := envOr("WAL_PATH", "data/mydb.wal")
	catalogPath := envOr("CATALOG_PATH", "data/catalog.json")
	addr := envOr("ADDR", ":8080")

	if err := os.MkdirAll("data", 0755); err != nil {
		log.Fatalf("data ディレクトリの作成に失敗: %v", err)
	}

	dm, err := page.NewDiskManager(dbPath)
	if err != nil {
		log.Fatalf("DiskManager の初期化に失敗: %v", err)
	}
	defer dm.Close() //nolint:errcheck

	wm, err := wal.NewWALManager(walPath)
	if err != nil {
		log.Fatalf("WALManager の初期化に失敗: %v", err)
	}
	defer wm.Close() //nolint:errcheck

	cat, err := catalog.NewCatalog(catalogPath)
	if err != nil {
		log.Fatalf("Catalog の初期化に失敗: %v", err)
	}

	repo, err := infrastructure.NewBTreeRepository(dm)
	if err != nil {
		log.Fatalf("Repository の初期化に失敗: %v", err)
	}

	// 起動時にカタログ上の全テーブルをrepositoryに登録する
	for _, name := range cat.TableNames() {
		schema, err := cat.GetSchema(name)
		if err != nil {
			log.Fatalf("スキーマ取得に失敗: %v", err)
		}
		if err := repo.OpenTable(name, schema); err != nil {
			log.Fatalf("テーブルのオープンに失敗: %v", err)
		}
	}

	pl := planner.New(cat)
	eng := executor.NewEngine(repo, cat)

	mux := http.NewServeMux()
	api.NewHandler(pl, eng).RegisterRoutes(mux)

	log.Printf("サーバー起動: %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("サーバーエラー: %v", err)
	}
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
