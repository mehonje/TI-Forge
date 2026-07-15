package main

import (
	"fmt"
	"log"
	"math/rand/v2"
	"os"
	"os/exec"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"ti_forge/convert"
	"unicode/utf8"

	"github.com/lithammer/fuzzysearch/fuzzy"
	"golang.org/x/term"
	"github.com/pkg/browser"
)


type State struct {
	Quit bool
	Cursor_row int
	Cursor_col int
	Mode int
	Text_buffer []rune
	Suggestion_idx int
	Input []byte
	Program_data [][][]string
	File_names []string
	Buffer_idx int
	Highlight_row int
	Highlight_col int
	Highlighting bool
	Copy_buffer [][]string
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
		state = get_input(state)
		state = process_input(state)
		
		if state.Quit {
			fmt.Print("\033[27m\033[0m") // reset colouring
			return
		}
	}
}

func is_highlighted(row int, col int, state State) bool {
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

func display_data(state State) State {
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
		
		if state.Suggestion_idx >= len(command_matches) {
			state.Suggestion_idx = max(len(command_matches) - 1, 0)
		} else if state.Suggestion_idx < 0 {
			state.Suggestion_idx = 0
		}

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
		case 3: state = process_visual_input(state)
	}

	if state.Cursor_row < 0 {
		state.Cursor_row = 0
	} else if state.Cursor_row >= len(state.Program_data[state.Buffer_idx]) {
		state.Cursor_row = len(state.Program_data[state.Buffer_idx]) - 1
	}

	var line_length int = len(state.Program_data[state.Buffer_idx][state.Cursor_row])
	if state.Cursor_col < 0 {
		state.Cursor_col = 0
	} else if state.Cursor_col >= line_length {
		state.Cursor_col = line_length - 1
	}

	switch {
		case slices.Equal(state.Input, []byte{27}): // [esc]
			state.Mode = 0
			state.Text_buffer = []rune{}
			state.Highlighting = false
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
			state.Program_data[state.Buffer_idx][state.Cursor_row] = slices.Delete(state.Program_data[state.Buffer_idx][state.Cursor_row], state.Cursor_col, state.Cursor_col + 1)
			if slices.Equal(state.Program_data[state.Buffer_idx][state.Cursor_row], []string{}) {
				state.Program_data[state.Buffer_idx] = slices.Delete(state.Program_data[state.Buffer_idx], state.Cursor_row, state.Cursor_row + 1)
				if len(state.Program_data) == 0 {
					state.Program_data = [][][]string{{{""}}}
				}
			}
		case slices.Equal(state.Input, []byte{118}): // [v], enter visual mode
			state.Mode = 3
			state.Highlighting = true
			state.Highlight_row = state.Cursor_row
			state.Highlight_col = state.Cursor_col
		case slices.Equal(state.Input, []byte{112}): // [p], put
			if len(state.Copy_buffer) == 1 {
				slices.Reverse(state.Copy_buffer[0])
				for _, command := range state.Copy_buffer[0] {
					if command == "\n" {
						continue
					}
					state.Program_data[state.Buffer_idx][state.Cursor_row] = slices.Insert(state.Program_data[state.Buffer_idx][state.Cursor_row], state.Cursor_col, command)
				}
			} else {
				slices.Reverse(state.Copy_buffer)
				for _, line := range state.Copy_buffer {
					state.Program_data[state.Buffer_idx] = slices.Insert(state.Program_data[state.Buffer_idx], state.Cursor_row, line)
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
			if len(state.Text_buffer) > 0 { // if text buffer has characters
				var command_matches []string
				command_matches = fuzzy.FindFold(string(state.Text_buffer), convert.Commands) // find commands that match the text buffer,
		
				if state.Suggestion_idx >= len(command_matches) {
					state.Suggestion_idx = max(len(command_matches) - 1, 0)
				} else if state.Suggestion_idx < 0 {
					state.Suggestion_idx = 0
				}

				if len(command_matches) > 1 {
					var idx int = min(state.Suggestion_idx, len(command_matches) - 1)
					state.Program_data[state.Buffer_idx][state.Cursor_row] = slices.Insert(state.Program_data[state.Buffer_idx][state.Cursor_row], state.Cursor_col, command_matches[idx])

					state.Input = []byte{}
					state.Suggestion_idx = 0
					state.Text_buffer = []rune{}
				}
			}
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

			state.Mode = 0
			state.Text_buffer = []rune{}

			switch split_command[0] {
				case "w": // write, second element is file path (if it exists)
					if len(split_command) > 1 {
						err := convert.Txt_to_eightxp(split_command[1], state.Program_data[state.Buffer_idx])
						if err != nil {
							state.Text_buffer = []rune(err.Error())
						}
					} else {
						err := convert.Txt_to_eightxp(state.File_names[state.Buffer_idx], state.Program_data[state.Buffer_idx])
						if err != nil {
							state.Text_buffer = []rune(err.Error())
						}
					}
				case "q": // quit
					state.Quit = true
				case "e": // edit, second element is file path
					var program_data []byte = []byte{}
					var program_metadata [4]string = [4]string{}
					program_data, program_metadata, err := convert.Read8xp(split_command[1])

					if err != nil {
						state.Text_buffer = []rune(err.Error())
					} else {
						program_data_commands := convert.Data_to_strings(program_data, program_metadata)
						if reflect.DeepEqual(state.Program_data, [][][]string{{{"\n"}}}) {
							state.Program_data = [][][]string{program_data_commands}
							state.File_names = []string{split_command[1]}
						} else {
							state.Program_data = append(state.Program_data, program_data_commands)
							state.File_names = append(state.File_names, split_command[1])
							state.Buffer_idx++
						}
					}
				case "bn": // next buffer
					if state.Buffer_idx < len(state.Program_data) - 1 {
						state.Buffer_idx++
					} else {
						state.Text_buffer = []rune("Reached last buffer")
					}
				case "bp": // previous buffer
					if state.Buffer_idx > 0 {
						state.Buffer_idx--
						state.Text_buffer = []rune("Reached first buffer")
					}
				case "help": // help
					num := rand.IntN(100)
					if num == 0 {
						err := browser.OpenURL("https://www.youtube.com/watch?v=dQw4w9WgXcQ")
						if err != nil {
							state.Text_buffer = []rune("Failed to open browser: " + err.Error())
						}
					}

					err := browser.OpenURL("https://github.com/mehonje/TI-Forge")
					if err != nil {
						state.Text_buffer = []rune("Failed to open browser: " + err.Error())
					} else {
						state.Text_buffer = []rune("Check your web browser")
					}
				default:
					state.Text_buffer = []rune("Unknown command \"" + split_command[0] + "\"")
			}
		default: 
			if len(state.Input) == 1 {
				state.Text_buffer = append(state.Text_buffer, rune(state.Input[0]))
			}
	}

	return state
}

func process_visual_input(state State) (State) {
	switch {
		case slices.Equal(state.Input, []byte{104}): // [h], left
			state.Cursor_col--
		case slices.Equal(state.Input, []byte{108}): // [l], right
			state.Cursor_col++
		case slices.Equal(state.Input, []byte{106}): // [j], down
			state.Cursor_row++
		case slices.Equal(state.Input, []byte{107}): // [k], up
			state.Cursor_row--
		case slices.Equal(state.Input, []byte{121}): // [y], yank
			start_row, start_col := state.Cursor_row, state.Cursor_col
			end_row, end_col := state.Highlight_row, state.Highlight_col
	
			if start_row > end_row || (start_row == end_row && start_col > end_col) {
				start_row, end_row = end_row, start_row
				start_col, end_col = end_col, start_col
			}

			state.Copy_buffer = [][]string{}

			if start_row == end_row {
				state.Copy_buffer = append(state.Copy_buffer, slices.Clone(state.Program_data[state.Buffer_idx][start_row][start_col:end_col + 1]))
			} else {
				state.Copy_buffer = append(state.Copy_buffer, slices.Clone(state.Program_data[state.Buffer_idx][start_row]))
				state.Copy_buffer[0] = state.Copy_buffer[0][start_col:]
		
				if start_row != end_row {
					for row := start_row + 1; row < end_row; row++ {
						state.Copy_buffer = append(state.Copy_buffer, slices.Clone(state.Program_data[state.Buffer_idx][row]))
					}

					state.Copy_buffer = append(state.Copy_buffer, slices.Clone(state.Program_data[state.Buffer_idx][end_row]))
					state.Copy_buffer[len(state.Copy_buffer) - 1] = state.Copy_buffer[len(state.Copy_buffer) - 1][:end_col + 1]
				}
			}

			state.Mode = 0
			state.Highlighting = false
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

