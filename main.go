package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"slices"
	"strconv"
	"strings"
	"ti_forge/convert"
	"unicode/utf8"

	"github.com/lithammer/fuzzysearch/fuzzy"
	"golang.org/x/term"
)

type State struct {
	Quit bool
	Cursor_row int
	Cursor_col int
	Mode int
	Text_buffer []rune
	Suggestion_idx int
	Input []byte
	Program_data [][]string
}

func main() {
	enable_utf8()

	state := State{
		Quit: false,
		Cursor_row: 0,
		Cursor_col: 0,
		Mode: 0,
		Text_buffer: []rune{},
		Suggestion_idx: 0,
		Input: []byte{},
		Program_data: [][]string{},
	}
	
	fd := int(os.Stdin.Fd())
	cooked_state, err := term.MakeRaw(fd)
	if err != nil {
		log.Fatal("Failed to enter raw mode: ", err)
	}
	defer term.Restore(fd, cooked_state)

	var program_data []byte = []byte{}
	var program_metadata [4]string = [4]string{}
	program_data, program_metadata = convert.Read8xp("C:/Users/Jack/Onedrive/Desktop/go_work/ti_forge/TEST.8xp")

	program_data_commands := convert.Data_to_strings(program_data, program_metadata)
	state.Program_data = program_data_commands
	for {
		state = display_data(state)
		state = get_input(state)
		state = process_input(state)
		
		if state.Quit {
			return
		}
	}
}

func display_data(state State) State {
	if state.Cursor_row < 0 {
		state.Cursor_row = 0
	}
	if state.Cursor_row >= len(state.Program_data) {
		state.Cursor_row = len(state.Program_data) - 1
	}
	if state.Cursor_col < 0 {
		state.Cursor_col = 0
	}
	var line_length int = len(state.Program_data[state.Cursor_row])
	if state.Cursor_col >= line_length {
		state.Cursor_col = line_length - 1
	}

	max_line_num_len := len(strconv.Itoa(len(state.Program_data)))
	line_num_fmtstr := fmt.Sprintf("%%%ds %%s", max_line_num_len)
	
	var command_matches []string
	var display_command_matches []string
	if state.Mode == 1 && len(state.Text_buffer) > 0 { // if in insert mode and text buffer has characters,
		command_matches = fuzzy.FindFold(string(state.Text_buffer), convert.Commands) // find commands that match the text buffer,
		
		if state.Suggestion_idx >= len(command_matches) {
			state.Suggestion_idx = max(len(command_matches) - 1, 0)
		} else if state.Suggestion_idx < 0 {
			state.Suggestion_idx = 0
		}

		if len(command_matches) > 0 {
			var start int = min(state.Suggestion_idx, len(command_matches) - 1)
			var end int = min(5 + state.Suggestion_idx, len(command_matches))
			display_command_matches = command_matches[start:end]

			if slices.Equal(state.Input, []byte{13}) { // [enter]
				state.Program_data[state.Cursor_row] = slices.Insert(state.Program_data[state.Cursor_row], state.Cursor_col, command_matches[start])
				state.Input = []byte{}
				state.Suggestion_idx = 0
				state.Text_buffer = []rune{}
			}
		}
	}

	_, height := get_term_size()
	height -= 6
	half_height := height/2
	var buffer_row int = state.Cursor_row
	var buffer_start int = max(0, buffer_row-half_height)
	var buffer_end int = min(buffer_row+half_height, len(state.Program_data))

	var builder strings.Builder

	for i := buffer_start; i < buffer_end; i++ {
		var line_builder strings.Builder
		for _, command := range state.Program_data[i] {
			line_builder.WriteString(command)
		}
		fmt.Fprintf(&builder, line_num_fmtstr, strconv.Itoa(i+1),line_builder.String())
	}

	var screen_row int = buffer_row-buffer_start
	var screen_col int = max_line_num_len+2
	for i := 0; i < state.Cursor_col; i++ {
    screen_col += utf8.RuneCountInString(state.Program_data[state.Cursor_row][i])
	}

	fmt.Print("\033[H\033[2J") // clear screen
	fmt.Print(builder.String()) // print program slice

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
	}
	fmt.Print(mode_string, "   ", prepend_string, string(state.Text_buffer), "\n")
	
	for _, command := range display_command_matches { // print the first 5 commands from the selected command
				fmt.Println(command)
	}

	fmt.Printf("\033[%d;%dH", screen_row + 1, screen_col) // move cursor
	
	return state
}

