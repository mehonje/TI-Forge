package render

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"ti_forge/ansi"
	"ti_forge/helpers"
	"ti_forge/state"
)

var LINE_NAMES = [4]string{"　name", "　comment", "　locked?", "　archived?"}

type highlight struct {
	Active bool
	Size int
}

func Display_data(state *state.State) {
	program_data := state.Buffers[state.Buffer_idx]

	max_line_num_len := len(strconv.Itoa(len(program_data)))
	line_num_fmtstr := fmt.Sprintf("%%%dd ", max_line_num_len)
	
	_, height := helpers.Get_term_size()
	height -= 6

	var builder strings.Builder

	builder.WriteString(ansi.Clear)
	builder.WriteString(ansi.Reset_cursor)

	indent_block := ""

	for i := 0; i < state.Options["indent_size"]; i++ {
		indent_block = indent_block + " "
	}

	highlight := highlight{
		Active: false,
		Size: 0,
	}

	build_viewport(&builder, &program_data, state, &highlight, &line_num_fmtstr, &indent_block, height)

	build_bottom_bars(&builder, state)

	os.Stdout.WriteString(builder.String())
}

func build_viewport(builder *strings.Builder, program_data *[][]string, state *state.State, highlight *highlight, line_num_fmtstr *string, indent_block *string, height int) {
	for i := state.Viewport_row; i < state.Viewport_row + height; i++ {
		if i >= len(*program_data) {
			break
		}

		var line_builder strings.Builder

		for j, command := range (*program_data)[i] {
			*highlight = process_block_highlight(command, *highlight, i, j, state.Cursor_row, state.Cursor_col)

			if is_highlighted(i, j, state) {
				line_builder.WriteString(ansi.Highlight)
			} else if state.Options["block_highlight"] == 1 && highlight.Active && highlight.Size == 0 {
				line_builder.WriteString(ansi.Highlight)
				if highlight.Size == 0 && highlight.Active {
					highlight.Active = false
					highlight.Size = 0
				}
			} else {
				line_builder.WriteString(ansi.Reset_text)
			}

			line_builder.WriteString(command)
		}

		line_builder.WriteString(ansi.Reset_text)
		fmt.Fprintf(builder, *line_num_fmtstr, i + 1) // padded line number, starts at 1
		builder.WriteString(strings.Repeat(*indent_block, state.Indentation[state.Buffer_idx][i])) // indent
		builder.WriteString(line_builder.String()) // line
		line_builder.WriteString(ansi.Reset_text)

		if i <= 3 {
			builder.WriteString(LINE_NAMES[i])
		}

		builder.WriteByte('\n')
	}
}

func build_bottom_bars(builder *strings.Builder, state *state.State) {
	builder.WriteString(ansi.Reset_text)
	
	builder.WriteString(state.File_names[state.Buffer_idx])
	builder.WriteByte('\n')

	var mode_string string = ""
	var prepend_string string = ""
	switch state.Mode {
		case 0:
			mode_string = "NORMAL"
		case 1:
			mode_string = "INSERT"
		case 2:
			mode_string = "COMMAND"
			prepend_string = ":"
		case 3:
			mode_string = "VISUAL"
	}
	builder.WriteString(mode_string)
	builder.WriteString("   ")
	builder.WriteString(prepend_string)
	
	if len(state.Text_buffer) > 0 {
			builder.WriteString(string(state.Text_buffer))
	}

	if state.Mode == 1 || state.Mode == 2 {
		builder.WriteString(ansi.Highlight)
		builder.WriteString(" ")
		builder.WriteString(ansi.Reset_text)
	}

	builder.WriteByte('\n')

	var display_command_matches []string
	if state.Mode == 1 && len(state.Text_buffer) > 0 { // if in insert mode and text buffer has characters,
		if len(state.Command_matches) > 0 {
			var start int = min(state.Suggestion_idx, len(state.Command_matches) - 1)
			var end int = min(5 + state.Suggestion_idx, len(state.Command_matches))
			display_command_matches = state.Command_matches[start:end]
		}
	}

	for idx, command := range display_command_matches { // print the first 5 commands from the selected command
		if idx == 0 {
			builder.WriteString(ansi.Highlight)
		}

		builder.WriteString(command)

		if idx == 0 {
			builder.WriteString(ansi.Reset_text)
		}

		if idx <= 3 {
			builder.WriteByte('\n')
		}
	}
}

func process_block_highlight(command string, highlight highlight, row int, col int, cursor_row int, cursor_col int) highlight {
	if command == "For(" || command == "Repeat " || command == "While " || command == "If " {
		if row == cursor_row && col == cursor_col {
			highlight.Active = true
			highlight.Size = 0
		}
		highlight.Size++
	} else if command == "End" {
		highlight.Size--

	} else if command == "Else" {
		if highlight.Active {
			highlight.Size--
		} else {
			if row == cursor_row && col == cursor_col {
				highlight.Active = true
				highlight.Size = 1
			}
		}
	}

	return highlight
}

func is_highlighted(row int, col int, state *state.State) bool {
	if !state.Highlighting {
		if row == state.Cursor_row && col == state.Cursor_col {
			return true
		}
		return false
	}

	if row == state.Cursor_row && col == state.Cursor_col {
		return true
	}
	
	start_row, start_col := state.Cursor_row, state.Cursor_col
	end_row, end_col := state.Highlight_row, state.Highlight_col

	if start_row > end_row || (start_row == end_row && start_col > end_col) {
		start_row, end_row = end_row, start_row
		start_col, end_col = end_col, start_col
	}

	if row < start_row || (row == start_row && col < start_col) {
		return false
	}

	if row > end_row || (row == end_row && col > end_col) {
		return false
	}

	return true
}
