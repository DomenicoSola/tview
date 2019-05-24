package tview

import (
	"log"

	"github.com/gdamore/tcell"
)

// TableCell represents one cell inside a Table.
type TableCell struct {
	// The text to be displayed in the table cell.
	Text string

	// The alignment of the cell text. One of AlignLeft (default), AlignCenter,
	// or AlignRight.
	Align int

	// The maximum width of the cell. This is used to give a column a maximum
	// width. Any cell text whose length exceeds this width is cut off. Set to
	// 0 if there is no maximum width.
	MaxWidth int

	// The color of the cell text.
	Color tcell.Color

	// Whether or not this cell may be selected.
	Selectable bool
}

// Table visualizes two-dimensional data consisting of rows and columns.
//
// Navigation
//
// If the table extends beyond the available space, it can be navigated with
// key bindings similar to Vim:
//
//   - h, left arrow: Move left by one column.
//   - l, right arrow: Move right by one column.
//   - j, down arrow: Move down by one row.
//   - k, up arrow: Move up by one row.
//   - g, home: Move to the top.
//   - G, end: Move to the bottom.
//   - Ctrl-F, page down: Move down by one page.
//   - Ctrl-B, page up: Move up by one page.
//
// When there is no selection, this affects the entire table (except for fixed
// rows and columns). When there is a selection, the user moves the selection.
// The class will attempt to always keep the selection in view.
type Table struct {
	*Box

	// Whether or not this table has borders around each cell.
	borders bool

	// The color of the borders or the separator.
	bordersColor tcell.Color

	// If there are no borders, the column separator.
	separator rune

	// The cells of the table. Rows first, then columns.
	cells [][]*TableCell

	// The rightmost column in the data set.
	lastColumn int

	// The number of fixed rows / columns.
	fixedRows, fixedColumns int

	// Whether or not rows or columns can be selected. If both are set to true,
	// cells can be selected.
	rowsSelectable, columnsSelectable bool

	// The currently selected row and column.
	selectedRow, selectedColumn int

	// The number of rows/columns by which the table is scrolled down/to the
	// right.
	rowOffset, columnOffset int

	// If set to true, the table's last row will always be visible.
	trackEnd bool

	// The number of visible rows the last time the table was drawn.
	visibleRows int

	// An optional function which gets called when the user presses Enter on a
	// selected cell. If entire rows selected, the column value is undefined.
	// Likewise for entire columns.
	selected func(row, column int)

	// An optional function which gets called when the user presses Escape, Tab,
	// or Backtab. Also when the user presses Enter if nothing is selectable.
	done func(key tcell.Key)
}

// NewTable returns a new table.
func NewTable() *Table {
	return &Table{
		Box:          NewBox(),
		bordersColor: tcell.ColorWhite,
		separator:    ' ',
		trackEnd:     true,
		lastColumn:   -1,
	}
}

// Clear removes all table data.
func (t *Table) Clear() *Table {
	t.cells = nil
	t.lastColumn = -1
	return t
}

// SetBorders sets whether or not each cell in the table is surrounded by a
// border.
func (t *Table) SetBorders(show bool) *Table {
	t.borders = show
	return t
}

// SetBordersColor sets the color of the cell borders.
func (t *Table) SetBordersColor(color tcell.Color) *Table {
	t.bordersColor = color
	return t
}

// SetSeparator sets the character used to fill the space between two
// neighboring cells. This is a space character ' ' per default but you may
// want to set it to GraphicsVertBar (or any other rune) if the column
// separation should be more visible. If cell borders are activated, this is
// ignored.
//
// Separators have the same color as borders.
func (t *Table) SetSeparator(separator rune) *Table {
	t.separator = separator
	return t
}

// SetFixed sets the number of fixed rows and columns which are always visible
// even when the rest of the cells are scrolled out of view. Rows are always the
// top-most ones. Columns are always the left-most ones.
func (t *Table) SetFixed(rows, columns int) *Table {
	t.fixedRows, t.fixedColumns = rows, columns
	return t
}

