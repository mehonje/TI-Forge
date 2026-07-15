package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"ti_forge/convert"
	"ti_forge/input"
	"ti_forge/state"
	"unicode/utf8"

	"golang.org/x/term"
	"github.com/lithammer/fuzzysearch/fuzzy"
)

func main() {
	enable_utf8()

	state := state.State{
		Quit: false,
		Cursor_row: 0,
		Cursor_col: 0,
		Mode: 0,
		Text_buffer: []rune{},
		Suggestion_idx: 0,
		Input: []byte{},
		Program_data: [][][]string{{{"\n"}}},
		File_names: []string{""},
		Buffer_idx: 0,
		Highlight_row: 0,
		Highlight_col: 0,
		Highlighting: false,
		Copy_buffer: [][]string{},
	}
	
	fd := int(os.Stdin.Fd())
	cooked_state, err := term.MakeRaw(fd)
	if err != nil {
		log.Fatal("Failed to enter raw mode: ", err)
	}
	defer term.Restore(fd, cooked_state)

	for {
		state = display_data(state)
		state = input.Get_input(state)
		state = input.Process_input(state)
		
		if state.Quit {
			fmt.Print("\033[27m\033[0m") // reset colouring
			return
		}
	}
}

func is_highlighted(row int, col int, state state.State) bool {
	if !state.Highlighting {
		if row == state.Cursor_row && col == state.Cursor_col {
			return len(state.Program_data[state.Buffer_idx][row][col]) > 1
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

func display_data(state state.State) state.State {
	max_line_num_len := len(strconv.Itoa(len(state.Program_data[state.Buffer_idx])))
	line_num_fmtstr := fmt.Sprintf("%%%dd ", max_line_num_len)
	
	_, height := get_term_size()
	height -= 6
	half_height := height/2
	var buffer_row int = state.Cursor_row
	var buffer_start int = max(0, buffer_row-half_height)
	var buffer_end int = min(buffer_row+half_height, len(state.Program_data[state.Buffer_idx]))

	var builder strings.Builder
	
	for i := buffer_start; i < buffer_end; i++ {
		var line_builder strings.Builder
		for j, command := range state.Program_data[state.Buffer_idx][i] {
			if is_highlighted(i, j, state) {
				line_builder.WriteString("\033[7m")
			} else {
				line_builder.WriteString("\033[0m")
			}

			line_builder.WriteString(command)
		}
		line_builder.WriteString("\033[0m")
		fmt.Fprintf(&builder, line_num_fmtstr, i + 1) // padded line number, starts at 1
		builder.WriteString(line_builder.String()) // line
		line_builder.WriteString("\033[0m")
	}

	var screen_row int = buffer_row-buffer_start
	var screen_col int = max_line_num_len+2
	for i := 0; i < state.Cursor_col; i++ {
    screen_col += utf8.RuneCountInString(state.Program_data[state.Buffer_idx][state.Cursor_row][i])
	}

	fmt.Print("\033[H\033[2J") // clear screen
	fmt.Print(builder.String()) // print program slice

	fmt.Print("\033[0m") // reset colouring
	
	fmt.Println(state.File_names[state.Buffer_idx])

	var mode_string string = ""
	var prepend_string string = ""
	switch state.Mode {
		case 0:
			mode_string = "NORMAL"
			fmt.Print("\033[2 q") // make cursor steady block
		case 1:
			mode_string = "INSERT"
			fmt.Print("\033[6 q") // make cursor steady bar
		case 2:
			mode_string = "COMMAND"
			prepend_string = ":"
		case 3:
			mode_string = "VISUAL"
	}
	fmt.Print(mode_string, "   ", prepend_string, string(state.Text_buffer), "\n")

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
				fmt.Print(command)
				if idx <= 3 {
					fmt.Print("\n")
				}
	}

	fmt.Printf("\033[%d;%dH", screen_row + 1, screen_col) // move cursor
	
	return state
}

func get_term_size() (int, int) {
	width, height, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		log.Fatal("Failed to get terminal dimensions: ", err)
	}
	return width, height
}

func enable_utf8() {
	cmd := exec.Command("cmd.exe", "/c", "chcp 65001 > nul")
	cmd.Stdout = os.Stdout
	cmd.Run()
}

