package main

import (
	"fmt"
	"image/color"
	"log"
	"sort"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
)

// NewGame はゲームの UI を初期化して返します。
func NewGame(gameData *GameData, config Config, font text.Face) *Game {
	g := &Game{
		GameData:              gameData,
		Config:                config,
		MplusFont:             font,
		TickCount:             0,
		DebugMode:             true,
		State:                 StatePlaying,
		PlayerTeam:            Team1,
		actionQueue:           make([]*Medarot, 0),
		sortedMedarotsForDraw: make([]*Medarot, 0),
	}

	g.Medarots = InitializeAllMedarots(g.GameData)
	if len(g.Medarots) == 0 {
		log.Fatal("No medarots were initialized.")
	}
	g.initializeMedarotLists()

	// --- UIの構築 ---
	rootContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewStackedLayout()),
	)

	// mainUIContainerをGridLayoutに変更し、情報パネルとバトルフィールドを並べて配置します。
	// 3つの列: チーム1パネル (固定幅), バトルフィールド (伸縮), チーム2パネル (固定幅)
	mainUIContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewGridLayout(
			widget.GridLayoutOpts.Columns(3),
			// 1列目 (チーム1パネル): 伸縮しない
			// 2列目 (バトルフィールド): 水平方向に伸縮
			// 3列目 (チーム2パネル): 伸縮しない
			widget.GridLayoutOpts.Stretch([]bool{false, true, false}, []bool{true}),
			widget.GridLayoutOpts.Spacing(g.Config.UI.InfoPanel.Padding, 0),
		)),
		// [FIXED] ここにあったPadding設定を完全に削除
	)
	rootContainer.AddChild(mainUIContainer)

	// チーム1の情報パネルのコンテナを1列目に配置
	team1PanelContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(g.Config.UI.InfoPanel.Padding),
		)),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.MinSize(int(g.Config.UI.InfoPanel.BlockWidth), 0), // 明示的に最小幅を設定
			widget.WidgetOpts.LayoutData(widget.GridLayoutData{
				HorizontalPosition: widget.GridLayoutPositionCenter,
				VerticalPosition:   widget.GridLayoutPositionCenter,
			}),
		),
	)
	mainUIContainer.AddChild(team1PanelContainer)

	// バトルフィールド描画領域を2列目に配置し、残りのスペースを埋めるようにStretchを設定します。
	g.battlefieldContainer = widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(image.NewNineSliceColor(color.Transparent)),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.GridLayoutData{
				HorizontalPosition: widget.GridLayoutPositionCenter,
				VerticalPosition:   widget.GridLayoutPositionCenter,
			}),
		),
	)
	mainUIContainer.AddChild(g.battlefieldContainer)

	// チーム2の情報パネルのコンテナを3列目に配置
	team2PanelContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(g.Config.UI.InfoPanel.Padding),
		)),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.MinSize(int(g.Config.UI.InfoPanel.BlockWidth), 0), // 明示的に最小幅を設定
			widget.WidgetOpts.LayoutData(widget.GridLayoutData{
				HorizontalPosition: widget.GridLayoutPositionCenter,
				VerticalPosition:   widget.GridLayoutPositionCenter,
			}),
		),
	)
	mainUIContainer.AddChild(team2PanelContainer)

	// メダロット情報パネルを生成して配置
	medarotInfoPanelUIs = make(map[string]*infoPanelUI)
	for _, m := range g.Medarots {
		panelUI := createSingleMedarotInfoPanel(g, m)
		medarotInfoPanelUIs[m.ID] = panelUI
		if m.Team == Team1 {
			team1PanelContainer.AddChild(panelUI.rootContainer)
		} else {
			team2PanelContainer.AddChild(panelUI.rootContainer)
		}
	}

	g.ui = &ebitenui.UI{
		Container: rootContainer,
	}

	log.Println("Game initialized successfully.")
	return g
}

