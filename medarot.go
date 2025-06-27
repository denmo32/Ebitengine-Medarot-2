package main

import (
	"fmt"
	"log"
	"math/rand"
)

// =============================================================================
// 初期化 (Initializer)
// =============================================================================

// InitializeAllMedarots は全てのメダロットデータをロードし、インスタンスを生成する
func InitializeAllMedarots(gameData *GameData) []*Medarot {
	var allMedarots []*Medarot

	for _, loadout := range gameData.Medarots {
		medal := findMedalByID(gameData.Medals, loadout.MedalID)
		if medal == nil {
			log.Printf("警告: メダルID '%s' が見つかりません。'%s'にはデフォルトメダルを使用します。", loadout.MedalID, loadout.Name)
			medal = &Medal{ID: "fallback", Name: "フォールバック", SkillLevel: 1}
		}

		medarot := NewMedarot(
			loadout.ID,
			loadout.Name,
			loadout.Team,
			medal,
			loadout.IsLeader,
			loadout.DrawIndex,
		)

		partIDMap := map[PartSlotKey]string{
			PartSlotHead:     loadout.HeadID,
			PartSlotRightArm: loadout.RightArmID,
			PartSlotLeftArm:  loadout.LeftArmID,
			PartSlotLegs:     loadout.LegsID,
		}

		for slot, partID := range partIDMap {
			if p, exists := gameData.AllParts[partID]; exists {
				// パーツの状態をリセットするために、ここでコピーを行うのが最も安全
				newPart := *p
				newPart.IsBroken = false
				medarot.Parts[slot] = &newPart
			} else {
				log.Printf("警告: パーツID '%s' が見つかりません。'%s'の%sスロットは空になります。", partID, medarot.Name, slot)
				placeholderPart := &Part{ID: "placeholder", PartName: "なし", IsBroken: true}
				medarot.Parts[slot] = placeholderPart
			}
		}
		allMedarots = append(allMedarots, medarot)
	}

	log.Printf("%d体のメダロットを初期化しました。", len(allMedarots))
	return allMedarots
}

// findMedalByID はIDでメダルを探す（コピーを返す）
func findMedalByID(allMedals []Medal, id string) *Medal {
	for _, medal := range allMedals {
		if medal.ID == id {
			newMedal := medal
			return &newMedal
		}
	}
	return nil
}

// =============================================================================
// メダロットのメソッド (Methods)
// =============================================================================

// NewMedarot は新しいメダロットのインスタンスを生成する（コンストラクタ）
func NewMedarot(id, name string, team TeamID, medal *Medal, isLeader bool, drawIndex int) *Medarot {
	return &Medarot{
		ID:        id,
		Name:      name,
		Team:      team,
		Medal:     medal,
		Parts:     make(map[PartSlotKey]*Part),
		IsLeader:  isLeader,
		DrawIndex: drawIndex,
		State:     StateIdle,
		Gauge:     0.0,
	}
}

// ChangeState はメダロットの状態を変更し、関連するカウンターをリセットする
func (m *Medarot) ChangeState(newState MedarotState) {
	if m.State == newState {
		return
	}
	log.Printf("%s のステートが %s から %s に変更されました。", m.Name, m.State, newState)
	m.State = newState

	switch newState {
	case StateIdle:
		m.Gauge = 0
		m.ProgressCounter = 0
		m.TotalDuration = 0
		m.SelectedPartKey = ""
		m.TargetedMedarot = nil
	case StateReady:
		m.Gauge = 100
	case StateBroken:
		m.Gauge = 0
	}
}

