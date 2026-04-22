// SPDX-License-Identifier: GPL-3.0-or-later
package ui

import (
	"fmt"
	"strings"
)

// PopupKind distinguishes the two popup modes.
type PopupKind int

const (
	PopupSelect PopupKind = iota
	PopupInput
)

// PopupParams carries all data needed to render either popup kind.
type PopupParams struct {
	Kind  PopupKind
	Title string

	// Select popup.
	Options   []string
	SelCursor int

	// Input popup.
	InputBuf string
	Masked   bool
	Prompt   string
}

// RenderPopup returns the popup box as a string (not overlaid on anything).
// Call PlaceOverlay to center it on the screen content.
func RenderPopup(params PopupParams, screenWidth int) string {
	switch params.Kind {
	case PopupSelect:
		return renderSelect(params, screenWidth)
	case PopupInput:
		return renderInput(params, screenWidth)
	}
	return ""
}

func renderSelect(p PopupParams, screenW int) string {
	pw := 44
	for _, o := range p.Options {
		if len(o)+10 > pw {
			pw = len(o) + 10
		}
	}
	if pw > screenW-4 {
		pw = screenW - 4
	}
	ph := len(p.Options) + 5
	inner := pw - 2

	var lines []string
	lines = append(lines, StyleBorder.Render("┌"+centerTitle(p.Title, inner)+"┐"))

	lines = append(lines, StyleBorder.Render("│")+strings.Repeat(" ", inner)+StyleBorder.Render("│"))

	for i, opt := range p.Options {
		sel := i == p.SelCursor
		var rowStr string
		marker := " "
		if sel {
			marker = "►"
		}
		content := fmt.Sprintf("%s %-*s", marker, inner-4, opt)
		if sel {
			rowStr = StyleBorder.Render("│") + StyleSelected.Render(content) + StyleBorder.Render("│")
		} else {
			rowStr = StyleBorder.Render("│") + StyleDefault.Render(content) + StyleBorder.Render("│")
		}
		lines = append(lines, rowStr)
	}

	lines = append(lines, StyleBorder.Render("│")+strings.Repeat(" ", inner)+StyleBorder.Render("│"))

	hint := "↑↓ select   Enter confirm   Esc cancel"
	hPad := (inner - len(hint)) / 2
	if hPad < 0 {
		hPad = 0
	}
	hintLine := StyleBorder.Render("│") +
		strings.Repeat(" ", hPad) +
		StyleYellow.Render(hint) +
		strings.Repeat(" ", inner-hPad-len(hint)) +
		StyleBorder.Render("│")
	lines = append(lines, hintLine)

	lines = append(lines, StyleBorder.Render("└"+strings.Repeat("─", inner)+"┘"))

	_ = ph
	return strings.Join(lines, "\n")
}

func renderInput(p PopupParams, screenW int) string {
	const ph = 8
	pw := 64
	if pw > screenW-4 {
		pw = screenW - 4
	}
	if pw < 24 {
		pw = 24
	}
	inner := pw - 2
	fieldW := inner - 4

	disp := p.InputBuf
	if p.Masked {
		disp = strings.Repeat("●", len(p.InputBuf))
	}
	if len(disp) > fieldW {
		disp = disp[len(disp)-fieldW:]
	}

	var lines []string
	lines = append(lines, StyleBorder.Render("┌"+centerTitle(p.Title, inner)+"┐"))
	lines = append(lines, StyleBorder.Render("│")+strings.Repeat(" ", inner)+StyleBorder.Render("│"))

	// Prompt line.
	promptPad := inner - len(p.Prompt)
	if promptPad < 0 {
		promptPad = 0
	}
	lines = append(lines, StyleBorder.Render("│")+" "+StyleYellow.Render(p.Prompt)+
		strings.Repeat(" ", promptPad-1)+StyleBorder.Render("│"))

	lines = append(lines, StyleBorder.Render("│")+strings.Repeat(" ", inner)+StyleBorder.Render("│"))

	// Input field box.
	lines = append(lines, StyleBorder.Render("│")+"  "+
		StyleBorder.Render("┌"+strings.Repeat("─", fieldW)+"┐")+"  "+StyleBorder.Render("│"))

	// Input content row.
	cursorVisible := (len(disp) < fieldW)
	cursor := ""
	if cursorVisible {
		cursor = StyleCyan.Bold(true).Render("▌")
	}
	fieldContent := fmt.Sprintf("%-*s", fieldW, disp) + cursor
	lines = append(lines, StyleBorder.Render("│")+"  "+
		StyleBorder.Render("│")+StyleDefault.Bold(true).Render(fieldContent[:fieldW])+StyleBorder.Render("│")+
		"  "+StyleBorder.Render("│"))

	lines = append(lines, StyleBorder.Render("│")+"  "+
		StyleBorder.Render("└"+strings.Repeat("─", fieldW)+"┘")+"  "+StyleBorder.Render("│"))

	// Hint.
	hint := "Enter confirm   Esc cancel   Backspace delete"
	hPad := (inner - len(hint)) / 2
	if hPad < 0 {
		hPad = 0
	}
	lines = append(lines, StyleBorder.Render("│")+
		strings.Repeat(" ", hPad)+StyleYellow.Render(hint)+
		strings.Repeat(" ", inner-hPad-len(hint))+StyleBorder.Render("│"))

	lines = append(lines, StyleBorder.Render("└"+strings.Repeat("─", inner)+"┘"))

	_ = ph
	return strings.Join(lines, "\n")
}
