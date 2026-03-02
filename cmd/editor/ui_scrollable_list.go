package main

import "github.com/ebitenui/ebitenui/widget"

const (
	defaultScrollableListMinHeight = 120
	defaultScrollableListMaxHeight = 180
)

func configureScrollableList(list *widget.List, minHeight int) {
	if list == nil {
		return
	}
	if minHeight <= 0 {
		minHeight = defaultScrollableListMinHeight
	}
	w := list.GetWidget()
	w.MinHeight = minHeight
	w.LayoutData = widget.RowLayoutData{
		Position:  widget.RowLayoutPositionStart,
		MaxHeight: defaultScrollableListMaxHeight,
	}
}