// Update はゲームのロジックを更新します
func (g *Game) Update() error {
	g.ui.Update() // UIの状態を更新します

	if g.restartRequested {
		// (リスタート処理は未実装)
		g.restartRequested = false
	}

	switch g.State {
	case StatePlaying:
		g.TickCount++
		g.updateMedarotGauges()
		g.processActions()
		g.checkGameEnd()
		updateAllInfoPanels(g)
	case StatePlayerActionSelect:
		if g.actionModal == nil {
			showUIActionModal(g)
		}
	case StateMessage, StateGameOver:
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			if g.State == StateMessage {
				hideUIMessage(g)
				if g.postMessageCallback != nil {
					g.postMessageCallback()
					g.postMessageCallback = nil
				}
				g.State = StatePlaying
			} else if g.State == StateGameOver {
				// (リスタート処理は未実装)
			}
		}
	}

	return nil
}

// Draw はゲーム画面を描画します
func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(g.Config.UI.Colors.Background)

	if g.battlefieldContainer != nil {
		bfRect := g.battlefieldContainer.GetWidget().Rect
		if bfRect.Dx() > 0 && bfRect.Dy() > 0 {
			battlefieldImage := screen.SubImage(bfRect).(*ebiten.Image)
			drawBattlefield(battlefieldImage, g)
			drawMedarotIcons(battlefieldImage, g)
		}
	}

	g.ui.Draw(screen) // UIを描画します

	if g.DebugMode {
		ebitenutil.DebugPrint(screen, fmt.Sprintf("TPS: %0.2f\nFPS: %0.2f\nState: %s",
			ebiten.ActualTPS(), ebiten.ActualFPS(), g.State))
	}
}

// Layout は画面サイズ変更時に呼び出されます
func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return g.Config.UI.Screen.Width, g.Config.UI.Screen.Height
}

func (g *Game) initializeMedarotLists() {
	g.sortedMedarotsForDraw = make([]*Medarot, len(g.Medarots))
	copy(g.sortedMedarotsForDraw, g.Medarots)
	sort.Slice(g.sortedMedarotsForDraw, func(i, j int) bool {
		if g.sortedMedarotsForDraw[i].Team != g.sortedMedarotsForDraw[j].Team {
			return g.sortedMedarotsForDraw[i].Team < g.sortedMedarotsForDraw[j].Team
		}
		return g.sortedMedarotsForDraw[i].DrawIndex < g.sortedMedarotsForDraw[j].DrawIndex
	})

	for _, m := range g.Medarots {
		if m.IsLeader {
			if m.Team == Team1 {
				g.team1Leader = m
			} else {
				g.team2Leader = m
			}
		}
	}
}

func (g *Game) updateMedarotGauges() {
	for _, m := range g.Medarots {
		if m.State != StateReadyToSelectAction {
			continue
		}

		propulsionFactor := float64(m.GetOverallPropulsion()) * g.Config.Balance.Time.PropulsionEffectRate
		m.Gauge += (1.0 + propulsionFactor) * g.Config.Balance.Time.OverallTimeDivisor

		if m.Gauge >= 100 {
			m.Gauge = 100
			m.ChangeState(StateReadyToExecuteAction)
			// 重複追加を防ぐ
			inQueue := false
			for _, qm := range g.actionQueue {
				if qm.ID == m.ID {
					inQueue = true
					break
				}
			}
			if !inQueue {
				g.actionQueue = append(g.actionQueue, m)
				log.Printf("%s の行動準備が完了し、キューに追加されました。", m.Name)
			}
		}
	}
}

func (g *Game) processActions() {
	if len(g.actionQueue) == 0 || g.State != StatePlaying {
		return
	}

	// キューの先頭から行動するメダロットを選ぶ
	sort.SliceStable(g.actionQueue, func(i, j int) bool {
		// チーム2 (AI) を優先
		if g.actionQueue[i].Team != g.actionQueue[j].Team {
			return g.actionQueue[i].Team > g.actionQueue[j].Team
		}
		return g.actionQueue[i].Gauge > g.actionQueue[j].Gauge
	})

	actingMedarot := g.actionQueue[0]

	if actingMedarot.State == StateBroken {
		g.actionQueue = g.actionQueue[1:]
		return
	}
	if actingMedarot.State != StateReadyToExecuteAction {
		return
		// PlayerActionSelect状態への遷移は、プレイヤーチームの行動準備ができた場合のみ行う
	}
	if actingMedarot.Team == g.PlayerTeam {
		g.State = StatePlayerActionSelect
		return
	}

	// AIの行動
	g.aiSelectAction(actingMedarot)
	g.executeAction(actingMedarot)
}