// SetSelectable sets the flags which determine what can be selected in a table.
// There are three selection modi:
//
//   - rows = false, columns = false: Nothing can be selected.
//   - rows = true, columns = false: Rows can be selected.
//   - rows = false, columns = true: Columns can be selected.
//   - rows = true, columns = true: Individual cells can be selected.
func (t *Table) SetSelectable(rows, columns bool) *Table {
	t.rowsSelectable, t.columnsSelectable = rows, columns
	return t
}

// SetSelected sets the selected cell. Depending on the selection settings
// specified via SetSelectable(), this may be an entire row or column, or even
// ignored completely.
func (t *Table) SetSelected(row, column int) *Table {
	t.selectedRow, t.selectedColumn = row, column
	return t
}

// SetOffset sets how many rows and columns should be skipped when drawing the
// table. This is useful for large tables that do not fit on the screen.
// Navigating a selection can change these values.
//
// Fixed rows and columns are never skipped.
func (t *Table) SetOffset(row, column int) *Table {
	t.rowOffset, t.columnOffset = row, column
	return t
}

// SetSelectedFunc sets a handler which is called whenever the user presses the
// Enter key on a selected cell/row/column. The handler receives the position of
// the selection and its cell contents. If entire rows are selected, the column
// index is undefined. Likewise for entire columns.
func (t *Table) SetSelectedFunc(handler func(row, column int)) *Table {
	t.selected = handler
	return t
}

// SetDoneFunc sets a handler which is called whenever the user presses the
// Escape, Tab, or Backtab key. If nothing is selected, it is also called when
// user presses the Enter key (because pressing Enter on a selection triggers
// the "selected" handler set via SetSelectedFunc()).
func (t *Table) SetDoneFunc(handler func(key tcell.Key)) *Table {
	t.done = handler
	return t
}

// SetCell sets the content of a cell the specified position. It is ok to
// directly instantiate a TableCell object. If the cell has contain, at least
// the Text and Color fields should be set.
//
// Note that setting cells in previously unknown rows and columns will
// automatically extend the internal table representation, e.g. starting with
// a row of 100,000 will immediately create 100,000 empty rows.
//
// To avoid unnecessary garbage collection, fill columns from left to right.
func (t *Table) SetCell(row, column int, cell *TableCell) *Table {
	if row >= len(t.cells) {
		t.cells = append(t.cells, make([][]*TableCell, row-len(t.cells)+1)...)
	}
	rowLen := len(t.cells[row])
	if column >= rowLen {
		t.cells[row] = append(t.cells[row], make([]*TableCell, column-rowLen+1)...)
		for c := rowLen; c < column; c++ {
			t.cells[row][c] = &TableCell{}
		}
	}
	t.cells[row][column] = cell
	if column > t.lastColumn {
		t.lastColumn = column
	}
	return t
}

// GetCell returns the contents of the cell at the specified position. A valid
// TableCell object is always returns but it will be uninitialized if the cell
// was not previously set.
func (t *Table) GetCell(row, column int) *TableCell {
	if row >= len(t.cells) || column >= len(t.cells[row]) {
		return &TableCell{}
	}
	return t.cells[row][column]
}

