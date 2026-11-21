package main

import (
	"fmt"
	"log"
	"os"
	"ti_forge/ansi"
	"ti_forge/input"
	"ti_forge/render"
	"ti_forge/state"
	"ti_forge/state/settings"

	"golang.org/x/term"
)

func main() {
	state := state.State{
		Quit:            false,
		Cursor_row:      0,
		Cursor_col:      0,
		Mode:            0,
		Text_buffer:     []rune{},
		Suggestion_idx:  0,
		Input:           []byte{},
		Buffers:         [][][]string{{{"　"}}},
		Indentation:     [][]int{{0}},
		File_names:      []string{""},
		Buffer_idx:      0,
		Highlight_row:   0,
		Highlight_col:   0,
		Highlighting:    false,
		Copy_buffer:     [][]string{},
		Settings:        settings.Settings{},
		Command_matches: []string{},
	}

	config_dir, err := os.UserConfigDir()
	if err != nil {
		log.Fatal("Failed to get config directory: ", err)
	}
	config_dir = config_dir + "/ti_forge"

	state.Settings, err = settings.Load_settings(config_dir + "/config.json")
	if err != nil {
		state.Text_buffer = []rune("Failed to load \"" + config_dir + "/config.json\", defaulting settings\n")
	}

	fd := int(os.Stdin.Fd())
	cooked_state, err := term.MakeRaw(fd)
	if err != nil {
		log.Fatal("Failed to enter raw mode: ", err)
	}
	defer term.Restore(fd, cooked_state)
	fmt.Print(ansi.Hide_cursor)

	for {
		render.Display_data(&state)
		input.Get_input(&state)
		input.Process_input(&state)

		if state.Quit {
			fmt.Print(ansi.Clear)
			fmt.Print(ansi.Reset_cursor)
			fmt.Print(ansi.Reset_text)
			fmt.Print(ansi.Show_cursor)
			return
		}
	}
}
