package main

import (
	// [FIXED] "math"は使用されていないため、import文を削除しました
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// drawBattlefield はバトルフィールドの描画を行います。
func drawBattlefield(screen *ebiten.Image, g *Game) {
	width, height := float32(screen.Bounds().Dx()), float32(screen.Bounds().Dy())
	vector.StrokeRect(screen, 0, 0, width, height, g.Config.UI.Battlefield.LineWidth, g.Config.UI.Colors.Gray, false)

	team1HomeX := width * 0.1
	team2HomeX := width * 0.9
	team1ExecX := width * 0.4
	team2ExecX := width * 0.6

	for i := 0; i < PlayersPerTeam; i++ {
		yPos := (height / float32(PlayersPerTeam+1)) * (float32(i) + 1)
		vector.StrokeCircle(screen, team1HomeX, yPos, g.Config.UI.Battlefield.HomeMarkerRadius, g.Config.UI.Battlefield.LineWidth, g.Config.UI.Colors.Gray, true)
		vector.StrokeCircle(screen, team2HomeX, yPos, g.Config.UI.Battlefield.HomeMarkerRadius, g.Config.UI.Battlefield.LineWidth, g.Config.UI.Colors.Gray, true)
	}

	vector.StrokeLine(screen, team1ExecX, 0, team1ExecX, height, g.Config.UI.Battlefield.LineWidth, g.Config.UI.Colors.White, true)
	vector.StrokeLine(screen, team2ExecX, 0, team2ExecX, height, g.Config.UI.Battlefield.LineWidth, g.Config.UI.Colors.White, true)
}

// drawMedarotIcons はメダロットアイコンの描画を行います。
func drawMedarotIcons(screen *ebiten.Image, g *Game) {
	height := float32(screen.Bounds().Dy())

	for _, medarot := range g.sortedMedarotsForDraw {
		iconColor := g.Config.UI.Colors.Gray
		if medarot.Team == Team1 {
			iconColor = g.Config.UI.Colors.Team1
		} else {
			iconColor = g.Config.UI.Colors.Team2
		}
		if medarot.State == StateBroken {
			iconColor = g.Config.UI.Colors.Broken
		}

		yPos := (height / float32(PlayersPerTeam+1)) * (float32(medarot.DrawIndex) + 1)
		xPos := calculateIconX(medarot, screen.Bounds().Dx())

		vector.DrawFilledCircle(screen, xPos, yPos, g.Config.UI.Battlefield.IconRadius, iconColor, true)
		if medarot.IsLeader {
			vector.StrokeCircle(screen, xPos, yPos, g.Config.UI.Battlefield.IconRadius+3, 2, g.Config.UI.Colors.Leader, true)
		}
	}
}

// calculateIconX はバトルフィールド内でのX座標を計算します。
func calculateIconX(medarot *Medarot, battlefieldWidth int) float32 {
	progress := float32(medarot.Gauge / 100.0)
	width := float32(battlefieldWidth)

	homeX, execX := width*0.1, width*0.4
	if medarot.Team == Team2 {
		homeX, execX = width*0.9, width*0.6
	}

	switch medarot.State {
	case StateReadyToSelectAction: // ゲージチャージ中
		return homeX + (execX-homeX)*progress
	case StateReadyToExecuteAction: // 実行位置
		return execX
	case StateActionCooldown: // クールダウン中
		return execX - (execX-homeX)*progress
	default:
		return homeX
	}
}
