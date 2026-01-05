# Chip8 Emulator Go

This project is a small experiment for me to learn a bit of Go. No AI was used to develop this since I wanted to learn things by myself without having the code be fully or partially generated.

## Bugs encountered along the way due to poor Go experience

- the `Chip8` type functions implemented originally received `(cpu Chip8)` instead of a pointer (`(cpu *Chip8)`) which lead to interesting scenarios
- the calculated FRAMEBUFFER index (`loc`) in the `DrawSprite` method was being incorrectly calculated due to a type overflow when calculating `x` and `y` as the multiplication was still being performed on `uint8`

## TODO

- [x] Keyboard input
- [x] Proper display rendering (no printf to terminal)
- [x] Remove the `Cpu` interface, i don't think it serves any purpose
- [ ] Better clock generation
- [ ] Beep tone
- [ ] Would the keyboard be better represented with a `uint16` instead or `[16]bool`?