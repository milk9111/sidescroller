package scenes

import (
	"bytes"
	"fmt"
	"log"
	"time"

	"image"
	"image/color"
	"image/draw"
	"image/gif"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	textv2 "github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/milk9111/sidescroller/assets"
	"github.com/milk9111/sidescroller/common"
	"golang.org/x/image/font/gofont/goregular"
)

type IntroSceneState int

const (
	IntroSceneQuote IntroSceneState = iota
	IntroSceneGIFZoomedOut
	IntroSceneGIFZoomedIn
	IntroSceneLanding
	IntroSceneBlackScreen
	IntroSceneDone
)

const (
	IntroQuote               = "\"Our past will bring about their future.\nRefute it, and suffer destruction.\"\n- The Prophet"
	introQuoteDurationFrames = 420
	introQuoteFadeFrames     = 60
)

type IntroScene struct {
	frameCount int

	windAudioPlayer    *audio.Player
	landingAudioPlayer *audio.Player

	quoteUI   *ebitenui.UI
	quoteText *widget.Text

	framesZoomedOut []*ebiten.Image
	framesZoomedIn  []*ebiten.Image
	delaysZoomedOut []int
	delaysZoomedIn  []int
	current         int
	elapsed         time.Duration
	lastTick        time.Time

	lastState IntroSceneState
	state     IntroSceneState
}

func NewIntroScene() *IntroScene {
	framesZoomedOut, delaysZoomedOut, err := loadGIF("intro_falling_zoomed_out_loop.gif")
	if err != nil {
		log.Fatalf("failed to load intro GIF: %v", err)
	}

	framesZoomedIn, delaysZoomedIn, err := loadGIF("intro_falling_loop.gif")
	if err != nil {
		log.Fatalf("failed to load intro GIF: %v", err)
	}

	windAudioPlayer, err := assets.LoadAudioPlayer("intro_long_fall.wav")
	if err != nil {
		log.Fatalf("failed to load intro sound: %v", err)
	}

	landingAudioPlayer, err := assets.LoadAudioPlayer("intro_long_fall_landing.wav")
	if err != nil {
		log.Fatalf("failed to load intro landing sound: %v", err)
	}

	quoteUI, quoteText, err := newIntroQuoteUI()
	if err != nil {
		log.Fatalf("failed to build intro quote UI: %v", err)
	}

	return &IntroScene{
		framesZoomedOut:    framesZoomedOut,
		framesZoomedIn:     framesZoomedIn,
		delaysZoomedOut:    delaysZoomedOut,
		delaysZoomedIn:     delaysZoomedIn,
		current:            0,
		lastTick:           time.Now(),
		windAudioPlayer:    windAudioPlayer,
		landingAudioPlayer: landingAudioPlayer,
		quoteUI:            quoteUI,
		quoteText:          quoteText,
		state:              IntroSceneGIFZoomedOut,
	}
}

func (s *IntroScene) Update() (string, error) {
	var nextState IntroSceneState
	switch s.state {
	case IntroSceneQuote:
		nextState = s.updateQuote()
	case IntroSceneGIFZoomedOut:
		nextState = s.updateGIFZoomedOut()
	case IntroSceneGIFZoomedIn:
		nextState = s.updateGIFZoomedIn()
	case IntroSceneLanding:
		nextState = s.updateLanding()
	case IntroSceneBlackScreen:
		nextState = s.updateBlackScreen()
	case IntroSceneDone:
		return SceneGame, nil
	}

	if s.state != nextState {
		s.lastTick = time.Now()
		s.frameCount = 0
		s.lastState = s.state
	}
	s.state = nextState

	return SceneIntro, nil
}

func (s *IntroScene) updateQuote() IntroSceneState {
	if s.quoteUI != nil {
		s.quoteUI.Update()
	}

	if s.frameCount >= introQuoteDurationFrames {
		s.frameCount = 0
		s.lastTick = time.Now()
		return IntroSceneBlackScreen
	}
	s.frameCount++

	return IntroSceneQuote
}

func (s *IntroScene) updateGIFZoomedOut() IntroSceneState {
	if !s.windAudioPlayer.IsPlaying() {
		s.windAudioPlayer.Play()
	}

	s.frameCount++
	if s.frameCount >= 150 {
		s.frameCount = 0
		s.elapsed = 0
		s.current = 0
		s.windAudioPlayer.Pause()
		return IntroSceneBlackScreen
	}

	now := time.Now()
	s.elapsed += now.Sub(s.lastTick)
	s.lastTick = now

	frameDuration := time.Duration(s.delaysZoomedOut[s.current]) * 10 * time.Millisecond
	if s.elapsed >= frameDuration {
		s.current = (s.current + 1) % len(s.framesZoomedOut)
		s.elapsed -= frameDuration
	}

	return IntroSceneGIFZoomedOut
}

func (s *IntroScene) updateGIFZoomedIn() IntroSceneState {
	if !s.windAudioPlayer.IsPlaying() {
		s.windAudioPlayer.Play()
	}

	s.frameCount++
	if s.frameCount >= 300 {
		s.frameCount = 0
		s.windAudioPlayer.Pause()
		s.windAudioPlayer.Close()
		return IntroSceneLanding
	}

	now := time.Now()
	s.elapsed += now.Sub(s.lastTick)
	s.lastTick = now

	frameDuration := time.Duration(s.delaysZoomedIn[s.current]) * 10 * time.Millisecond
	if s.elapsed >= frameDuration {
		s.current = (s.current + 1) % len(s.framesZoomedIn)
		s.elapsed -= frameDuration
	}

	return IntroSceneGIFZoomedIn
}

