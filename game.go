package main

import (
	"fmt"
	"log"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// ... (Game構造体、NewGame, Updateは変更なし) ...
type Game struct {
	GameData              *GameData
	Config                Config
	MplusFont             text.Face
	TickCount             int
	DebugMode             bool
	State                 GameState
	PlayerTeam            TeamID
	actionQueue           []*Medarot
	sortedMedarotsForDraw []*Medarot
	Medarots              []*Medarot
	team1Leader           *Medarot
	team2Leader           *Medarot
	ui                    *UI
	message               string
	postMessageCallback   func()
	winner                TeamID
	restartRequested      bool
	playerMedarotToAct    *Medarot
}

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
		playerMedarotToAct:    nil,
	}
	g.Medarots = InitializeAllMedarots(g.GameData)
	if len(g.Medarots) == 0 {
		log.Fatal("No medarots were initialized.")
	}
	g.initializeMedarotLists()
	g.ui = NewUI(g)
	log.Println("Game initialized successfully.")
	return g
}
func (g *Game) Update() error {
	g.ui.ebitenui.Update()
	if g.restartRequested {
		g.restartRequested = false
	}
	switch g.State {
	case StatePlaying:
		g.TickCount++
		g.updateProgress()
		g.processReadyQueue()
		g.processIdleMedarots()
		g.checkGameEnd()
		updateAllInfoPanels(g)
		if g.ui.battlefieldWidget != nil {
			g.ui.battlefieldWidget.UpdatePositions()
		}
	case StatePlayerActionSelect:
		if g.ui.actionModal == nil && g.playerMedarotToAct != nil {
			g.ui.ShowActionModal(g, g.playerMedarotToAct)
		}
	case StateMessage, StateGameOver:
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			if g.State == StateMessage {
				g.ui.HideMessageWindow()
				if g.postMessageCallback != nil {
					g.postMessageCallback()
					g.postMessageCallback = nil
				}
				g.State = StatePlaying
				g.processIdleMedarots()
			} else if g.State == StateGameOver {
			}
		}
	}
	return nil
}

// [ADDED] getTargetCandidatesメソッドをラッパーとして復活
func (g *Game) getTargetCandidates(actingMedarot *Medarot) []*Medarot {
	// 内部でai.goの関数を呼び出す
	return getTargetCandidates(g, actingMedarot)
}

// (以下の関数は変更なし)
func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(g.Config.UI.Colors.Background)
	g.ui.ebitenui.Draw(screen)
	bf := g.ui.battlefieldWidget
	if bf != nil {
		bf.DrawBackground(screen)
		bf.DrawIcons(screen)
		bf.DrawDebug(screen)
	}
	if g.DebugMode {
		ebitenutil.DebugPrint(screen, fmt.Sprintf("TPS: %0.2f\nFPS: %0.2f\nState: %s",
			ebiten.ActualTPS(), ebiten.ActualFPS(), g.State))
	}
}
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
func (g *Game) updateProgress() {
	for _, m := range g.Medarots {
		if m.State != StateCharging && m.State != StateCooldown {
			continue
		}
		m.ProgressCounter++
		if m.TotalDuration > 0 {
			m.Gauge = (m.ProgressCounter / m.TotalDuration) * 100
		} else {
			m.Gauge = 100
		}
		if m.ProgressCounter >= m.TotalDuration {
			if m.State == StateCharging {
				m.ChangeState(StateReady)
				g.actionQueue = append(g.actionQueue, m)
				log.Printf("%s のチャージが完了。実行キューに追加。", m.Name)
			} else if m.State == StateCooldown {
				m.ChangeState(StateIdle)
			}
		}
	}
}
func (g *Game) processReadyQueue() {
	if len(g.actionQueue) == 0 {
		return
	}
	sort.SliceStable(g.actionQueue, func(i, j int) bool {
		return g.actionQueue[i].GetOverallPropulsion() > g.actionQueue[j].GetOverallPropulsion()
	})
	for len(g.actionQueue) > 0 {
		actingMedarot := g.actionQueue[0]
		g.actionQueue = g.actionQueue[1:]
		actingMedarot.ExecuteAction(&g.Config.Balance)
		g.enqueueMessage(actingMedarot.LastActionLog, func() {
			if actingMedarot.State != StateBroken {
				actingMedarot.StartCooldown(&g.Config.Balance)
			}
		})
		return
	}
}
func (g *Game) processIdleMedarots() {
	if g.playerMedarotToAct != nil || g.State != StatePlaying {
		return
	}
	for _, m := range g.Medarots {
		if m.State == StateIdle && m.Team != g.PlayerTeam {
			aiSelectAction(g, m)
		}
	}
	nextPlayerMedarot := g.findNextIdlePlayerMedarot()
	if nextPlayerMedarot != nil {
		g.playerMedarotToAct = nextPlayerMedarot
		g.State = StatePlayerActionSelect
	}
}
func (g *Game) findNextIdlePlayerMedarot() *Medarot {
	for _, m := range g.Medarots {
		if m.Team == g.PlayerTeam && m.State == StateIdle {
			return m
		}
	}
	return nil
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
	// チーム1リーダーの頭部が破壊されているか、またはチーム2が全滅した場合
if g.team1Leader.GetPart(PartSlotHead).IsBroken || team2Func == 0 {
    g.winner = Team2
    g.State = StateGameOver
    // チーム1リーダーが機能停止したことをメッセージに追加するとより分かりやすい
    g.enqueueMessage(fmt.Sprintf("%sが機能停止！ チーム2の勝利！", g.team1Leader.Name), nil)
// チーム2リーダーの頭部が破壊されているか、またはチーム1が全滅した場合
} else if g.team2Leader.GetPart(PartSlotHead).IsBroken || team1Func == 0 {
    g.winner = Team1
    g.State = StateGameOver
    g.enqueueMessage(fmt.Sprintf("%sが機能停止！ チーム1の勝利！", g.team2Leader.Name), nil)
}
}
func (g *Game) enqueueMessage(msg string, callback func()) {
	g.message = msg
	g.postMessageCallback = callback
	g.State = StateMessage
	g.ui.ShowMessageWindow(g)
}
