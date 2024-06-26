/*
 * Copyright (c) 2024 Bjoern Beier.
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy of
 * this software and associated documentation files (the "Software"), to deal in
 * the Software without restriction, including without limitation the rights to
 * use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
 * the Software, and to permit persons to whom the Software is furnished to do so,
 * subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
 * FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
 * COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
 * IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
 * CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
 */

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
		for column := 0; column < number_of_columns && column < len(chunks); column++ {
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