// SelectAndStartCharge は行動を選択し、チャージを開始する
func (m *Medarot) SelectAndStartCharge(partKey PartSlotKey, target *Medarot, balanceConfig *BalanceConfig) bool {
	part := m.GetPart(partKey)
	if part == nil || part.IsBroken {
		log.Printf("%s: 選択されたパーツ %s は存在しないか破壊されています。", m.Name, partKey)
		return false
	}
	if target == nil || target.State == StateBroken {
		log.Printf("%s: ターゲットが存在しないか破壊されています。", m.Name)
		return false
	}

	m.SelectedPartKey = partKey
	m.TargetedMedarot = target
	log.Printf("%sは%sで%sを狙う！", m.Name, part.PartName, target.Name)

	baseSeconds := float64(part.Charge)
	if baseSeconds <= 0 {
		baseSeconds = 0.1
	}
	propulsionFactor := 1.0 + (float64(m.GetOverallPropulsion()) * balanceConfig.Time.PropulsionEffectRate)
	totalTicks := (baseSeconds * 60.0) / (balanceConfig.Time.GameSpeedMultiplier * propulsionFactor)

	m.TotalDuration = totalTicks
	if m.TotalDuration < 1 {
		m.TotalDuration = 1
	}

	m.ChangeState(StateCharging)
	return true
}

// StartCooldown はクールダウンを開始する
func (m *Medarot) StartCooldown(balanceConfig *BalanceConfig) {
	part := m.GetPart(m.SelectedPartKey)
	baseSeconds := 1.0
	if part != nil {
		baseSeconds = float64(part.Cooldown)
	}
	if baseSeconds <= 0 {
		baseSeconds = 0.1
	}

	totalTicks := (baseSeconds * 60.0) / balanceConfig.Time.GameSpeedMultiplier

	m.TotalDuration = totalTicks
	if m.TotalDuration < 1 {
		m.TotalDuration = 1
	}

	m.ProgressCounter = 0
	m.Gauge = 0

	m.ChangeState(StateCooldown)
}

// ExecuteAction は選択された行動を実行する
func (m *Medarot) ExecuteAction(balanceConfig *BalanceConfig) {
	if m.SelectedPartKey == "" || m.TargetedMedarot == nil {
		m.LastActionLog = fmt.Sprintf("%sは行動に失敗した。", m.Name)
		return
	}
	part := m.GetPart(m.SelectedPartKey)
	target := m.TargetedMedarot
	if target.State == StateBroken {
		m.LastActionLog = fmt.Sprintf("%sは%sを狙ったが、既に行動不能だった！", m.Name, target.Name)
		return
	}
	log.Printf("%s が %s を実行！", m.Name, part.PartName)
	isHit := m.calculateHit(part, target, balanceConfig)
	if isHit {
		damage, isCritical := m.calculateDamage(part, balanceConfig)
		targetPart := target.selectRandomPartToDamage()
		if targetPart != nil {
			target.applyDamage(targetPart, damage)
			m.LastActionLog = m.generateActionLog(target, targetPart, damage, isCritical)
		} else {
			m.LastActionLog = fmt.Sprintf("%sの攻撃は%sに当たらなかった。", m.Name, target.Name)
		}
	} else {
		m.LastActionLog = fmt.Sprintf("%sの%s攻撃は%sに外れた！", m.Name, part.PartName, target.Name)
	}
	
	// 将来の拡張で、行動の結果自身がダメージを受ける効果（カウンター、反動など）によって
    // 頭部が破壊された場合を考慮し、自身の状態をチェックする。
	if head := m.GetPart(PartSlotHead); head != nil && head.IsBroken {
		m.ChangeState(StateBroken)
	}
}

// =============================================================================
// ヘルパーメソッド (Helpers)
// =============================================================================

// GetPart は指定されたスロットのパーツを取得する
func (m *Medarot) GetPart(slotKey PartSlotKey) *Part {
	part, exists := m.Parts[slotKey]
	if !exists || part == nil {
		return nil
	}
	return part
}

