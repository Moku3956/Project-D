// Package game はsql-monsterの対戦ロジックを担う。Project-Dへは client パッケージ
// 経由でのみアクセスし、executor/planner/txn などの内部パッケージには触れない。
package game

import (
	"fmt"

	"github.com/Moku3956/Project-D/client"
	"github.com/Moku3956/Project-D/types"
)

// PlayerID は現状シングルプレイヤー前提のため固定。認証・複数プレイヤーは扱わない
// (docs/frontend_architecture.md参照)。
const PlayerID int64 = 1

// PlayerMaxHP はプレイヤーの最大HP。バランス調整前の暫定値。
const PlayerMaxHP int64 = 1000

// Monster は1体分のモンスター定義。この構造体がモンスターの正となる情報で、
// DBのレコードはここから投入・リセットする。
type Monster struct {
	ID         int64
	Name       string
	Level      int64 // 1〜6。ホーム画面のパス上の並び順と難易度帯に対応する
	HP         int64 // 最大HP。対戦中に減った分はバトル開始時にここまで戻す
	Weakness   string
	Weaknesses []Weakness
	Attacks    []MonsterAttack
}

// IsBoss はそのモンスターがボス(最終レベル)かを返す。
func (m Monster) IsBoss() bool { return m.Level == 6 }

// Weakness はmonster_weaknessesテーブルの1行に対応する。①攻撃用分析で
// プレイヤーがSELECTして弱点を探る対象。
type Weakness struct {
	DmgType  string // 例: "ACID"
	Severity int64  // 弱点の強さ(大きいほど有効)
}

// MonsterAttack はmonster_attacksテーブルの1行に対応する。
type MonsterAttack struct {
	Tag        string // 例: "SMASH_BLUNT"
	Likelihood int64  // 出やすさ(0-100)、値が大きいほど有力な候補
	Power      int64  // 防御に失敗した(ブロックできなかった)ときのダメージ量
}

// tables はsql-monsterが使うテーブルのDDL。存在確認用のSELECTと対にしてある。
var tables = []struct {
	name string
	ddl  string
}{
	{"players", `CREATE TABLE players (id INT PRIMARY KEY, hp INT)`},
	{"monsters", `CREATE TABLE monsters (id INT PRIMARY KEY, name VARCHAR(50), hp INT, weakness VARCHAR(20))`},
	{"monster_attacks", `CREATE TABLE monster_attacks (id INT PRIMARY KEY, monster_id INT, tag VARCHAR(20), likelihood INT, power INT)`},
	{"monster_weaknesses", `CREATE TABLE monster_weaknesses (id INT PRIMARY KEY, monster_id INT, dmg_type VARCHAR(20), severity INT)`},
	{"player_progress", `CREATE TABLE player_progress (monster_id INT PRIMARY KEY, cleared INT)`},
}

// SetupSchema はsql-monsterが使うテーブルを作成する。既に存在するテーブルは飛ばすので、
// 2回目以降の起動でもそのまま呼べる。
func SetupSchema(db *client.DB) error {
	for _, t := range tables {
		if tableExists(db, t.name) {
			continue
		}
		if _, err := db.Exec(t.ddl); err != nil {
			return fmt.Errorf("SetupSchema: %s: %w", t.name, err)
		}
	}
	return nil
}

// tableExists はテーブルへのSELECTが通るかどうかで存在を判定する。
// clientパッケージにはカタログを覗くAPIがないため、この方法をとっている。
func tableExists(db *client.DB, name string) bool {
	_, err := db.Exec(fmt.Sprintf("SELECT * FROM %s", name))
	return err == nil
}

// SeedAll は初回起動時にモンスター・プレイヤー・進行状況の初期レコードを投入する。
// 既にモンスターが入っていれば何もしない。
func SeedAll(db *client.DB) error {
	existing, err := db.Exec("SELECT id FROM monsters")
	if err != nil {
		return fmt.Errorf("SeedAll: %w", err)
	}
	if len(existing.Rows) > 0 {
		return nil
	}

	for _, m := range Monsters() {
		if err := SeedMonster(db, m); err != nil {
			return err
		}
		insert := fmt.Sprintf("INSERT INTO player_progress VALUES (%d, 0)", m.ID)
		if _, err := db.Exec(insert); err != nil {
			return fmt.Errorf("SeedAll: player_progress(%d): %w", m.ID, err)
		}
	}
	return SeedPlayer(db, PlayerID, PlayerMaxHP)
}

