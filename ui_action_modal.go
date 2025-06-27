package main

import (
	"fmt"
	"image/color"
	"log"

	"github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
)

func createActionModalUI(game *Game) widget.PreferredSizeLocateableWidget {
	if len(game.actionQueue) == 0 {
		return nil
	}
	actingMedarot := game.actionQueue[0]
	c := game.Config.UI

	overlay := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
		widget.ContainerOpts.BackgroundImage(image.NewNineSliceColor(color.NRGBA{0, 0, 0, 180})),
	)

	panel := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(image.NewNineSliceColor(color.NRGBA{20, 20, 30, 255})),
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(c.ActionModal.ButtonSpacing),
			widget.RowLayoutOpts.Padding(widget.NewInsetsSimple(15)),
		)),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
				HorizontalPosition: widget.AnchorLayoutPositionCenter,
				VerticalPosition:   widget.AnchorLayoutPositionCenter,
			}),
			widget.WidgetOpts.MinSize(int(c.ActionModal.ButtonWidth)+30, 0),
		),
	)
	overlay.AddChild(panel)

	panel.AddChild(widget.NewText(
		widget.TextOpts.Text(fmt.Sprintf("行動選択: %s", actingMedarot.Name), game.MplusFont, c.Colors.White),
	))

	buttonImage := &widget.ButtonImage{
		Idle:    image.NewNineSliceColor(c.Colors.Gray),
		Hover:   image.NewNineSliceColor(color.RGBA{180, 180, 180, 255}),
		Pressed: image.NewNineSliceColor(color.RGBA{100, 100, 100, 255}),
	}

	availableParts := actingMedarot.GetAvailableAttackParts()
	if len(availableParts) == 0 {
		panel.AddChild(widget.NewText(
			widget.TextOpts.Text("利用可能なパーツがありません。", game.MplusFont, c.Colors.White),
		))
	}

	for _, part := range availableParts {
		capturedPart := part
		actionButton := widget.NewButton(
			widget.ButtonOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.RowLayoutData{
				Stretch: true,
			})),
			widget.ButtonOpts.Image(buttonImage),
			// [FIXED] ボタンのテキスト色指定を修正しました
			widget.ButtonOpts.Text(fmt.Sprintf("%s (%s)", capturedPart.PartName, capturedPart.Category), game.MplusFont, &widget.ButtonTextColor{
				Idle: c.Colors.White,
			}),
			widget.ButtonOpts.TextPadding(widget.NewInsetsSimple(5)),
			widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
				handleActionSelection(game, actingMedarot, capturedPart)
			}),
		)
		panel.AddChild(actionButton)
	}

	return overlay
}

func handleActionSelection(game *Game, actingMedarot *Medarot, selectedPart *Part) {
	target := game.team2Leader
	if actingMedarot.Team == Team2 {
		target = game.team1Leader
	}
	if target == nil || target.State == StateBroken {
		candidates := game.getTargetCandidates(actingMedarot)
		if len(candidates) > 0 {
			target = candidates[0]
		} else {
			game.enqueueMessage("ターゲットがいません！", nil)
			return
		}
	}
	actingMedarot.TargetedMedarot = target

	var slotKey PartSlotKey
	for s, p := range actingMedarot.Parts {
		if p.ID == selectedPart.ID {
			slotKey = s
			break
		}
	}

	if actingMedarot.SelectAction(slotKey) {
		hideUIActionModal(game)
		game.executeAction(actingMedarot)
		game.State = StatePlaying
	}
}

func showUIActionModal(g *Game) {
	if g.actionModal != nil {
		hideUIActionModal(g)
	}
	modal := createActionModalUI(g)
	if modal == nil {
		g.State = StatePlaying
		return
	}
	g.actionModal = modal
	g.ui.Container.AddChild(g.actionModal)
	log.Println("Action modal shown.")
}

func hideUIActionModal(g *Game) {
	if g.actionModal != nil {
		g.ui.Container.RemoveChild(g.actionModal)
		g.actionModal = nil
		log.Println("Action modal hidden.")
	}
}
