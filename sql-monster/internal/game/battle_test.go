package game

import (
	"fmt"
	"testing"

	"github.com/Moku3956/Project-D/client"
)

// ---- ヘルパー ----

func setupBattle(t *testing.T, maxRes Resources) (*client.DB, *Battle) {
	t.Helper()
	dir := t.TempDir()
	db, err := client.Open(dir)
	if err != nil {
		t.Fatalf("Open error: %v", err)
	}
	t.Cleanup(func() { db.Close() }) //nolint:errcheck

	if err := SetupSchema(db); err != nil {
		t.Fatalf("SetupSchema error: %v", err)
	}
	if err := SeedPlayer(db, 1, 100); err != nil {
		t.Fatalf("SeedPlayer error: %v", err)
	}
	monster := Monster{
		ID:       1,
		Name:     "スライム",
		HP:       100,
		Weakness: "fire",
		Attacks: []MonsterAttack{
			{Tag: "fire", Likelihood: 60, Power: 20},
			{Tag: "ice", Likelihood: 40, Power: 15},
		},
	}
	if err := SeedMonster(db, monster); err != nil {
		t.Fatalf("SeedMonster error: %v", err)
	}

	battle, err := NewBattle(db, 1, 1, maxRes)
	if err != nil {
		t.Fatalf("NewBattle error: %v", err)
	}
	return db, battle
}

// ---- 正常系 ----

func TestNewBattleInitializesResources(t *testing.T) {
	_, battle := setupBattle(t, Resources{Analysis: 50, AttackDefense: 50})

	res := battle.Resources()
	if res.Analysis != 50 || res.AttackDefense != 50 {
		t.Errorf("Resources = %+v, want {50 50}", res)
	}
}

func TestAnalyzeWeaknessConsumesAnalysisResource(t *testing.T) {
	_, battle := setupBattle(t, Resources{Analysis: 50, AttackDefense: 50})

	result, err := battle.AnalyzeWeakness("SELECT * FROM monsters WHERE id = 1")
	if err != nil {
		t.Fatalf("AnalyzeWeakness error: %v", err)
	}
	if len(result.Rows) != 1 {
		t.Fatalf("レコード数 = %d, want 1", len(result.Rows))
	}
	if battle.Resources().Analysis != 49 {
		t.Errorf("Analysis = %d, want 49", battle.Resources().Analysis)
	}
}

func TestAttackDealsDamageAndConsumesResource(t *testing.T) {
	_, battle := setupBattle(t, Resources{Analysis: 50, AttackDefense: 50})

	dealt, ok, err := battle.Attack("UPDATE monsters SET hp = 70 WHERE id = 1")
	if err != nil {
		t.Fatalf("Attack error: %v", err)
	}
	if !ok {
		t.Fatal("ok = false, want true")
	}
	if dealt != 30 {
		t.Errorf("dealt = %d, want 30", dealt)
	}
	if battle.Resources().AttackDefense != 20 {
		t.Errorf("AttackDefense = %d, want 20", battle.Resources().AttackDefense)
	}

	hp, err := battle.MonsterHP()
	if err != nil {
		t.Fatalf("MonsterHP error: %v", err)
	}
	if hp != 70 {
		t.Errorf("MonsterHP = %d, want 70", hp)
	}
}

func TestAttackRollsBackWhenOverBudget(t *testing.T) {
	_, battle := setupBattle(t, Resources{Analysis: 50, AttackDefense: 10})

	dealt, ok, err := battle.Attack("UPDATE monsters SET hp = 70 WHERE id = 1") // 差分30 > 予算10
	if err != nil {
		t.Fatalf("Attack error: %v", err)
	}
	if ok {
		t.Fatal("ok = true, want false(不発)")
	}
	if dealt != 0 {
		t.Errorf("dealt = %d, want 0", dealt)
	}
	if battle.Resources().AttackDefense != 10 {
		t.Errorf("AttackDefense = %d, want 10(消費されない)", battle.Resources().AttackDefense)
	}

	hp, err := battle.MonsterHP()
	if err != nil {
		t.Fatalf("MonsterHP error: %v", err)
	}
	if hp != 100 {
		t.Errorf("MonsterHP = %d, want 100(ROLLBACKされている)", hp)
	}
}

