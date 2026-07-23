package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"ti_forge/ansi"
	"ti_forge/input"
	"ti_forge/render"
	"ti_forge/state"

	"golang.org/x/term"
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
	fmt.Print(ansi.Hide_cursor)

	for {
		state = render.Display_data(state)
		state = input.Get_input(state)
		state = input.Process_input(state)
		
		if state.Quit {
			fmt.Print(ansi.Clear)
			fmt.Print(ansi.Reset_cursor)
			fmt.Print(ansi.Reset_text)
			fmt.Print(ansi.Show_cursor)
			return
		}
	}
}

func enable_utf8() {
	cmd := exec.Command("cmd.exe", "/c", "chcp 65001 > nul")
	cmd.Stdout = os.Stdout
	cmd.Run()
}