func (s *IntroScene) updateLanding() IntroSceneState {
	if !s.landingAudioPlayer.IsPlaying() {
		s.landingAudioPlayer.Play()
	}

	s.frameCount++
	if s.frameCount >= 60 {
		s.frameCount = 0
		s.landingAudioPlayer.Pause()
		s.landingAudioPlayer.Close()
		return IntroSceneBlackScreen
	}

	return IntroSceneLanding
}

func (s *IntroScene) updateBlackScreen() IntroSceneState {
	if s.frameCount >= 60 {
		if s.lastState == IntroSceneQuote {
			return IntroSceneGIFZoomedIn
		} else if s.lastState == IntroSceneGIFZoomedOut {
			return IntroSceneQuote
		} else {
			return IntroSceneDone
		}
	}
	s.frameCount++

	return IntroSceneBlackScreen
}

func (s *IntroScene) Draw(screen *ebiten.Image) {
	screen.Fill(color.Black)

	switch s.state {
	case IntroSceneQuote:
		s.drawQuote(screen)
	case IntroSceneGIFZoomedOut:
		screen.DrawImage(s.framesZoomedOut[s.current], nil)
	case IntroSceneGIFZoomedIn:
		screen.DrawImage(s.framesZoomedIn[s.current], nil)
	}
}

func (s *IntroScene) drawQuote(screen *ebiten.Image) {
	if s.quoteUI == nil || s.quoteText == nil {
		return
	}

	alpha := quoteAlpha(s.frameCount)
	if alpha <= 0 {
		return
	}

	s.quoteText.SetColor(color.NRGBA{R: 236, G: 240, B: 250, A: uint8(alpha * 255)})
	s.quoteUI.Draw(screen)
}

func (s *IntroScene) LayoutF(outsideWidth, outsideHeight float64) (float64, float64) {
	return common.BaseWidth, common.BaseHeight
}

func (s *IntroScene) Layout(outsideWidth, outsideHeight int) (int, int) {
	return common.BaseWidth, common.BaseHeight
}

func loadGIF(path string) ([]*ebiten.Image, []int, error) {
	b, err := assets.LoadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("load GIF: %w", err)
	}

	g, err := gif.DecodeAll(bytes.NewReader(b))
	if err != nil {
		return nil, nil, fmt.Errorf("decode GIF: %w", err)
	}

	delays := g.Delay
	canvasBounds := image.Rect(0, 0, g.Config.Width, g.Config.Height)
	canvas := image.NewNRGBA(canvasBounds)
	background := gifBackgroundColor(g)

	images := make([]*ebiten.Image, len(g.Image))
	for i, frame := range g.Image {
		previous := cloneNRGBA(canvas)
		draw.Draw(canvas, frame.Bounds(), frame, frame.Bounds().Min, draw.Over)
		images[i] = ebiten.NewImageFromImage(cloneNRGBA(canvas))

		disposal := byte(0)
		if i < len(g.Disposal) {
			disposal = g.Disposal[i]
		}

		switch disposal {
		case gif.DisposalBackground:
			draw.Draw(canvas, frame.Bounds(), &image.Uniform{C: background}, image.Point{}, draw.Src)
		case gif.DisposalPrevious:
			canvas = previous
		}
	}

	return images, delays, nil
}

func cloneNRGBA(src *image.NRGBA) *image.NRGBA {
	clone := image.NewNRGBA(src.Bounds())
	draw.Draw(clone, clone.Bounds(), src, src.Bounds().Min, draw.Src)
	return clone
}

func gifBackgroundColor(g *gif.GIF) color.Color {
	if len(g.Image) == 0 {
		return color.Transparent
	}

	index := int(g.BackgroundIndex)
	if index < len(g.Image[0].Palette) {
		return g.Image[0].Palette[index]
	}

	return color.Transparent
}

func newIntroQuoteUI() (*ebitenui.UI, *widget.Text, error) {
	fontSource, err := textv2.NewGoTextFaceSource(bytes.NewReader(goregular.TTF))
	if err != nil {
		return nil, nil, fmt.Errorf("load intro quote font: %w", err)
	}

	face := textv2.Face(&textv2.GoTextFace{Source: fontSource, Size: 22})
	root := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)

	quote := widget.NewText(
		widget.TextOpts.Text(IntroQuote, &face, color.NRGBA{R: 236, G: 240, B: 250, A: 255}),
		widget.TextOpts.Position(widget.TextPositionCenter, widget.TextPositionCenter),
		widget.TextOpts.MaxWidth(common.BaseWidth-160),
		widget.TextOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
			HorizontalPosition: widget.AnchorLayoutPositionCenter,
			VerticalPosition:   widget.AnchorLayoutPositionCenter,
		})),
	)

	root.AddChild(quote)
	return &ebitenui.UI{Container: root}, quote, nil
}

func quoteAlpha(frame int) float32 {
	if frame < 0 || frame >= introQuoteDurationFrames {
		return 0
	}

	if frame < introQuoteFadeFrames {
		return float32(frame) / float32(introQuoteFadeFrames)
	}

	fadeOutStart := introQuoteDurationFrames - introQuoteFadeFrames
	if frame >= fadeOutStart {
		return float32(introQuoteDurationFrames-frame) / float32(introQuoteFadeFrames)
	}

	return 1
}
