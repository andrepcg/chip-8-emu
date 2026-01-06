package chip8

import (
	"fmt"
	"math/rand"
	"os"
	"path"
)

const (
	FB_WIDTH            = 64
	FB_HEIGHT           = 32
	CHIP8_PROGRAM_START = 0x200
	DIGITS_LEN          = 5
	OPERATIONS_PER_SEC  = 16
)

type Chip8 struct {
	V           [16]byte
	I, PC       uint16
	SP, DT, ST  uint8
	STACK       [16]uint16
	FRAMEBUFFER [FB_WIDTH * FB_HEIGHT]bool
	RAM         [4096]byte
	KEYBOARD    uint16
}

func (cpu *Chip8) Initialize(romPath string) {
	// Load digits
	println("Loading digits")
	digits := LoadRomFromFile("chip8/digits.rom")
	cpu.LoadRom(digits, 0)

	rom := LoadRomFromFile(path.Join("roms", romPath))

	// Load ROM
	fmt.Printf("Loading ROM with size %d bytes\n", len(rom))
	cpu.LoadRom(rom, CHIP8_PROGRAM_START)
	cpu.PC = CHIP8_PROGRAM_START

	println("Initialization complete")
}

func LoadRomFromFile(filePath string) []byte {
	fmt.Printf("Loading file %s\n", filePath)

	dat, err := os.ReadFile(filePath)

	if err != nil {
		panic(err)
	}

	fmt.Printf("Loaded %s. Size = %d bytes\n", filePath, len(dat))

	return dat
}

func (cpu *Chip8) LoadRom(rom []byte, location uint16) {
	copy(cpu.RAM[location:], rom[:])
}

func (cpu *Chip8) PrintDebugCompact() {
	fmt.Printf("PC:0x%04X I:0x%04X SP:%d DT:%d ST:%d | ", cpu.PC, cpu.I, cpu.SP, cpu.DT, cpu.ST)
	fmt.Print("V:")
	for i := 0; i < 16; i++ {
		fmt.Printf("%02X ", cpu.V[i])
	}
	if cpu.PC < 4095 {
		instruction := uint16(cpu.RAM[cpu.PC])<<8 | uint16(cpu.RAM[cpu.PC+1])
		fmt.Printf("| Instr:0x%04X", instruction)
	}
	fmt.Println()
}

func (cpu *Chip8) IsKeyPressed(key byte) bool {
	if key > 15 {
		return false
	}
	return (cpu.KEYBOARD>>uint16(16-1-key))&0x1 > 0
}

func (cpu *Chip8) IsAnyKeyPressed() bool {
	return cpu.KEYBOARD > 0
}

func (cpu *Chip8) PressedKeys() []uint8 {
	var keys []uint8

	for k := byte(0x0); k < 0xF; k++ {
		if cpu.IsKeyPressed(k) {
			keys = append(keys, uint8(k))
		}
	}

	return keys
}

func (cpu *Chip8) UpdateKeyboard(pressedKeys []byte) {
	cpu.KEYBOARD = 0
	for _, v := range pressedKeys {
		cpu.KEYBOARD |= uint16(1) << (16 - 1 - v)
	}
}

func (cpu *Chip8) StackPush(value uint16) {
	cpu.STACK[cpu.SP] = value
	cpu.SP += 1
}

func (cpu *Chip8) StackPop() uint16 {
	cpu.SP -= 1
	v := cpu.STACK[cpu.SP]
	return v
}

func (cpu *Chip8) Fetch() uint16 {
	if cpu.PC < CHIP8_PROGRAM_START {
		fmt.Printf("PC = 0x%x\n", cpu.PC)
		panic("CORRUPTED PROGRAM COUNTER")
	}

	byte1 := uint16(cpu.RAM[cpu.PC])
	byte2 := uint16(cpu.RAM[cpu.PC+1])
	cpu.PC += 2
	return (byte1 << 8) | byte2
}

