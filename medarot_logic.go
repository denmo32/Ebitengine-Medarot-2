package main

import (
	"fmt"
	"log"
	"math/rand"
)

func NewMedarot(id, name string, team TeamID, medal *Medal, isLeader bool, drawIndex int) *Medarot {
	return &Medarot{
		ID:        id,
		Name:      name,
		Team:      team,
		Medal:     medal,
		Parts:     make(map[PartSlotKey]*Part),
		IsLeader:  isLeader,
		DrawIndex: drawIndex,
		State:     StateReadyToSelectAction,
		Gauge:     0.0,
	}
}

func (m *Medarot) ChangeState(newState MedarotState) {
	if m.State == newState {
		return
	}
	log.Printf("%s のステートが %s から %s に変更されました。", m.Name, m.State, newState)
	m.State = newState
	switch newState {
	case StateReadyToSelectAction:
		m.Gauge = 0
		m.SelectedPartKey = ""
	case StateActionCooldown:
		m.Gauge = 0
	case StateBroken:
		m.Gauge = 0
	}
}

func (m *Medarot) GetPart(slotKey PartSlotKey) *Part {
	part, exists := m.Parts[slotKey]
	if !exists || part == nil { // 壊れていてもパーツ情報自体は返す
		return nil
	}
	return part
}

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

func (m *Medarot) SelectAction(partKey PartSlotKey) bool {
	part := m.GetPart(partKey)
	if part == nil || part.IsBroken {
		log.Printf("%s: 選択されたパーツ %s は存在しないか破壊されています。", m.Name, partKey)
		return false
	}
	m.SelectedPartKey = partKey
	log.Printf("%sは%sを選択しました。", m.Name, part.PartName)
	return true
}

func (m *Medarot) ExecuteAction(balanceConfig *BalanceConfig) {
	if m.SelectedPartKey == "" || m.TargetedMedarot == nil {
		m.LastActionLog = fmt.Sprintf("%sは行動に失敗した。", m.Name)
		return
	}
	part := m.GetPart(m.SelectedPartKey)
	target := m.TargetedMedarot

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

	m.handlePostAttack()
}

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

func (m *Medarot) GetOverallPropulsion() int {
	legs := m.GetPart(PartSlotLegs)
	if legs == nil || legs.IsBroken {
		return 1 // 最低保証
	}
	return legs.Propulsion
}

func (m *Medarot) GetOverallMobility() int {
	legs := m.GetPart(PartSlotLegs)
	if legs == nil || legs.IsBroken {
		return 1 // 最低保証
	}
	return legs.Mobility
}

func (m *Medarot) applyDamage(part *Part, damage int) {
	part.Armor -= damage
	if part.Armor <= 0 {
		part.Armor = 0
		part.IsBroken = true
	}
}

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

func (m *Medarot) handlePostAttack() {
	if head := m.GetPart(PartSlotHead); head != nil && head.IsBroken {
		m.ChangeState(StateBroken)
	}
}

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
