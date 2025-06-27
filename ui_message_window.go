package main

import (
	"image/color"

	"github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
)

func createMessageWindowUI(game *Game) widget.PreferredSizeLocateableWidget {
	c := game.Config.UI

	// [FIXED] Paddingにポインタではなく値そのものを渡すように修正しました
	overlay := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewAnchorLayout(
			widget.AnchorLayoutOpts.Padding(widget.Insets{Bottom: 50}),
		)),
		widget.ContainerOpts.BackgroundImage(image.NewNineSliceColor(color.NRGBA{0, 0, 0, 180})),
	)

	panel := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(image.NewNineSliceColor(color.NRGBA{20, 20, 30, 220})),
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Padding(widget.NewInsetsSimple(15)),
			widget.RowLayoutOpts.Spacing(10),
		)),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
				HorizontalPosition: widget.AnchorLayoutPositionCenter,
				VerticalPosition:   widget.AnchorLayoutPositionEnd,
			}),
			widget.WidgetOpts.MinSize(int(float32(c.Screen.Width)*0.8), 0),
		),
	)
	overlay.AddChild(panel)

	messageText := widget.NewText(
		widget.TextOpts.Text(game.message, game.MplusFont, c.Colors.White),
	)
	panel.AddChild(messageText)

	promptText := "クリックして続行..."
	if game.State == StateGameOver {
		promptText = "クリックしてリスタート"
	}
	panel.AddChild(widget.NewText(
		widget.TextOpts.Text(promptText, game.MplusFont, c.Colors.Yellow),
		widget.TextOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.RowLayoutData{
			Stretch:  true,
			Position: widget.RowLayoutPositionEnd,
		})),
	))

	return overlay
}

func showUIMessage(g *Game) {
	if g.messageWindow != nil {
		hideUIMessage(g)
	}
	msgWindow := createMessageWindowUI(g)
	g.messageWindow = msgWindow
	g.ui.Container.AddChild(g.messageWindow)
}

func hideUIMessage(g *Game) {
	if g.messageWindow != nil {
		g.ui.Container.RemoveChild(g.messageWindow)
		g.messageWindow = nil
	}
}