func get_input(state State) State {
    buf := make([]byte, 10)
    n, err := os.Stdin.Read(buf)
    if err != nil || n == 0 {
        state.Input = []byte{}
				return state
    }
		state.Input = buf[:n]
		return state
}

func process_input(state State) (State) {
	switch state.Mode {
		case 0: state = process_normal_input(state)
		case 1: state = process_insert_input(state)
		case 2: state = process_command_input(state)
	}

	switch {
		case slices.Equal(state.Input, []byte{27}): // [esc]
			state.Mode = 0
			state.Text_buffer = []rune{}
	}

	return state
}

func process_normal_input(state State) (State) {
	switch {
		case slices.Equal(state.Input, []byte{104}): // [h], left
			state.Cursor_col--
		case slices.Equal(state.Input, []byte{108}): // [l], right
			state.Cursor_col++
		case slices.Equal(state.Input, []byte{106}): // [j], down
			state.Cursor_row++
		case slices.Equal(state.Input, []byte{107}): // [k], up
			state.Cursor_row--
		case slices.Equal(state.Input, []byte{105}): // [i], enter insert mode
			state.Mode = 1
		case slices.Equal(state.Input, []byte{58}): // [:], enter command mode
			state.Mode = 2
		case slices.Equal(state.Input, []byte{120}): // [x], delete command at cursor
			state.Program_data[state.Cursor_row] = slices.Delete(state.Program_data[state.Cursor_row], state.Cursor_col, state.Cursor_col + 1)
			if slices.Equal(state.Program_data[state.Cursor_row], []string{}) {
				state.Program_data = slices.Delete(state.Program_data, state.Cursor_row, state.Cursor_row + 1)
				if len(state.Program_data) == 0 {
					state.Program_data = [][]string{[]string{""}}
				}
			}
	}

	return state
}

func process_insert_input(state State) (State) {
	switch {
		case slices.Equal(state.Input, []byte{127}): // [backspace]
			if len(state.Text_buffer) > 0 {
				state.Text_buffer = state.Text_buffer[:len(state.Text_buffer) - 1]
			}
		case slices.Equal(state.Input, []byte{9}): // [tab], suggestion down
			state.Suggestion_idx++
		case slices.Equal(state.Input, []byte{27, 91, 90}): // [shift][tab], suggestion up
			state.Suggestion_idx--
		case slices.Equal(state.Input, []byte{13}): // [enter]
		default:
			if len(state.Input) == 1 {
				state.Text_buffer = append(state.Text_buffer, rune(state.Input[0]))
			}
	}

	return state
}

func process_command_input(state State) (State) {
	switch {
		case slices.Equal(state.Input, []byte{127}): // [backspace]
			if len(state.Text_buffer) > 0 {
				state.Text_buffer = state.Text_buffer[:len(state.Text_buffer) - 1]
			}
		case slices.Equal(state.Input, []byte{13}): // [enter]
			var split_command []string = strings.Split(string(state.Text_buffer), " ")
			switch split_command[0] {
				case "w": // write, second element is file path
					err := convert.Txt_to_eightxp(split_command[1], state.Program_data)
					state.Mode = 0
					state.Text_buffer = []rune{}
					if err != nil {
						state.Text_buffer = []rune(err.Error())
					}
				case "q": // quit
					state.Quit = true
			}
		default: 
			if len(state.Input) == 1 {
				state.Text_buffer = append(state.Text_buffer, rune(state.Input[0]))
			}
	}

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

