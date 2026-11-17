# Langton's ant in Go

A native Go simulation with an original retro sci-fi look: an orange armored
ant with animated legs, cyan energized tiles, a metallic grid, and a smooth
tracking camera. Uses Ebitengine for graphics. No game assets are copied.

## Rules

The ant starts at the middle of the visible grid, facing up, on a dark square.
Every other square starts light. On each step:

1. Turn right on a dark square, or left on a light square.
2. Flip the color of the square the ant is leaving.
3. Move forward one square.

The grid has no fixed boundary. The camera stays still until the ant approaches
an edge, then pans to keep it visible. Squares outside the window retain their
colors. Coordinates increase rightward and downward.

## Run

Install Go 1.22 or newer from https://go.dev/dl/, then open a terminal in this
folder:

```powershell
go run .
```

By default this opens a resizable graphical window when launched from a terminal.
Close the window or press **Esc** to stop. Movement and rotation are interpolated
at 60 updates per second, independently of the simulation speed. The camera
keeps the ant on screen, including after resizing or zooming.

| Control | Action |
| --- | --- |
| Space | Pause / resume |
| Up / Down | Increase / decrease simulation speed |
| Mouse wheel or + / - | Zoom in / out |
| Right arrow | Advance one step while paused |
| R | Reset the simulation (keeps current speed, zoom, and pause state) |
| G | Toggle grid lines |
| F | Toggle fullscreen |
| Esc | Exit |

`-delay` sets the initial simulation speed, capped at 240 steps per second in
the graphical view. `-steps` pauses the graphical view at the exact step limit;
reset to run it again. The terminal and plain modes exit at the step limit.
Width and height flags apply only to terminal and plain modes.

## Terminal mode

```powershell
go run . -terminal
```

Press **Ctrl+C** to stop. Use a modern terminal such as Windows Terminal or the
VS Code integrated terminal for this mode.

The animation uses a separate screen buffer and hides the cursor while running.
Ctrl+C restores your original terminal screen and cursor. It refreshes at up to
30 frames per second, independently of simulation speed. Terminal size is checked
each frame, so resizing the window automatically adjusts the grid without
scrolling. Very small windows show a resize prompt.

```powershell
# Faster animation, larger window
go run . -terminal -delay=20ms -width=41 -height=21

# Stop after 1,000 steps
go run . -steps=1000 -delay=10ms

# Plain snapshots without terminal control codes (useful for redirected output)
go run . -steps=5 -delay=0 -width=11 -height=7 -plain

# Show options
go run . -help
```

The terminal view uses black and light-gray cells and highlights the ant in
gold. In plain output, `#` represents a dark square and `.` a light square.
`^ > v <` show the ant's direction and cover the current square's color.
Redirected output automatically uses plain snapshots.

Defaults: up to 31 columns, 15 rows, 50 milliseconds per step, unlimited steps.
Width supports 1-200 columns and height supports 1-100 rows. These are maximum
dimensions in animated mode: the grid shrinks to fit the terminal. Each cell
takes two terminal characters. Delay must be nonnegative; zero batches up to
256 steps per frame to keep the display responsive. Plain mode prints every step.

## Build and test

```powershell
go test ./...
go vet ./...
go build -o ant.exe .
.\ant.exe
```

The simulation rules are in `ant.go`, native graphics are in `window.go`,
terminal animation is in `terminal.go`, and command-line options and plain
output are in `main.go`.

The optional native smoke test opens a window, captures an actual rendered
frame, and closes automatically:

```powershell
$env:ANT_WINDOW_SMOKE_IMAGE = "$env:TEMP\ant-preview.png"
go test -run TestNativeWindowSmoke -count=1
Remove-Item Env:\ANT_WINDOW_SMOKE_IMAGE
```
