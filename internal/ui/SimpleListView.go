package ui

import (
	"fmt"
	"github.com/begris-net/qtoolbox/internal/util"
	"github.com/pterm/pterm"
	"math"
	"strings"
)

type ViewItem struct {
	Header   string
	Elements []ViewElement
}

type ViewElement struct {
	Name      string
	Installed bool
	Default   bool
}

func (e ViewElement) isInstalled() string {
	if e.Installed {
		return "*"
	}
	return " "
}

func (e ViewElement) isDefault() string {
	if e.Default {
		return ">"
	}
	return " "
}

const (
	min_row_size    = 15
	four_row_break  = min_row_size * 3
	minimum_columns = 3
	maximum_columns = 4
	line_width      = 80
)

var vertical_separator = strings.Repeat("=", line_width)

/*
(15 * 2) + 10 == 40
-- items <= 45 --> 3 columns && 15 rows

*/

func NewViewItem(header string, elements []ViewElement) *ViewItem {
	return &ViewItem{
		Header:   header,
		Elements: elements,
	}
}

func (v *ViewItem) generateRows() [][]string {
	items := len(v.Elements)
	var number_of_rows int
	var number_of_columns int
	if items > four_row_break {
		number_of_rows = int(math.Ceil(float64((items / maximum_columns))))
		number_of_columns = maximum_columns
	} else {
		number_of_rows = min_row_size
		number_of_columns = minimum_columns
	}
	chunks, _ := util.Chunks(number_of_rows, v.Elements)

	rows := make([][]string, number_of_rows)
	for row := 0; row < number_of_rows; row++ {
		currentRow := make([]string, number_of_columns)
		for column := 0; column < number_of_columns; column++ {
			if len(chunks[column]) > row {
				currentRow[column] = chunks[column][row].renderViewElement()
			}
		}
		rows[row] = currentRow
	}

	return rows
}

func (v *ViewItem) renderHeader() string {
	return fmt.Sprintf("%s\n%s\n%s\n", vertical_separator, v.Header, vertical_separator)
}

func (e ViewElement) renderViewElement() string {
	return fmt.Sprintf(" %s %s %-15s", e.isDefault(), e.isInstalled(), e.Name)
}

func (v *ViewItem) renderFooter() string {
	//
	//================================================================================
	//* - installed                                     Number of candidates:     56
	//> - currently in use
	//================================================================================

	footer := `` +
		vertical_separator + "\n" +
		`* - installed` + fmt.Sprintf("%58s %6d\n", "Number of candidates:", len(v.Elements)) +
		`> - currently in use` + "\n" +
		vertical_separator + "\n"

	return footer
}

func (v *ViewItem) Show() {
	pterm.Print(v.renderHeader())
	pterm.DefaultTable.WithHasHeader(false).WithLeftAlignment().WithSeparator("").WithData(v.generateRows()).Render()
	pterm.Print(v.renderFooter())
}
