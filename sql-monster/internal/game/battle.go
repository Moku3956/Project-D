package game

import (
	"fmt"
	"math/rand"

	"github.com/Moku3956/Project-D/client"
	"github.com/Moku3956/Project-D/types"
)

// healBonus は④防御で1行に絞った的中(ブロック率100%)をしたときのHP増加量。
// 具体的な値は未確定(sql-monster/docs/spec.md「未定事項」参照)、暫定値。
const healBonus = 10

// Resources はプレイヤーが1ターンに使えるリソース。
type Resources struct {
	Analysis      int64 // ①③④のSELECTで読める行数の合計
	AttackDefense int64 // ②の攻撃で実測できるHP差分の合計
}

// Battle はplayerIDとmonsterIDの対戦1回分を表す。
type Battle struct {
	db        *client.DB
	playerID  int64
	monsterID int64
	maxRes    Resources
	res       Resources

	// currentAttackID は今ターンの実際の攻撃。DBのどのテーブルにも書かず、
	// ここ(Goのメモリ)だけで保持する。sql-monster/docs/spec.md参照。
	currentAttackID int64
}

// NewBattle はplayerID・monsterIDの対戦を開始する。プレイヤー・モンスターの
// レコードは事前にSeedPlayer/SeedMonsterで投入済みであること。
func NewBattle(db *client.DB, playerID, monsterID int64, maxRes Resources) (*Battle, error) {
	b := &Battle{db: db, playerID: playerID, monsterID: monsterID, maxRes: maxRes}
	if err := b.NextTurn(); err != nil {
		return nil, err
	}
	return b, nil
}

// NextTurn はリソースを上限まで全回復し、今ターンの実際の攻撃を候補から選び直す。
func (b *Battle) NextTurn() error {
	b.res = b.maxRes

	sql := fmt.Sprintf("SELECT id FROM monster_attacks WHERE monster_id = %d", b.monsterID)
	result, err := b.db.Exec(sql)
	if err != nil {
		return err
	}
	if len(result.Rows) == 0 {
		return fmt.Errorf("monster %d has no attacks", b.monsterID)
	}
	idx := rand.Intn(len(result.Rows))
	b.currentAttackID = result.Rows[idx].Values[0].(types.IntValue).V
	return nil
}

// Resources は現在のリソース残量を返す。
func (b *Battle) Resources() Resources { return b.res }

// MaxResources は1ターンあたりのリソース上限を返す。
func (b *Battle) MaxResources() Resources { return b.maxRes }

// MonsterID は対戦相手のモンスターIDを返す。
func (b *Battle) MonsterID() int64 { return b.monsterID }

// AnalyzeWeakness はモンスターの弱点データに対するSELECTを実行する(①攻撃用分析)。
func (b *Battle) AnalyzeWeakness(sql string) (*client.Result, error) {
	return b.analyze(sql)
}

// AnalyzeDefense はモンスターの攻撃手がかりに対するSELECTを実行する(③防御用分析)。
func (b *Battle) AnalyzeDefense(sql string) (*client.Result, error) {
	return b.analyze(sql)
}

// analyze はSELECTを実行し、読んだ行数ぶん分析リソースを消費する。
func (b *Battle) analyze(sql string) (*client.Result, error) {
	result, err := b.db.Exec(sql)
	if err != nil {
		return nil, err
	}
	b.res.Analysis -= int64(len(result.Rows))
	return result, nil
}

// Attack はモンスターへの攻撃SQLをトランザクション内で実行し、実行前後のHP差分を
// 実測する(②攻撃)。差分が攻撃防御リソースの残量を超える場合はROLLBACKして
// 不発になる(dealt=0, ok=false, err=nil)。
//
// 現状の言語には算術式(hp - 50 等)がないため、呼び出し元のSQLは
// SET hp = <絶対値> の形で書く必要がある。project_issues.mdの
// 「算術演算子が言語に存在しない」が解消され次第、この制約はなくなる。
func (b *Battle) Attack(sql string) (dealt int64, ok bool, err error) {
	before, err := b.MonsterHP()
	if err != nil {
		return 0, false, err
	}

	tx := b.db.Begin()
	if _, err := tx.Exec(sql); err != nil {
		_ = tx.Rollback()
		return 0, false, err
	}

	afterResult, err := tx.Exec(fmt.Sprintf("SELECT hp FROM monsters WHERE id = %d", b.monsterID))
	if err != nil {
		_ = tx.Rollback()
		return 0, false, err
	}
	if len(afterResult.Rows) == 0 {
		_ = tx.Rollback()
		return 0, false, fmt.Errorf("monster %d not found", b.monsterID)
	}
	after := afterResult.Rows[0].Values[0].(types.IntValue).V

	diff := before - after
	if diff < 0 {
		diff = 0 // 回復方向の変化は攻撃として扱わない
	}

	if diff > b.res.AttackDefense {
		if err := tx.Rollback(); err != nil {
			return 0, false, err
		}
		return 0, false, nil
	}

	if err := tx.Commit(); err != nil {
		return 0, false, err
	}
	b.res.AttackDefense -= diff
	return diff, true, nil
}

