package editorui

import (
	"image"
	"image/color"
	"testing"

	"github.com/ebitenui/ebitenui"
	euiimage "github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	editorcomponents "github.com/milk9111/sidescroller/cmd/editor/ui/components"
)

func TestPointerOverUIIgnoresRootButBlocksPanels(t *testing.T) {
	root := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)
	root.GetWidget().Rect = image.Rect(0, 0, 1200, 800)

	leftPanel := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(dummyNineSlice()),
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)
	leftPanel.GetWidget().Rect = image.Rect(0, 56, 280, 800)

	toolbar := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(dummyNineSlice()),
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)
	toolbar.GetWidget().Rect = image.Rect(0, 0, 1200, 56)

	root.AddChild(leftPanel, toolbar)

	ui := &EditorUI{UI: &ebitenui.UI{Container: root}}

	if ui.PointerOverUI(400, 300) {
		t.Fatalf("expected canvas area to not count as UI hover")
	}
	if !ui.PointerOverUI(120, 120) {
		t.Fatalf("expected panel area to count as UI hover")
	}
	if !ui.PointerOverUI(500, 20) {
		t.Fatalf("expected toolbar area to count as UI hover")
	}
	if ui.PointerOverUI(-10, -10) {
		t.Fatalf("expected points outside the UI to be ignored")
	}
}

func TestLayoutMetricsUsesActualWidgetRects(t *testing.T) {
	ui := &EditorUI{
		Toolbar:    &editorcomponents.Toolbar{Root: widget.NewContainer(widget.ContainerOpts.Layout(widget.NewAnchorLayout()))},
		InfoPanel:  &editorcomponents.InfoPanel{Root: widget.NewContainer(widget.ContainerOpts.BackgroundImage(dummyNineSlice()), widget.ContainerOpts.Layout(widget.NewAnchorLayout()))},
		AssetPanel: &editorcomponents.AssetPanel{Root: widget.NewContainer(widget.ContainerOpts.BackgroundImage(dummyNineSlice()), widget.ContainerOpts.Layout(widget.NewAnchorLayout()))},
	}
	ui.Toolbar.Root.GetWidget().Rect = image.Rect(0, 0, 1280, 64)
	ui.InfoPanel.Root.GetWidget().Rect = image.Rect(0, 56, 340, 720)
	ui.AssetPanel.Root.GetWidget().Rect = image.Rect(930, 56, 1280, 720)

	metrics := ui.LayoutMetrics(1280, 720)
	if metrics.LeftInset != 340 {
		t.Fatalf("expected left inset 340, got %v", metrics.LeftInset)
	}
	if metrics.RightInset != 350 {
		t.Fatalf("expected right inset 350, got %v", metrics.RightInset)
	}
	if metrics.TopInset != 64 {
		t.Fatalf("expected top inset 64, got %v", metrics.TopInset)
	}
}

func TestFocusedInputReturnsFocusedEditorField(t *testing.T) {
	fileInput := widget.NewTextInput()
	renameInput := widget.NewTextInput()
	renameInput.Focus(true)

	ui := &EditorUI{
		InfoPanel: &editorcomponents.InfoPanel{
			FileInput:  fileInput,
			LayerPanel: &editorcomponents.LayerPanel{RenameInput: renameInput},
		},
	}

	if got := ui.FocusedInput(); got != renameInput {
		t.Fatalf("expected focused input %p, got %p", renameInput, got)
	}
	if !ui.AnyInputFocused() {
		t.Fatalf("expected AnyInputFocused to report true")
	}
}

func dummyNineSlice() *euiimage.NineSlice {
	return euiimage.NewNineSliceColor(color.NRGBA{R: 20, G: 20, B: 20, A: 255})
}
