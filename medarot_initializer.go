package main

import (
	"log"
)

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
			if p := findPartByID(gameData.AllParts, partID); p != nil {
				medarot.Parts[slot] = p
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

func findPartByID(allParts map[string]*Part, id string) *Part {
	originalPart, exists := allParts[id]
	if !exists {
		return nil
	}
	newPart := *originalPart
	return &newPart
}

func findMedalByID(allMedals []Medal, id string) *Medal {
	for _, medal := range allMedals {
		if medal.ID == id {
			newMedal := medal
			return &newMedal
		}
	}
	return nil
}
