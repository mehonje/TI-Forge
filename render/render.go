package render

import(
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"ti_forge/ansi"
	"ti_forge/state"
	"ti_forge/tokens"
	"unicode/utf8"

	"github.com/lithammer/fuzzysearch/fuzzy"
	"golang.org/x/term"
)

type indent struct {
	Str string
	Change int
	Size int
	Block string
}

type highlight struct {
	Active bool
	Size int
}

func Display_data(state *state.State) {
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

	indent := indent{
		Str: "",
		Change: 0,
		Size: state.Options["indent_size"],
		Block: "",
	}

	for i := 0; i < indent.Size; i++ {
		indent.Block = indent.Block + " "
	}

	highlight := highlight{
		Active: false,
		Size: 0,
	}

	for i := buffer_start; i < buffer_end; i++ {
		var line_builder strings.Builder
		indent.Change = 0

		for j, command := range state.Program_data[state.Buffer_idx][i] {
			highlight, indent = process_block_highlight(command, highlight, indent, i, j, state.Cursor_row, state.Cursor_col)

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
		fmt.Fprintf(&builder, line_num_fmtstr, i + 1) // padded line number, starts at 1
		builder.WriteString(indent.Str)
		builder.WriteString(line_builder.String()) // line
		line_builder.WriteString(ansi.Reset_text)
		builder.WriteByte('\n')

		for indent.Change > 0 {
			indent.Str = indent.Str + indent.Block
			indent.Change--
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
		command_matches = fuzzy.FindFold(string(state.Text_buffer), tokens.Commands) // find commands that match the text buffer,
		
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
}

func process_block_highlight(command string, highlight highlight, indent indent, row int, col int, cursor_row int, cursor_col int) (highlight, indent) {
	if command == "For(" || command == "Repeat " || command == "While " || command == "If " {
		if row == cursor_row && col == cursor_col {
			highlight.Active = true
			highlight.Size = 0
		}
		highlight.Size++

		indent.Change++
	} else if command == "End" {
		highlight.Size--

		if len(indent.Str) > 0 {
			indent.Str = indent.Str[:len(indent.Str) - indent.Size] 
		}
	} else if command == "Else" {
		if highlight.Active {
			highlight.Size--
		} else {
			if row == cursor_row && col == cursor_col {
				highlight.Active = true
				highlight.Size = 1
			}
		}

		if len(indent.Str) > 0 {
			indent.Str = indent.Str[:len(indent.Str) - indent.Size] 
		}
		indent.Change++

	}

	return highlight, indent
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

func get_term_size() (int, int) {
	width, height, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		log.Fatal("Failed to get terminal dimensions: ", err)
	}
	return width, height
}