// Draw draws this primitive onto the screen.
func (t *Table) Draw(screen tcell.Screen) {
	t.Box.Draw(screen)

	// What's our available screen space?
	x, y, width, height := t.GetInnerRect()
	if t.borders {
		t.visibleRows = height / 2
	} else {
		t.visibleRows = height
	}

	// Return the cell at the specified position (nil if it doesn't exist).
	getCell := func(row, column int) *TableCell {
		if row >= len(t.cells) || column >= len(t.cells[row]) {
			return nil
		}
		return t.cells[row][column]
	}

	// Clamp row offsets.
	log.Print(t.rowOffset, t.selectedRow, height)
	if t.rowsSelectable {
		if t.selectedRow >= t.fixedRows && t.selectedRow < t.fixedRows+t.rowOffset {
			t.rowOffset = t.selectedRow - t.fixedRows
			t.trackEnd = false
		}
		if t.borders {
			if 2*(t.selectedRow+1-t.rowOffset) >= height {
				t.rowOffset = t.selectedRow + 1 - height/2
				t.trackEnd = false
			}
		} else {
			if t.selectedRow+1-t.rowOffset >= height {
				t.rowOffset = t.selectedRow + 1 - height
				t.trackEnd = false
			}
		}
	}
	if t.borders {
		if 2*(len(t.cells)-t.rowOffset) < height {
			t.trackEnd = true
		}
	} else {
		if len(t.cells)-t.rowOffset < height {
			t.trackEnd = true
		}
	}
	if t.trackEnd {
		if t.borders {
			t.rowOffset = len(t.cells) - height/2
		} else {
			t.rowOffset = len(t.cells) - height
		}
	}
	if t.rowOffset < 0 {
		t.rowOffset = 0
	}

	// Clamp column offset. (Only left side here. The right side is more
	// difficult and we'll do it below.)
	if t.columnsSelectable && t.selectedColumn >= t.fixedColumns && t.selectedColumn < t.fixedColumns+t.columnOffset {
		t.columnOffset = t.selectedColumn - t.fixedColumns
	}
	if t.columnOffset < 0 {
		t.columnOffset = 0
	}
	if t.selectedColumn < 0 {
		t.selectedColumn = 0
	}

	// Determine the indices and widths of the columns which fit on the screen.
	var (
		columns, rows, widths   []int
		tableHeight, tableWidth int
	)
	rowStep := 1
	if t.borders {
		rowStep = 2    // With borders, every table row takes two screen rows.
		tableWidth = 1 // We start at the second character because of the left table border.
	}
	indexRow := func(row int) bool { // Determine if this row is visible, store its index.
		if tableHeight >= height {
			return false
		}
		rows = append(rows, row)
		tableHeight += rowStep
		return true
	}
	for row := 0; row < t.fixedRows && row < len(t.cells); row++ { // Do the fixed rows first.
		if !indexRow(row) {
			break
		}
	}
	for row := t.fixedRows + t.rowOffset; row < len(t.cells); row++ { // Then the remaining rows.
		if !indexRow(row) {
			break
		}
	}
	var skipped, lastTableWidth int
ColumnLoop:
	for column := 0; column <= t.lastColumn; column++ {
		// If we've moved beyond the right border, we stop or skip a column.
		for tableWidth-1 >= width { // -1 because we include one extra column if the separator falls on the right end of the box.
			// We've moved beyond the available space.
			if column < t.fixedColumns {
				break ColumnLoop // We're in the fixed area. We're done.
			}
			if !t.columnsSelectable && skipped >= t.columnOffset {
				break ColumnLoop // There is no selection and we've already reached the offset.
			}
			if t.columnsSelectable && t.selectedColumn-skipped == t.fixedColumns {
				break ColumnLoop // The selected column reached the leftmost point before disappearing.
			}
			if t.columnsSelectable && skipped >= t.columnOffset &&
				(t.selectedColumn < column && lastTableWidth < width-1 || t.selectedColumn < column-1) {
				break ColumnLoop // We've skipped as many as requested and the selection is visible.
			}
			if len(columns) <= t.fixedColumns {
				break // Nothing to skip.
			}

			// We need to skip a column.
			skipped++
			lastTableWidth -= widths[t.fixedColumns] + 1
			tableWidth -= widths[t.fixedColumns] + 1
			columns = append(columns[:t.fixedColumns], columns[t.fixedColumns+1:]...)
			widths = append(widths[:t.fixedColumns], widths[t.fixedColumns+1:]...)
		}

		// What's this column's width?
		maxWidth := -1
		for _, row := range rows {
			if cell := getCell(row, column); cell != nil {
				cellWidth := len(cell.Text)
				if cell.MaxWidth > 0 && cell.MaxWidth < cellWidth {
					cellWidth = cell.MaxWidth
				}
				if cellWidth > maxWidth {
					maxWidth = cellWidth
				}
			}
		}
		if maxWidth < 0 {
			break // No more cells found in this column.
		}

		// Store new column info at the end.
		columns = append(columns, column)
		widths = append(widths, maxWidth)
		lastTableWidth = tableWidth
		tableWidth += maxWidth + 1
	}
	t.columnOffset = skipped

	// Helper function which draws border runes.
	borderStyle := tcell.StyleDefault.Background(t.backgroundColor).Foreground(t.bordersColor)
	selectedBorderStyle := tcell.StyleDefault.Background(t.bordersColor).Foreground(t.backgroundColor)
	drawBorder := func(colX, rowY int, ch rune, selected bool) {
		style := borderStyle
		if selected {
			style = selectedBorderStyle
		}
		screen.SetContent(x+colX, y+rowY, ch, nil, style)
	}

	// Draw the cells (and borders).
	var columnX int
	if !t.borders {
		columnX--
	}
	for columnIndex, column := range columns {
		columnWidth := widths[columnIndex]
		columnSelected := t.columnsSelectable && !t.rowsSelectable && column == t.selectedColumn
		for rowY, row := range rows {
			// Is this row/column/cell selected?
			rowSelected := t.rowsSelectable && !t.columnsSelectable && row == t.selectedRow
			cellSelected := columnSelected || rowSelected || t.rowsSelectable && t.columnsSelectable && column == t.selectedColumn && row == t.selectedRow

			if t.borders {
				// Draw borders.
				rowY *= 2
				for pos := 0; pos < columnWidth && columnX+1+pos < width; pos++ {
					drawBorder(columnX+pos+1, rowY, GraphicsHoriBar, columnSelected)
				}
				ch := GraphicsCross
				if columnIndex == 0 {
					if rowY == 0 {
						ch = GraphicsTopLeftCorner
					} else {
						ch = GraphicsLeftT
					}
				} else if rowY == 0 {
					ch = GraphicsTopT
				}
				drawBorder(columnX, rowY, ch, false)
				rowY++
				if rowY >= height {
					break // No space for the text anymore.
				}
				drawBorder(columnX, rowY, GraphicsVertBar, rowSelected)
			} else if columnIndex > 0 {
				// Draw separator.
				drawBorder(columnX, rowY, t.separator, rowSelected)
			}

			// Get the cell.
			cell := getCell(row, column)

			// Determine colors.
			bgColor := t.backgroundColor
			textColor := cell.Color
			if cellSelected {
				bgColor = cell.Color
				textColor = t.backgroundColor
			}

			// Draw cell background.
			bgStyle := tcell.StyleDefault.Background(bgColor)
			for pos := 0; pos < columnWidth && columnX+1+pos < width; pos++ {
				screen.SetContent(x+columnX+1+pos, y+rowY, ' ', nil, bgStyle)
			}

			// Draw text.
			w := columnWidth
			if columnX+1+w >= width {
				w = width - columnX - 1
			}
			text := []rune(cell.Text)
			if w < len(text) && w > 0 {
				text = append(text[:w-1], GraphicsEllipsis)
			}
			Print(screen, string(text), x+columnX+1, y+rowY, w, cell.Align, textColor)
		}

		// Draw bottom border.
		if rowY := 2 * len(rows); t.borders && rowY < height {
			for pos := 0; pos < columnWidth && columnX+1+pos < width; pos++ {
				drawBorder(columnX+pos+1, rowY, GraphicsHoriBar, columnSelected)
			}
			ch := GraphicsBottomT
			if columnIndex == 0 {
				ch = GraphicsBottomLeftCorner
			}
			drawBorder(columnX, rowY, ch, false)
		}

		columnX += columnWidth + 1
	}

	// Draw right border.
	if t.borders && columnX < width {
		for rowY, row := range rows {
			rowSelected := t.rowsSelectable && !t.columnsSelectable && row == t.selectedRow
			rowY *= 2
			if rowY+1 < height {
				drawBorder(columnX, rowY+1, GraphicsVertBar, rowSelected)
			}
			ch := GraphicsRightT
			if rowY == 0 {
				ch = GraphicsTopRightCorner
			}
			drawBorder(columnX, rowY, ch, false)
		}
		if rowY := 2 * len(rows); rowY < height {
			drawBorder(columnX, rowY, GraphicsBottomRightCorner, false)
		}
	}
}

