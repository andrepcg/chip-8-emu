package main

import (
	chip8 "andrepcg/chip8emu/chip8"
	"fmt"
	"io/fs"
	"log"
	"os"
	"strings"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	WINDOW_SCALING = 13
	TIMERS_HZ      = 60
)

var windowWidth int32 = chip8.FB_WIDTH * WINDOW_SCALING
var windowHeight int32 = chip8.FB_HEIGHT * WINDOW_SCALING
var showDebug bool = false
var monoFont rl.Font

var KEY_MAP = map[int32]byte{
	rl.KeyA:     0xA,
	rl.KeyB:     0xB,
	rl.KeyC:     0xC,
	rl.KeyD:     0xD,
	rl.KeyE:     0xE,
	rl.KeyF:     0xF,
	rl.KeyZero:  0x0,
	rl.KeyOne:   0x1,
	rl.KeyTwo:   0x2,
	rl.KeyThree: 0x3,
	rl.KeyFour:  0x4,
	rl.KeyFive:  0x5,
	rl.KeySix:   0x6,
	rl.KeySeven: 0x7,
	rl.KeyEight: 0x8,
	rl.KeyNine:  0x9,
}

func RenderDisplay(cpu *chip8.Chip8) {

	for y := 0; y < chip8.FB_HEIGHT; y += 1 {
		for x := 0; x < chip8.FB_WIDTH; x += 1 {
			if cpu.FRAMEBUFFER[y*chip8.FB_WIDTH+x] {
				rl.DrawRectangle(int32(x*WINDOW_SCALING), int32(y*WINDOW_SCALING), WINDOW_SCALING, WINDOW_SCALING, rl.White)
			}
		}
	}

	if showDebug {
		renderCpuDebug(cpu)
	}
}

func PressedKeys() []byte {
	var keys []byte

	for k, v := range KEY_MAP {
		if rl.IsKeyDown(k) {
			keys = append(keys, v)
		}
	}
	return keys
}

var currentRomCursorIndex int = 0
var availableRoms []string
var currentScreen Screen = RomSelect

func RenderRomSelectScreen(cpu *chip8.Chip8) {
	rl.ClearBackground(rl.RayWhite)

	if rl.IsKeyReleased(rl.KeyDown) {
		currentRomCursorIndex++
		currentRomCursorIndex %= len(availableRoms)
	} else if rl.IsKeyReleased(rl.KeyUp) {
		if currentRomCursorIndex > 0 {
			currentRomCursorIndex--
		} else {
			currentRomCursorIndex = len(availableRoms) - 1
		}
	} else if rl.IsKeyReleased(rl.KeyEnter) {
		cpu.Initialize(availableRoms[currentRomCursorIndex])
		currentScreen = EmulatorScreen
	}

	baseY := float32(30)
	baseX := float32(45)

	for i, romPath := range availableRoms {
		s := fmt.Sprintf("%d - %s", i, romPath)
		currentY := int(baseY)*(i+1) - 6*i

		if currentY >= int(windowHeight) {
			baseX += 120
		}

		rl.DrawText(s, int32(baseX), int32(currentY), 14, rl.DarkGray)

		if currentRomCursorIndex == i {
			rl.DrawCircle(int32(baseX)-10, int32(currentY)+6, 5, rl.Red)
		}
	}
}

func renderCpuDebug(cpu *chip8.Chip8) {

	var sb strings.Builder
	fmt.Fprintf(&sb, "PC:0x%04X\tI:0x%04X\tSP:%2d\tDT:%2d\tST:%2d\t|\tV:", cpu.PC, cpu.I, cpu.SP, cpu.DT, cpu.ST)

	for i := 0; i < len(cpu.V); i++ {
		fmt.Fprintf(&sb, "%02X\t", cpu.V[i])
	}

	rl.DrawTextEx(monoFont, sb.String(), rl.Vector2{X: 0, Y: 0}, 15, 0, rl.RayWhite)
}

func RenderEmulatorScreen(cpu *chip8.Chip8) {
	rl.ClearBackground(rl.DarkGray)

	cpu.UpdateKeyboard(PressedKeys())
	// cpu.Step()

	RenderDisplay(cpu)
	// cpu.PrintDebugCompact()
}

type Screen int

const (
	RomSelect Screen = iota
	EmulatorScreen
)

func listRoms() []string {
	root := os.DirFS("roms")

	mdFiles, err := fs.Glob(root, "*.ch8")

	if err != nil {
		log.Fatal(err)
	}

	var files []string
	for _, v := range mdFiles {
		files = append(files, v)
	}
	return files
}

func init() {
	availableRoms = listRoms()
	showDebug = false
}

func main() {
	cpu := new(chip8.Chip8)

	rl.InitWindow(windowWidth, windowHeight, "chip8 emulator")
	defer rl.CloseWindow()

	rl.SetTargetFPS(30)

	monoFont = rl.LoadFontEx("LTSuperiorMono-Regular.otf", 16, nil, 0)

	for !rl.WindowShouldClose() {
		rl.BeginDrawing()

		switch currentScreen {
		case RomSelect:
			RenderRomSelectScreen(cpu)
		case EmulatorScreen:
			RenderEmulatorScreen(cpu)
		}

		rl.EndDrawing()
	}
}
