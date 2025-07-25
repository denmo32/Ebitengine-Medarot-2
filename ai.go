package main

import (
	"log"
	"sort"
)

// aiSelectAction はAIメダロットの行動を決定し、チャージを開始させる
func aiSelectAction(game *Game, medarot *Medarot) {
	// 攻撃可能なパーツを取得
	availableParts := medarot.GetAvailableAttackParts()
	if len(availableParts) == 0 {
		log.Printf("%s: AIは攻撃可能なパーツがないため待機。", medarot.Name)
		return
	}

	// 攻撃対象の候補を取得
	targetCandidates := getTargetCandidates(game, medarot)
	if len(targetCandidates) == 0 {
		log.Printf("%s: AIは攻撃対象がいないため待機。", medarot.Name)
		return
	}

	// === シンプルなAI思考ルーチン ===
	// TODO: ここをより賢いロジックに拡張する
	// 今回は、利用可能な最初のパーツで、相手のリーダーを優先して狙う

	// 1. ターゲット選択
	var target *Medarot
	for _, cand := range targetCandidates {
		if cand.IsLeader {
			target = cand
			break
		}
	}
	if target == nil { // リーダーがいない（破壊された）場合
		target = targetCandidates[0]
	}

	// 2. 使用パーツ選択
	selectedPart := availableParts[0]

	// 3. 選択したパーツのスロットキーを取得
	var slotKey PartSlotKey
	for s, p := range medarot.Parts {
		if p.ID == selectedPart.ID {
			slotKey = s
			break
		}
	}

	// 4. 行動を決定し、チャージを開始
	medarot.SelectAndStartCharge(slotKey, target, &game.Config.Balance)
}

// getTargetCandidates は指定されたメダロットの攻撃対象候補リストを返す
func getTargetCandidates(game *Game, actingMedarot *Medarot) []*Medarot {
	candidates := []*Medarot{}
	var opponentTeamID TeamID = Team2
	if actingMedarot.Team == Team2 {
		opponentTeamID = Team1
	}

	for _, m := range game.Medarots {
		if m.Team == opponentTeamID && m.State != StateBroken {
			candidates = append(candidates, m)
		}
	}

	// 描画順でソートして、常に同じ優先順位でターゲットを選ぶようにする
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].DrawIndex < candidates[j].DrawIndex
	})
	return candidates
}