// SeedMonster はモンスター1体分のレコードをmonsters/monster_attacks/monster_weaknessesに投入する。
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
	for i, w := range m.Weaknesses {
		weaknessID := m.ID*1000 + int64(i)
		insertWeakness := fmt.Sprintf(
			"INSERT INTO monster_weaknesses VALUES (%d, %d, '%s', %d)",
			weaknessID, m.ID, w.DmgType, w.Severity,
		)
		if _, err := db.Exec(insertWeakness); err != nil {
			return fmt.Errorf("SeedMonster(%d): weakness %q: %w", m.ID, w.DmgType, err)
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

// ResetHP はモンスターとプレイヤーのHPを最大値に戻す。バトル開始・やり直しのたびに呼ぶ。
func ResetHP(db *client.DB, m Monster) error {
	if _, err := db.Exec(fmt.Sprintf("UPDATE monsters SET hp = %d WHERE id = %d", m.HP, m.ID)); err != nil {
		return fmt.Errorf("ResetHP: monster %d: %w", m.ID, err)
	}
	if _, err := db.Exec(fmt.Sprintf("UPDATE players SET hp = %d WHERE id = %d", PlayerMaxHP, PlayerID)); err != nil {
		return fmt.Errorf("ResetHP: player: %w", err)
	}
	return nil
}

// ClearedIDs はクリア済みのモンスターIDの集合を返す。
func ClearedIDs(db *client.DB) (map[int64]bool, error) {
	result, err := db.Exec("SELECT monster_id, cleared FROM player_progress")
	if err != nil {
		return nil, fmt.Errorf("ClearedIDs: %w", err)
	}
	cleared := make(map[int64]bool)
	for _, row := range result.Rows {
		id, ok := row.Values[0].(types.IntValue)
		if !ok {
			continue
		}
		flag, ok := row.Values[1].(types.IntValue)
		if !ok {
			continue
		}
		if flag.V != 0 {
			cleared[id.V] = true
		}
	}
	return cleared, nil
}

// MarkCleared は指定モンスターをクリア済みにする。
func MarkCleared(db *client.DB, monsterID int64) error {
	sql := fmt.Sprintf("UPDATE player_progress SET cleared = 1 WHERE monster_id = %d", monsterID)
	if _, err := db.Exec(sql); err != nil {
		return fmt.Errorf("MarkCleared(%d): %w", monsterID, err)
	}
	return nil
}

// ResourcesFor はモンスターのレベルに応じた1ターンぶんのリソース上限を返す。
// レベルが上がるほど分析に使える行数を絞り、無駄なクエリを許さなくする
// (spec.md「難易度スケーリング」)。数値はバランス調整前の暫定値。
func ResourcesFor(level int64) Resources {
	analysis := 13 - level // Lv1:12 … Lv6:7
	if analysis < 5 {
		analysis = 5
	}
	attack := 100 + level*20 // Lv1:120 … Lv6:220
	return Resources{Analysis: analysis, AttackDefense: attack}
}

// Monsters はゲームに登場するモンスターの定義一覧をレベル順で返す。
// 現状6体。今後増やす前提(docs/home_screen_ui.md参照)。
func Monsters() []Monster {
	return []Monster{
		{
			ID: 1, Name: "SLIME_v1", Level: 1, HP: 300, Weakness: "ACID",
			Weaknesses: []Weakness{
				{DmgType: "ACID", Severity: 8},
				{DmgType: "FIRE", Severity: 3},
			},
			Attacks: []MonsterAttack{
				{Tag: "SPLASH", Likelihood: 70, Power: 60},
				{Tag: "ABSORB", Likelihood: 30, Power: 40},
			},
		},
		{
			ID: 2, Name: "RUST_HOUND", Level: 2, HP: 400, Weakness: "WATER",
			Weaknesses: []Weakness{
				{DmgType: "WATER", Severity: 7},
				{DmgType: "SHOCK", Severity: 5},
			},
			Attacks: []MonsterAttack{
				{Tag: "BITE", Likelihood: 55, Power: 90},
				{Tag: "HOWL", Likelihood: 25, Power: 50},
				{Tag: "CHARGE", Likelihood: 20, Power: 120},
			},
		},
		{
			ID: 3, Name: "OBSIDIAN GOLEM_v2", Level: 3, HP: 600, Weakness: "ACID",
			Weaknesses: []Weakness{
				{DmgType: "ACID", Severity: 8},
				{DmgType: "WATER", Severity: 7},
				{DmgType: "SHOCK", Severity: 2},
			},
			Attacks: []MonsterAttack{
				{Tag: "SMASH_BLUNT", Likelihood: 68, Power: 130},
				{Tag: "LASER_BEAM", Likelihood: 12, Power: 240},
				{Tag: "RECHARGE_BUFF", Likelihood: 20, Power: 0},
			},
		},
		{
			ID: 4, Name: "VOID_WRAITH", Level: 4, HP: 800, Weakness: "FIRE",
			Weaknesses: []Weakness{
				{DmgType: "FIRE", Severity: 9},
				{DmgType: "ACID", Severity: 4},
			},
			Attacks: []MonsterAttack{
				{Tag: "DRAIN", Likelihood: 45, Power: 150},
				{Tag: "PHASE_SHIFT", Likelihood: 35, Power: 90},
				{Tag: "NIGHTMARE", Likelihood: 20, Power: 210},
			},
		},
		{
			ID: 5, Name: "MAGMA_TITAN", Level: 5, HP: 1000, Weakness: "WATER",
			Weaknesses: []Weakness{
				{DmgType: "WATER", Severity: 9},
				{DmgType: "SHOCK", Severity: 6},
				{DmgType: "FIRE", Severity: 1},
			},
			Attacks: []MonsterAttack{
				{Tag: "ERUPTION", Likelihood: 40, Power: 200},
				{Tag: "MAGMA_WAVE", Likelihood: 35, Power: 160},
				{Tag: "HARDEN", Likelihood: 25, Power: 60},
			},
		},
		{
			ID: 6, Name: "NULL_SOVEREIGN", Level: 6, HP: 1500, Weakness: "SHOCK",
			Weaknesses: []Weakness{
				{DmgType: "SHOCK", Severity: 9},
				{DmgType: "ACID", Severity: 6},
				{DmgType: "FIRE", Severity: 5},
				{DmgType: "WATER", Severity: 3},
			},
			Attacks: []MonsterAttack{
				{Tag: "TRUNCATE", Likelihood: 30, Power: 260},
				{Tag: "DEADLOCK", Likelihood: 30, Power: 180},
				{Tag: "CASCADE", Likelihood: 25, Power: 220},
				{Tag: "VACUUM", Likelihood: 15, Power: 120},
			},
		},
	}
}

// FindMonster はIDからモンスター定義を引く。
func FindMonster(id int64) (Monster, bool) {
	for _, m := range Monsters() {
		if m.ID == id {
			return m, true
		}
	}
	return Monster{}, false
}