// Defend はmonster_attacksに対するWHERE付きSELECTを実行する(④防御)。
// ブロック率(1/該当行数、含まれなければ0)を判定し、被ダメージをplayersに反映する。
// 精密な的中(ブロック率100%)ならHP増加ボーナスも与える。
func (b *Battle) Defend(sql string) (result *client.Result, blockRate float64, damageTaken int64, err error) {
	result, err = b.analyze(sql)
	if err != nil {
		return nil, 0, 0, err
	}

	hit := false
	for _, row := range result.Rows {
		if row.Values[0].(types.IntValue).V == b.currentAttackID {
			hit = true
			break
		}
	}

	power, err := b.currentAttackPower()
	if err != nil {
		return result, 0, 0, err
	}

	if !hit {
		damageTaken = power
		if err := b.applyPlayerDamage(damageTaken); err != nil {
			return result, 0, damageTaken, err
		}
		return result, 0, damageTaken, nil
	}

	blockRate = 1 / float64(len(result.Rows))
	damageTaken = power - int64(float64(power)*blockRate)
	if err := b.applyPlayerDamage(damageTaken); err != nil {
		return result, blockRate, damageTaken, err
	}
	if blockRate == 1 {
		if err := b.healPlayer(healBonus); err != nil {
			return result, blockRate, damageTaken, err
		}
	}
	return result, blockRate, damageTaken, nil
}

// IsOver はどちらかのHPが0になっていれば決着とみなし、プレイヤーが勝ったかを返す。
func (b *Battle) IsOver() (over bool, playerWon bool, err error) {
	playerHP, err := b.PlayerHP()
	if err != nil {
		return false, false, err
	}
	monsterHP, err := b.MonsterHP()
	if err != nil {
		return false, false, err
	}
	if monsterHP <= 0 {
		return true, true, nil
	}
	if playerHP <= 0 {
		return true, false, nil
	}
	return false, false, nil
}

// PlayerHP は現在のプレイヤーのHPを返す。
func (b *Battle) PlayerHP() (int64, error) {
	result, err := b.db.Exec(fmt.Sprintf("SELECT hp FROM players WHERE id = %d", b.playerID))
	if err != nil {
		return 0, err
	}
	if len(result.Rows) == 0 {
		return 0, fmt.Errorf("player %d not found", b.playerID)
	}
	return result.Rows[0].Values[0].(types.IntValue).V, nil
}

// MonsterHP は現在のモンスターのHPを返す。
func (b *Battle) MonsterHP() (int64, error) {
	result, err := b.db.Exec(fmt.Sprintf("SELECT hp FROM monsters WHERE id = %d", b.monsterID))
	if err != nil {
		return 0, err
	}
	if len(result.Rows) == 0 {
		return 0, fmt.Errorf("monster %d not found", b.monsterID)
	}
	return result.Rows[0].Values[0].(types.IntValue).V, nil
}

func (b *Battle) currentAttackPower() (int64, error) {
	sql := fmt.Sprintf("SELECT power FROM monster_attacks WHERE id = %d", b.currentAttackID)
	result, err := b.db.Exec(sql)
	if err != nil {
		return 0, err
	}
	if len(result.Rows) == 0 {
		return 0, fmt.Errorf("attack %d not found", b.currentAttackID)
	}
	return result.Rows[0].Values[0].(types.IntValue).V, nil
}

func (b *Battle) setPlayerHP(hp int64) error {
	_, err := b.db.Exec(fmt.Sprintf("UPDATE players SET hp = %d WHERE id = %d", hp, b.playerID))
	return err
}

func (b *Battle) applyPlayerDamage(amount int64) error {
	hp, err := b.PlayerHP()
	if err != nil {
		return err
	}
	hp -= amount
	if hp < 0 {
		hp = 0
	}
	return b.setPlayerHP(hp)
}

func (b *Battle) healPlayer(amount int64) error {
	hp, err := b.PlayerHP()
	if err != nil {
		return err
	}
	return b.setPlayerHP(hp + amount)
}