func (cpu *Chip8) DrawSprite(vx, vy byte, bytes []byte) {
	// These bytes are then displayed as sprites on screen at coordinates (Vx, Vy).
	// Sprites are XORed onto the existing screen. If this causes any pixels to be erased, VF is set to 1,
	// otherwise it is set to 0. If the sprite is positioned so part of it is outside the coordinates of the display,
	// it wraps around to the opposite side of the screen.

	cpu.V[0xF] = 0

	for dy, v := range bytes {
		for index := byte(0); index < 8; index += 1 {
			dx := 8 - 1 - index
			pixel := ((v >> dx) & 1) > 0
			x := uint16(vx+index) % FB_WIDTH
			y := uint16(vy+byte(dy)) % FB_HEIGHT
			loc := int(y)*FB_WIDTH + int(x)
			old := cpu.FRAMEBUFFER[loc]

			if pixel {
				if old {
					cpu.V[0xF] = 1
				}
				cpu.FRAMEBUFFER[loc] = !old
			}
		}
	}
}

func (cpu *Chip8) DecodeExecute(instruction uint16) {
	switch instruction {
	case 0x00E0:
		clear(cpu.FRAMEBUFFER[:])
		return
	case 0x00EE:
		// The interpreter sets the program counter to the address at the top of the stack, then subtracts 1 from the stack pointer.
		cpu.PC = cpu.StackPop()
		return
	}

	x := instruction & 0x0F00 >> 8
	y := instruction & 0x00F0 >> 4

	// fmt.Printf("Cur instruction 0x%x\n", instruction)

	if instruction&0xF000 == 0x1000 { // 1nnn - JP addr
		cpu.PC = instruction & 0x0FFF
	} else if instruction&0xF000 == 0x2000 { // 2nnn - CALL addr
		// The interpreter increments the stack pointer, then puts the current PC on the top of the stack. The PC is then set to nnn.
		cpu.StackPush(cpu.PC)
		cpu.PC = instruction & 0x0FFF
	} else if instruction&0xF000 == 0x3000 { // 3xkk - SE Vx, byte
		// Skip next instruction if Vx = kk. The interpreter compares register Vx to kk, and if they are equal, increments the program counter by 2.
		if cpu.V[x] == byte(instruction&0x00FF) {
			cpu.PC += 2
		}
	} else if instruction&0xF000 == 0x4000 { // 4xkk - SNE Vx, byte
		// Skip next instruction if Vx != kk. The interpreter compares register Vx to kk, and if they are not equal, increments the program counter by 2.
		if cpu.V[x] != byte(instruction) {
			cpu.PC += 2
		}
	} else if instruction&0xF00F == 0x5000 { // 5xy0 - SE Vx, Vy
		//Skip next instruction if Vx = Vy. The interpreter compares register Vx to register Vy, and if they are equal, increments the program counter by 2.
		if cpu.V[x] == cpu.V[y] {
			cpu.PC += 2
		}
	} else if instruction&0xF000 == 0x6000 { // 6xkk - LD Vx, byte
		// Set Vx = kk. The interpreter puts the value kk into register Vx.
		cpu.V[x] = byte(instruction & 0x00FF)
	} else if instruction&0xF000 == 0x7000 { // 7xkk - ADD Vx, byte
		// Set Vx = Vx + kk. Adds the value kk to the value of register Vx, then stores the result in Vx.
		cpu.V[x] += byte(instruction & 0x00FF)
	} else if instruction&0xF00F == 0x8000 { // 8xy0 - LD Vx, Vy
		// Set Vx = Vy. Stores the value of register Vy in register Vx.
		cpu.V[x] = cpu.V[y]
	} else if instruction&0xF00F == 0x8001 { // 8xy1 - OR Vx, Vy
		// Set Vx = Vx OR Vy. Performs a bitwise OR on the values of Vx and Vy, then stores the result in Vx. A
		// bitwise OR compares the corresponding bits from two values, and if either bit is 1, then the same bit in the
		// result is also 1. Otherwise, it is 0.
		cpu.V[x] = cpu.V[x] | cpu.V[y]
	} else if instruction&0xF00F == 0x8002 { // 8xy2 - AND Vx, Vy
		cpu.V[x] = cpu.V[x] & cpu.V[y]
	} else if instruction&0xF00F == 0x8003 { // 8xy3 - XOR Vx, Vy
		cpu.V[x] = cpu.V[x] ^ cpu.V[y]
	} else if instruction&0xF00F == 0x8004 { // 8xy4 - ADD Vx, Vy
		// Set Vx = Vx + Vy, set VF = carry. The values of Vx and Vy are added together. If the result is greater
		// than 8 bits (i.e., ¿ 255,) VF is set to 1, otherwise 0. Only the lowest 8 bits of the result are kept, and stored in Vx.
		sum := uint16(cpu.V[x]) + uint16(cpu.V[y])

		if sum > 0xFF {
			cpu.V[0xF] = 1
		} else {
			cpu.V[0xF] = 0
		}
		cpu.V[x] = byte(sum)
	} else if instruction&0xF00F == 0x8005 { // 8xy5 - SUB Vx, Vy
		// Set Vx = Vx - Vy, set VF = NOT borrow. If Vx > Vy, then VF is set to 1, otherwise 0. Then Vy is
		// subtracted from Vx, and the results stored in Vx.
		vx := cpu.V[x]
		vy := cpu.V[y]
		cpu.V[x] = vx - vy
		if vx >= vy {
			cpu.V[0xF] = 1
		} else {
			cpu.V[0xF] = 0
		}
	} else if instruction&0xF00F == 0x8006 { // 8xy6 - SHR Vx {, Vy}
		// Set Vx = Vx SHR 1. If the least-significant bit of Vx is 1, then VF is set to 1, otherwise 0. Then Vx is divided by 2.
		vx := cpu.V[x]
		cpu.V[x] = vx >> 1
		cpu.V[0xF] = vx & 0x01
	} else if instruction&0xF00F == 0x8007 { // 8xy7 - SUBN Vx, Vy
		// Set Vx = Vy - Vx, set VF = NOT borrow.
		// If Vy >= Vx, then VF is set to 1, otherwise 0. Then Vx is subtracted from Vy, and the results stored in Vx.
		vx := cpu.V[x]
		vy := cpu.V[y]
		cpu.V[x] = vy - vx
		if vy >= vx {
			cpu.V[0xF] = 1
		} else {
			cpu.V[0xF] = 0
		}
	} else if instruction&0xF00F == 0x800E { // 8xyE - SHL Vx {, Vy}
		// Set Vx = Vx SHL 1.
		// If the most-significant bit of Vx is 1, then VF is set to 1, otherwise to 0. Then Vx is multiplied by 2.
		vx := cpu.V[x]
		cpu.V[x] = vx << 1
		cpu.V[0xF] = (vx >> 7) & 0x01
	} else if instruction&0xF00F == 0x9000 { // 9xy0 - SNE Vx, Vy
		// Skip next instruction if Vx != Vy.
		// The values of Vx and Vy are compared, and if they are not equal, the program counter is increased by 2.
		if cpu.V[y] != cpu.V[x] {
			cpu.PC += 2
		}
	} else if instruction&0xF000 == 0xA000 { // Annn - LD I, addr
		// Set I = nnn.
		cpu.I = instruction & 0x0FFF
	} else if instruction&0xF000 == 0xB000 { // Bnnn - JP V0, addr
		// The program counter is set to nnn plus the value of V0.
		cpu.PC = instruction&0x0FFF + uint16(cpu.V[0])
	} else if instruction&0xF000 == 0xC000 { // Cxkk - RND Vx, byte
		// The interpreter generates a random number from 0 to 255, which is then ANDed with the value kk. The results are stored in Vx.
		cpu.V[x] = byte(rand.Intn(256)) & byte(instruction&0x00FF)
	} else if instruction&0xF000 == 0xD000 { // Dxyn - DRW Vx, Vy, nibble
		// Display n-byte sprite starting at memory location I at (Vx, Vy), set VF = collision.
		// The interpreter reads n bytes from memory, starting at the address stored in I. These bytes are then displayed as sprites on
		// screen at coordinates (Vx, Vy). Sprites are XORed onto the existing screen. If this causes any pixels to be erased, VF is set to 1,
		// otherwise it is set to 0. If the sprite is positioned so part of it is outside the coordinates of the display,
		// it wraps around to the opposite side of the screen.

		n := instruction & 0x000F
		cpu.DrawSprite(cpu.V[x], cpu.V[y], cpu.RAM[cpu.I:(cpu.I+n)])
	} else if instruction&0xF0FF == 0xE09E { // Ex9E - SKP Vx
		// Skip next instruction if key with the value of Vx is pressed.
		// Checks the keyboard, and if the key corresponding to the value of Vx is currently in the down position, PC is increased by 2.

		if cpu.IsKeyPressed(cpu.V[x]) {
			cpu.PC += 2
		}
	} else if instruction&0xF0FF == 0xE0A1 { // ExA1 - SKNP Vx
		if !cpu.IsKeyPressed(cpu.V[x]) {
			cpu.PC += 2
		}
	} else if instruction&0xF0FF == 0xF007 { // Fx07 - LD Vx, DT
		// The value of DT is placed into Vx.
		cpu.V[x] = cpu.DT
	} else if instruction&0xF0FF == 0xF00A { // Fx0A - LD Vx, K
		// All execution stops until a key is pressed, then the value of that key is stored in Vx.

		for !cpu.IsAnyKeyPressed() {
		} // busy wait

		// FIXME: this is not totally correct
		cpu.V[x] = cpu.PressedKeys()[0]
	} else if instruction&0xF0FF == 0xF015 { // Fx15 - LD DT, Vx
		// DT is set equal to the value of Vx.
		cpu.DT = cpu.V[x]
	} else if instruction&0xF0FF == 0xF018 { // Fx18 - LD ST, Vx
		cpu.ST = cpu.V[x]
	} else if instruction&0xF0FF == 0xF01E { // Fx1E - ADD I, Vx
		// Set I = I + Vx.
		cpu.I += uint16(cpu.V[x])
	} else if instruction&0xF0FF == 0xF029 { // Fx29 - LD F, Vx
		// Set I = location of sprite for digit Vx.
		// The value of I is set to the location for the hexadecimal sprite corresponding to the value of Vx.

		cpu.I = uint16(cpu.V[x]) * DIGITS_LEN
	} else if instruction&0xF0FF == 0xF033 { // Fx33 - LD B, Vx
		// The interpreter takes the decimal value of Vx, and places the hundreds digit in memory at location in I, the tens digit at location I+1, and the ones digit at location I+2.

		cpu.RAM[cpu.I] = cpu.V[x] / 100
		cpu.RAM[cpu.I+1] = (cpu.V[x] / 10) % 10
		cpu.RAM[cpu.I+2] = cpu.V[x] % 10
	} else if instruction&0xF0FF == 0xF055 { // Fx55 - LD [I], Vx
		// The interpreter copies the values of registers V0 through Vx into memory, starting at the address in I.
		copy(cpu.RAM[cpu.I:cpu.I+x+1], cpu.V[0:x+1])
	} else if instruction&0xF0FF == 0xF065 { // Fx65 - LD Vx, [I]
		// The interpreter reads values from memory starting at location I into registers V0 through Vx.
		copy(cpu.V[0:x+1], cpu.RAM[cpu.I:cpu.I+x+1])
	}

}

func (cpu *Chip8) UpdateTimers() {
	if cpu.ST > 0 {
		cpu.ST -= 1
	}

	if cpu.DT > 0 {
		cpu.DT -= 1
	}
}

func (cpu *Chip8) Step() {
	instruction := cpu.Fetch()
	cpu.DecodeExecute(instruction)
}