// InputHandler returns the handler for this primitive.
func (t *Table) InputHandler() func(event *tcell.EventKey, setFocus func(p Primitive)) {
	return func(event *tcell.EventKey, setFocus func(p Primitive)) {
		key := event.Key()

		if (!t.rowsSelectable && !t.columnsSelectable && key == tcell.KeyEnter) ||
			key == tcell.KeyEscape ||
			key == tcell.KeyTab ||
			key == tcell.KeyBacktab {
			if t.done != nil {
				t.done(key)
			}
			return
		}

		// Movement functions.
		var (
			home = func() {
				if t.rowsSelectable {
					t.selectedRow = 0
					t.selectedColumn = 0
				} else {
					t.trackEnd = false
					t.rowOffset = 0
					t.columnOffset = 0
				}
			}

			end = func() {
				if t.rowsSelectable {
					t.selectedRow = len(t.cells) - 1
					t.selectedColumn = t.lastColumn
				} else {
					t.trackEnd = true
					t.columnOffset = 0
				}
			}

			down = func() {
				if t.rowsSelectable {
					t.selectedRow++
					if t.selectedRow >= len(t.cells) {
						t.selectedRow = len(t.cells) - 1
					}
				} else {
					t.rowOffset++
				}
			}

			up = func() {
				if t.rowsSelectable {
					t.selectedRow--
					if t.selectedRow < 0 {
						t.selectedRow = 0
					}
				} else {
					t.trackEnd = false
					t.rowOffset--
				}
			}

			left = func() {
				if t.columnsSelectable {
					t.selectedColumn--
					if t.selectedColumn < 0 {
						t.selectedColumn = 0
					}
				} else {
					t.columnOffset--
				}
			}

			right = func() {
				if t.columnsSelectable {
					t.selectedColumn++
					if t.selectedColumn > t.lastColumn {
						t.selectedColumn = t.lastColumn
					}
				} else {
					t.columnOffset++
				}
			}

			pageDown = func() {
				if t.rowsSelectable {
					t.selectedRow += t.visibleRows
					if t.selectedRow >= len(t.cells) {
						t.selectedRow = len(t.cells) - 1
					}
				} else {
					t.rowOffset += t.visibleRows
				}
			}

			pageUp = func() {
				if t.rowsSelectable {
					t.selectedRow -= t.visibleRows
					if t.selectedRow < 0 {
						t.selectedRow = 0
					}
				} else {
					t.trackEnd = false
					t.rowOffset -= t.visibleRows
				}
			}
		)

		switch key {
		case tcell.KeyRune:
			switch event.Rune() {
			case 'g':
				home()
			case 'G':
				end()
			case 'j':
				down()
			case 'k':
				up()
			case 'h':
				left()
			case 'l':
				right()
			}
		case tcell.KeyHome:
			home()
		case tcell.KeyEnd:
			end()
		case tcell.KeyUp:
			up()
		case tcell.KeyDown:
			down()
		case tcell.KeyLeft:
			left()
		case tcell.KeyRight:
			right()
		case tcell.KeyPgDn, tcell.KeyCtrlF:
			pageDown()
		case tcell.KeyPgUp, tcell.KeyCtrlB:
			pageUp()
		case tcell.KeyEnter:
			if (t.rowsSelectable || t.columnsSelectable) && t.selected != nil {
				t.selected(t.selectedRow, t.selectedColumn)
			}
		}
	}
}
