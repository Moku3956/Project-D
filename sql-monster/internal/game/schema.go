// Package game はsql-monsterの対戦ロジックを担う。Project-Dへは client パッケージ
// 経由でのみアクセスし、executor/planner/txn などの内部パッケージには触れない。
package game

import (
	"fmt"

	"github.com/Moku3956/Project-D/client"
)

// Monster は低レベル層のモンスター定義。弱点・攻撃候補が直接カラムに現れる。
type Monster struct {
	ID       int64
	Name     string
	HP       int64
	Weakness string // 例: "fire"
	Attacks  []MonsterAttack
}

// MonsterAttack はmonster_attacksテーブルの1行に対応する。
type MonsterAttack struct {
	Tag        string // 例: "fire"
	Likelihood int64  // 出やすさ(0-100)、値が大きいほど有力な候補
	Power      int64  // 防御に失敗した(ブロックできなかった)ときのダメージ量
}

// SetupSchema はsql-monsterが使うテーブルを作成する。新規のdirに対して1回だけ呼ぶ想定。
func SetupSchema(db *client.DB) error {
	stmts := []string{
		`CREATE TABLE players (id INT PRIMARY KEY, hp INT)`,
		`CREATE TABLE monsters (id INT PRIMARY KEY, name VARCHAR(50), hp INT, weakness VARCHAR(20))`,
		`CREATE TABLE monster_attacks (id INT PRIMARY KEY, monster_id INT, tag VARCHAR(20), likelihood INT, power INT)`,
	}
	for _, sql := range stmts {
		if _, err := db.Exec(sql); err != nil {
			return fmt.Errorf("SetupSchema: %s: %w", sql, err)
		}
	}
	return nil
}

// SeedMonster はモンスター1体分のレコードをmonsters/monster_attacksに投入する。
func SeedMonster(db *client.DB, m Monster) error {
	insertMonster := fmt.Sprintf(
		"INSERT INTO monsters VALUES (%d, '%s', %d, '%s')",
		m.ID, m.Name, m.HP, m.Weakness,
	)
	if _, err := db.Exec(insertMonster); err != nil {
		return fmt.Errorf("SeedMonster(%d): %w", m.ID, err)
	}
	for i, a := range m.Attacks {
		attackID := m.ID*1000 + int64(i) // モンスターIDと採番をまとめて一意にする簡易採番
		insertAttack := fmt.Sprintf(
			"INSERT INTO monster_attacks VALUES (%d, %d, '%s', %d, %d)",
			attackID, m.ID, a.Tag, a.Likelihood, a.Power,
		)
		if _, err := db.Exec(insertAttack); err != nil {
			return fmt.Errorf("SeedMonster(%d): attack %q: %w", m.ID, a.Tag, err)
		}
	}
	return nil
}

// SeedPlayer はプレイヤー1人分のレコードをplayersに投入する。
func SeedPlayer(db *client.DB, playerID, hp int64) error {
	sql := fmt.Sprintf("INSERT INTO players VALUES (%d, %d)", playerID, hp)
	if _, err := db.Exec(sql); err != nil {
		return fmt.Errorf("SeedPlayer(%d): %w", playerID, err)
	}
	return nil
}