func TestDefendPrecisePreventsAllDamageAndHeals(t *testing.T) {
	_, battle := setupBattle(t, Resources{Analysis: 50, AttackDefense: 50})

	before, err := battle.PlayerHP()
	if err != nil {
		t.Fatalf("PlayerHP error: %v", err)
	}

	// 同一パッケージ内のテストなのでcurrentAttackID(非公開)に直接アクセスし、
	// 今ターンの実際の攻撃を1件だけ狙い撃つクエリを組み立てる。
	sql := fmt.Sprintf("SELECT id FROM monster_attacks WHERE id = %d", battle.currentAttackID)
	_, blockRate, damageTaken, err := battle.Defend(sql)
	if err != nil {
		t.Fatalf("Defend error: %v", err)
	}
	if blockRate != 1 {
		t.Errorf("blockRate = %v, want 1", blockRate)
	}
	if damageTaken != 0 {
		t.Errorf("damageTaken = %d, want 0", damageTaken)
	}

	after, err := battle.PlayerHP()
	if err != nil {
		t.Fatalf("PlayerHP error: %v", err)
	}
	if after != before+healBonus {
		t.Errorf("PlayerHP = %d, want %d(healBonus分増加)", after, before+healBonus)
	}
}

func TestDefendMissTakesFullDamage(t *testing.T) {
	_, battle := setupBattle(t, Resources{Analysis: 50, AttackDefense: 50})

	before, err := battle.PlayerHP()
	if err != nil {
		t.Fatalf("PlayerHP error: %v", err)
	}

	// 存在しないIDで外す
	_, blockRate, damageTaken, err := battle.Defend("SELECT id FROM monster_attacks WHERE id = 999999")
	if err != nil {
		t.Fatalf("Defend error: %v", err)
	}
	if blockRate != 0 {
		t.Errorf("blockRate = %v, want 0", blockRate)
	}
	if damageTaken <= 0 {
		t.Errorf("damageTaken = %d, want > 0", damageTaken)
	}

	after, err := battle.PlayerHP()
	if err != nil {
		t.Fatalf("PlayerHP error: %v", err)
	}
	if after != before-damageTaken {
		t.Errorf("PlayerHP = %d, want %d", after, before-damageTaken)
	}
}

func TestNextTurnResetsResources(t *testing.T) {
	_, battle := setupBattle(t, Resources{Analysis: 50, AttackDefense: 50})

	if _, err := battle.AnalyzeWeakness("SELECT * FROM monsters WHERE id = 1"); err != nil {
		t.Fatalf("AnalyzeWeakness error: %v", err)
	}
	if battle.Resources().Analysis == 50 {
		t.Fatal("リソースが消費されていない(テスト前提が崩れている)")
	}

	if err := battle.NextTurn(); err != nil {
		t.Fatalf("NextTurn error: %v", err)
	}
	if battle.Resources().Analysis != 50 {
		t.Errorf("Analysis = %d, want 50(全回復)", battle.Resources().Analysis)
	}
}

func TestIsOverDetectsMonsterDefeat(t *testing.T) {
	_, battle := setupBattle(t, Resources{Analysis: 50, AttackDefense: 200})

	if _, _, err := battle.Attack("UPDATE monsters SET hp = 0 WHERE id = 1"); err != nil {
		t.Fatalf("Attack error: %v", err)
	}

	over, playerWon, err := battle.IsOver()
	if err != nil {
		t.Fatalf("IsOver error: %v", err)
	}
	if !over || !playerWon {
		t.Errorf("over=%v playerWon=%v, want true/true", over, playerWon)
	}
}

// ---- 異常系 ----

func TestAttackParseErrorReturnsError(t *testing.T) {
	_, battle := setupBattle(t, Resources{Analysis: 50, AttackDefense: 50})

	if _, _, err := battle.Attack("UPDATE monsters SET"); err == nil {
		t.Fatal("パースエラーが期待されたがnil")
	}
}
