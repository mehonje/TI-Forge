package render

import(
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"ti_forge/ansi"
	"ti_forge/convert"
	"ti_forge/state"
	"unicode/utf8"

	"github.com/lithammer/fuzzysearch/fuzzy"
	"golang.org/x/term"
)

func Display_data(state state.State) state.State {
	max_line_num_len := len(strconv.Itoa(len(state.Program_data[state.Buffer_idx])))
	line_num_fmtstr := fmt.Sprintf("%%%dd ", max_line_num_len)
	
	_, height := get_term_size()
	height -= 6
	half_height := height/2
	var buffer_row int = state.Cursor_row
	var buffer_start int = max(0, buffer_row-half_height)
	var buffer_end int = min(buffer_row+half_height, len(state.Program_data[state.Buffer_idx]))

	var builder strings.Builder

	builder.WriteString(ansi.Clear)
	builder.WriteString(ansi.Reset_cursor)

	var indent_size string = ""
	for i := 0; i < state.Options["indent_size"]; i++ {
		indent_size = indent_size + " "
	}
	var indent_str string = ""

	var tracking_block bool = false
	var block_indent int = 0

	for i := buffer_start; i < buffer_end; i++ {
		var line_builder strings.Builder
		var indent_change int = 0

		for j, command := range state.Program_data[state.Buffer_idx][i] {
			tracking_block, block_indent, indent_str, indent_change = process_block_highlight(command, tracking_block, block_indent, i, j, state.Cursor_row, state.Cursor_col, indent_str, indent_change, state.Options["indent_size"])

			if is_highlighted(i, j, state) {
				line_builder.WriteString(ansi.Highlight)
			} else if state.Options["block_highlight"] == 1 && tracking_block && block_indent <= 0 {
				line_builder.WriteString(ansi.Highlight)
				if block_indent <= 0 && tracking_block {
					tracking_block = false
					block_indent = 0
				}
			} else {
				line_builder.WriteString(ansi.Reset_text)
			}

			line_builder.WriteString(command)
		}

		line_builder.WriteString(ansi.Reset_text)
		fmt.Fprintf(&builder, line_num_fmtstr, i + 1) // padded line number, starts at 1
		builder.WriteString(indent_str)
		builder.WriteString(line_builder.String()) // line
		line_builder.WriteString(ansi.Reset_text)
		builder.WriteByte('\n')

		for indent_change > 0 {
			indent_str = indent_str + indent_size
			indent_change--
		}
	}

	var screen_col int = max_line_num_len+2
	for i := 0; i < state.Cursor_col; i++ {
    screen_col += utf8.RuneCountInString(state.Program_data[state.Buffer_idx][state.Cursor_row][i])
	}

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
		var command_matches []string
		command_matches = fuzzy.FindFold(string(state.Text_buffer), convert.Commands) // find commands that match the text buffer,
		
		if len(command_matches) > 0 {
			var start int = min(state.Suggestion_idx, len(command_matches) - 1)
			var end int = min(5 + state.Suggestion_idx, len(command_matches))
			display_command_matches = command_matches[start:end]
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

	os.Stdout.WriteString(builder.String())
	

	return state
}

func process_block_highlight(command string, tracking_block bool, block_indent int, row int, col int, cursor_row int, cursor_col int, indent_str string, indent_change int, indent_size int) (bool, int, string, int) {
	if command == "For(" || command == "Repeat " || command == "While " || command == "If " {
		if row == cursor_row && col == cursor_col {
			tracking_block = true
			block_indent = 0
		}
		block_indent++

		indent_change++
	} else if command == "End" {
		block_indent--

		if len(indent_str) > 0 {
			indent_str = indent_str[:len(indent_str) - indent_size] 
		}
	} else if command == "Else" {
		if tracking_block {
			block_indent--
		} else {
			if row == cursor_row && col == cursor_col {
				tracking_block = true
				block_indent = 1
			}
		}

		if len(indent_str) > 0 {
			indent_str = indent_str[:len(indent_str) - indent_size] 
		}
		indent_change++

	}

	return tracking_block, block_indent, indent_str, indent_change
}

func is_highlighted(row int, col int, state state.State) bool {
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

func get_term_size() (int, int) {
	width, height, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		log.Fatal("Failed to get terminal dimensions: ", err)
	}
	return width, height
}