// GetAvailableAttackParts は攻撃に使用可能なパーツのリストを取得する
func (m *Medarot) GetAvailableAttackParts() []*Part {
	var availableParts []*Part
	slotsToConsider := []PartSlotKey{PartSlotHead, PartSlotRightArm, PartSlotLeftArm}
	for _, slot := range slotsToConsider {
		part := m.GetPart(slot)
		if part != nil && !part.IsBroken && part.Category != CategoryNone {
			availableParts = append(availableParts, part)
		}
	}
	return availableParts
}

// GetOverallPropulsion は脚部の推進力を取得する
func (m *Medarot) GetOverallPropulsion() int {
	legs := m.GetPart(PartSlotLegs)
	if legs == nil || legs.IsBroken {
		return 1
	}
	return legs.Propulsion
}

// GetOverallMobility は脚部の機動力を取得する
func (m *Medarot) GetOverallMobility() int {
	legs := m.GetPart(PartSlotLegs)
	if legs == nil || legs.IsBroken {
		return 1
	}
	return legs.Mobility
}

// applyDamage はパーツにダメージを適用する
func (m *Medarot) applyDamage(part *Part, damage int) {
    part.Armor -= damage
    if part.Armor <= 0 {
        part.Armor = 0
        part.IsBroken = true
        
        // もし破壊されたのが頭パーツなら、即座に機能停止状態にする
        if part.Type == PartTypeHead {
            m.ChangeState(StateBroken)
        }
    }
}

// calculateHit は命中判定を行う
func (m *Medarot) calculateHit(part *Part, target *Medarot, balanceConfig *BalanceConfig) bool {
	baseChance := balanceConfig.Hit.BaseChance
	accuracyBonus := part.Accuracy / 2
	evasionPenalty := target.GetOverallMobility() / 2
	chance := baseChance + accuracyBonus - evasionPenalty
	switch part.Trait {
	case TraitAim:
		chance += balanceConfig.Hit.TraitAimBonus
	case TraitStrike:
		chance += balanceConfig.Hit.TraitStrikeBonus
	case TraitBerserk:
		chance += balanceConfig.Hit.TraitBerserkDebuff
	}
	if chance < 10 {
		chance = 10
	} else if chance > 95 {
		chance = 95
	}
	roll := rand.Intn(100)
	log.Printf("命中判定: %s -> %s | 命中率: %d, ロール: %d", m.Name, target.Name, chance, roll)
	return roll < chance
}

// calculateDamage はダメージ計算を行う
func (m *Medarot) calculateDamage(part *Part, balanceConfig *BalanceConfig) (int, bool) {
	baseDamage := part.Power
	isCritical := false
	criticalChance := m.Medal.SkillLevel * 2
	if rand.Intn(100) < criticalChance {
		baseDamage = int(float64(baseDamage) * balanceConfig.Damage.CriticalMultiplier)
		isCritical = true
	}
	baseDamage += m.Medal.SkillLevel * balanceConfig.Damage.MedalSkillFactor
	return baseDamage, isCritical
}

// generateActionLog は行動ログの文字列を生成する
func (m *Medarot) generateActionLog(target *Medarot, part *Part, damage int, isCritical bool) string {
	logMsg := fmt.Sprintf("%sの%sに%dダメージ！", target.Name, part.PartName, damage)
	if isCritical {
		logMsg = fmt.Sprintf("%sの%sにクリティカル！ %dダメージ！", target.Name, part.PartName, damage)
	}
	if part.IsBroken {
		logMsg += " パーツを破壊した！"
	}
	return logMsg
}

// selectRandomPartToDamage はダメージを受けるパーツをランダムに選択する
func (m *Medarot) selectRandomPartToDamage() *Part {
	vulnerable := []*Part{}
	slots := []PartSlotKey{PartSlotHead, PartSlotRightArm, PartSlotLeftArm, PartSlotLegs}
	for _, s := range slots {
		if part := m.GetPart(s); part != nil && !part.IsBroken {
			vulnerable = append(vulnerable, part)
		}
	}
	if len(vulnerable) == 0 {
		return nil
	}
	return vulnerable[rand.Intn(len(vulnerable))]
}