func (g *Game) executeAction(actingMedarot *Medarot) {
	actingMedarot.ExecuteAction(&g.Config.Balance)
	g.enqueueMessage(actingMedarot.LastActionLog, func() {
		actingMedarot.ChangeState(StateActionCooldown)
		go g.handleCooldown(actingMedarot)
	})
	// キューから削除
	for i, m := range g.actionQueue {
		if m.ID == actingMedarot.ID {
			g.actionQueue = append(g.actionQueue[:i], g.actionQueue[i+1:]...)
			break
		}
	}
}

func (g *Game) aiSelectAction(medarot *Medarot) {
	availableParts := medarot.GetAvailableAttackParts()
	if len(availableParts) == 0 {
		log.Printf("%s: AIは攻撃可能なパーツがありません。クールダウンへ。", medarot.Name)
		medarot.ChangeState(StateActionCooldown)
		go g.handleCooldown(medarot)
		return
	}
	targetCandidates := g.getTargetCandidates(medarot)
	if len(targetCandidates) == 0 {
		log.Printf("%s: AIは攻撃対象がいません。クールダウンへ。", medarot.Name)
		medarot.ChangeState(StateActionCooldown)
		go g.handleCooldown(medarot)
		return
	}

	medarot.TargetedMedarot = targetCandidates[0]
	selectedPart := availableParts[0]

	var slotKey PartSlotKey
	for s, p := range medarot.Parts {
		if p.ID == selectedPart.ID {
			slotKey = s
			break
		}
	}
	medarot.SelectAction(slotKey)
}

func (g *Game) getTargetCandidates(actingMedarot *Medarot) []*Medarot {
	candidates := []*Medarot{}
	var opponentTeamID TeamID = Team2
	if actingMedarot.Team == Team2 {
		opponentTeamID = Team1
	}
	for _, m := range g.Medarots {
		if m.Team == opponentTeamID && m.State != StateBroken {
			candidates = append(candidates, m)
		}
	}
	return candidates
}

func (g *Game) checkGameEnd() {
	if g.State == StateGameOver {
		return
	}
	team1Func := 0
	team2Func := 0
	for _, m := range g.Medarots {
		if m.State != StateBroken {
			if m.Team == Team1 {
				team1Func++
			} else {
				team2Func++
			}
		}
	}

	if g.team1Leader.State == StateBroken || team2Func == 0 {
		g.winner = Team2
		g.State = StateGameOver
		g.enqueueMessage("チーム2の勝利！", nil)
	} else if g.team2Leader.State == StateBroken || team1Func == 0 {
		g.winner = Team1
		g.State = StateGameOver
		g.enqueueMessage("チーム1の勝利！", nil)
	}
}

func (g *Game) enqueueMessage(msg string, callback func()) {
	g.message = msg
	g.postMessageCallback = callback
	g.State = StateMessage
	showUIMessage(g)
}

func (g *Game) handleCooldown(medarot *Medarot) {
	part := medarot.GetPart(medarot.SelectedPartKey)
	if part == nil {
		medarot.ChangeState(StateReadyToSelectAction)
		return
	}
	duration := float64(part.Cooldown)
	if duration <= 0 {
		duration = 1
	}

	tickDuration := duration * 60.0 * g.Config.Balance.Time.OverallTimeDivisor
	for i := 0.0; i < tickDuration; i++ {
		if medarot.State != StateActionCooldown {
			return
		}
		medarot.Gauge = (i / tickDuration) * 100
		time.Sleep(16 * time.Millisecond)
	}
	medarot.ChangeState(StateReadyToSelectAction)
}