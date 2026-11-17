package main

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"math"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font/basicfont"
)

var (
	spaceColor = color.RGBA{8, 13, 23, 255}
	panelColor = color.RGBA{15, 25, 39, 255}
	gridColor  = color.RGBA{26, 43, 58, 255}
	cyanColor  = color.RGBA{83, 225, 214, 255}
	mutedColor = color.RGBA{100, 131, 147, 255}
	whiteColor = color.RGBA{220, 236, 232, 255}
	amberColor = color.RGBA{255, 166, 56, 255}
)

type windowGame struct {
	ctx                      context.Context
	s                        *simulation
	previous                 point
	previousFacing           direction
	phase, speed, zoom       float64
	cameraX, cameraY         float64
	width, height            int
	paused, grid, fullscreen bool
	limit                    uint64
	ticks                    uint64
}

func newWindowGame(ctx context.Context, delay time.Duration, limit uint64) *windowGame {
	speed := 240.0
	if delay > 0 {
		speed = min(240, float64(time.Second)/float64(delay))
	}
	return &windowGame{
		ctx: ctx, s: newSimulation(), speed: speed, zoom: 30,
		width: 1100, height: 760, grid: true, limit: limit,
	}
}

func runWindow(ctx context.Context, delay time.Duration, steps uint64) error {
	ebiten.SetWindowSize(1100, 760)
	ebiten.SetWindowTitle("ANT // XENO SURVEY")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowSizeLimits(640, 480, -1, -1)
	return ebiten.RunGame(newWindowGame(ctx, delay, steps))
}

func (g *windowGame) singleStep() {
	if g.limit != 0 && g.s.steps >= g.limit {
		return
	}
	g.previous, g.previousFacing = g.s.position, g.s.facing
	g.s.step()
}

func (g *windowGame) Update() error {
	if g.ctx.Err() != nil || inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		return ebiten.Termination
	}
	g.ticks++
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		g.paused = !g.paused
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyG) {
		g.grid = !g.grid
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF) {
		g.fullscreen = !g.fullscreen
		ebiten.SetFullscreen(g.fullscreen)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		g.s = newSimulation()
		g.previous, g.previousFacing = point{}, north
		g.phase, g.cameraX, g.cameraY = 0, 0, 0
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowRight) && g.paused {
		g.singleStep()
		g.phase = 1
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) {
		g.speed = min(240, g.speed*1.5)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) {
		g.speed = max(0.25, g.speed/1.5)
	}
	_, wheel := ebiten.Wheel()
	if wheel != 0 {
		g.zoom = max(10, min(64, g.zoom*math.Pow(1.12, wheel)))
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEqual) {
		g.zoom = min(64, g.zoom*1.2)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyMinus) {
		g.zoom = max(10, g.zoom/1.2)
	}

	g.advanceFrame()
	return nil
}

func (g *windowGame) advanceFrame() {
	if !g.paused {
		g.phase += g.speed / 60
		for g.phase >= 1 && (g.limit == 0 || g.s.steps < g.limit) {
			g.phase--
			g.singleStep()
		}
		if g.limit != 0 && g.s.steps >= g.limit {
			g.phase = min(1, g.phase)
			if g.phase >= 1 {
				g.paused = true
			}
		}
	}
	x, y := g.antPosition()
	marginX := max(1, float64(g.width-100)/g.zoom*0.18)
	marginY := max(1, float64(g.height-240)/g.zoom*0.18)
	g.cameraX += cameraTarget(g.cameraX, x, marginX) * 0.12
	g.cameraY += cameraTarget(g.cameraY, y, marginY) * 0.12
	// Keep the sprite inside the board even after a sudden resize or fast step.
	halfX := max(0.1, float64(g.width-56)/(2*g.zoom)-0.6)
	halfY := max(0.1, float64(g.height-222)/(2*g.zoom)-0.6)
	g.cameraX = max(x-halfX, min(x+halfX, g.cameraX))
	g.cameraY = max(y-halfY, min(y+halfY, g.cameraY))
}

func cameraTarget(camera, position, margin float64) float64 {
	if position < camera-margin {
		return position + margin - camera
	}
	if position > camera+margin {
		return position - margin - camera
	}
	return 0
}

func (g *windowGame) antPosition() (float64, float64) {
	t := min(1, g.phase)
	return float64(g.previous.x) + float64(g.s.position.x-g.previous.x)*t,
		float64(g.previous.y) + float64(g.s.position.y-g.previous.y)*t
}

func (g *windowGame) Layout(width, height int) (int, int) {
	g.width, g.height = max(1, width), max(1, height)
	return g.width, g.height
}

func rectangle(screen *ebiten.Image, x, y, w, h float64, c color.Color) {
	vector.DrawFilledRect(screen, float32(x), float32(y), float32(w), float32(h), c, false)
}

func label(screen *ebiten.Image, value string, x, y int, c color.Color) {
	text.Draw(screen, value, basicfont.Face7x13, x, y+13, c)
}

func (g *windowGame) Draw(screen *ebiten.Image) {
	screen.Fill(spaceColor)
	w, h := float64(g.width), float64(g.height)
	left, top, right, bottom := 28.0, 110.0, w-28, h-112
	centerX, centerY := (left+right)/2, (top+bottom)/2
	worldX := func(x float64) float64 { return centerX + (x-g.cameraX)*g.zoom }
	worldY := func(y float64) float64 { return centerY + (y-g.cameraY)*g.zoom }

	rectangle(screen, left, top, right-left, bottom-top, panelColor)
	board := screen.SubImage(image.Rect(int(left), int(top), int(right), int(bottom))).(*ebiten.Image)
	minX := int(math.Floor(g.cameraX + (left-centerX)/g.zoom - 1))
	maxX := int(math.Ceil(g.cameraX + (right-centerX)/g.zoom + 1))
	minY := int(math.Floor(g.cameraY + (top-centerY)/g.zoom - 1))
	maxY := int(math.Ceil(g.cameraY + (bottom-centerY)/g.zoom + 1))
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			sx, sy := worldX(float64(x)-0.5), worldY(float64(y)-0.5)
			if g.s.dark[point{x, y}] {
				rectangle(board, sx+1, sy+1, g.zoom-2, g.zoom-2, color.RGBA{20, 72, 81, 255})
				rectangle(board, sx+2, sy+2, g.zoom-4, 2, cyanColor)
				rectangle(board, sx+g.zoom/2-1, sy+g.zoom/2-1, 2, 2, cyanColor)
			}
			if g.grid {
				rectangle(board, sx, sy, g.zoom, 1, gridColor)
				rectangle(board, sx, sy, 1, g.zoom, gridColor)
			}
		}
	}
	// A fixed origin marker makes the camera's movement easy to read.
	ox, oy := worldX(0), worldY(0)
	vector.StrokeCircle(board, float32(ox), float32(oy), float32(g.zoom*0.36), 1,
		color.NRGBA{83, 225, 214, 75}, true)
	x, y := g.antPosition()
	g.drawAnt(board, worldX(x), worldY(y))

	for _, line := range [][4]float64{
		{left, top, right, top}, {left, bottom, right, bottom},
		{left, top, left, bottom}, {right, top, right, bottom},
	} {
		vector.StrokeLine(screen, float32(line[0]), float32(line[1]), float32(line[2]), float32(line[3]), 1, gridColor, false)
	}
	for _, corner := range [][2]float64{{left, top}, {right - 20, top}, {left, bottom}, {right - 20, bottom}} {
		rectangle(screen, corner[0], corner[1], 20, 2, cyanColor)
	}

	label(screen, "X E N O   /   S U R V E Y", 30, 24, cyanColor)
	label(screen, "ANT-01    AUTONOMOUS EXPLORER", 30, 48, whiteColor)
	label(screen, "UNBOUNDED SECTOR  //  LANGTON PROTOCOL", 30, 76, mutedColor)
	status := "RUNNING"
	if g.paused {
		status = "PAUSED"
	}
	label(screen, fmt.Sprintf("%s\n%5.1f STEPS / SEC", status, g.speed), g.width-180, 34, amberColor)
	label(screen, fmt.Sprintf("CYCLES  %08d       SECTOR  %d : %d       ACTIVE TILES  %d",
		g.s.steps, g.s.position.x, g.s.position.y, len(g.s.dark)), 30, g.height-88, whiteColor)
	label(screen, "SPACE pause   UP/DOWN speed   WHEEL or +/- zoom   R reset", 30, g.height-58, mutedColor)
	label(screen, "RIGHT step (paused)   G grid   F fullscreen   ESC exit", 30, g.height-36, mutedColor)
}

func (g *windowGame) drawAnt(screen *ebiten.Image, x, y float64) {
	scale := g.zoom / 30
	delta := float64(g.s.facing - g.previousFacing)
	if delta > 2 {
		delta -= 4
	}
	if delta < -2 {
		delta += 4
	}
	turn := min(1, g.phase*3)
	turn = turn * turn * (3 - 2*turn)
	angle := (float64(g.previousFacing) + delta*turn) * math.Pi / 2
	transform := func(px, py float64) (float32, float32) {
		return float32(x + (px*math.Cos(angle)-py*math.Sin(angle))*scale),
			float32(y + (px*math.Sin(angle)+py*math.Cos(angle))*scale)
	}
	line := func(ax, ay, bx, by float64, width float32, c color.Color) {
		x1, y1 := transform(ax, ay)
		x2, y2 := transform(bx, by)
		vector.StrokeLine(screen, x1, y1, x2, y2, width*float32(scale), c, true)
	}
	dot := func(px, py, radius float64, c color.Color) {
		sx, sy := transform(px, py)
		vector.DrawFilledCircle(screen, sx, sy, float32(radius*scale), c, true)
	}
	for radius := 22; radius >= 14; radius -= 4 {
		vector.DrawFilledCircle(screen, float32(x), float32(y), float32(float64(radius)*scale),
			color.NRGBA{255, 135, 32, 12}, true)
	}
	walk := 0.0
	if !g.paused {
		walk = math.Sin(float64(g.ticks) * 0.45)
	}
	for side := -1.0; side <= 1; side += 2 {
		for i := 0; i < 3; i++ {
			py := float64(i*6 - 6)
			swing := walk * 2 * math.Cos(float64(i)*math.Pi)
			line(side*4, py, side*10, py+3+swing, 2, amberColor)
			line(side*10, py+3+swing, side*13, py+7+swing, 2, mutedColor)
		}
		line(side*3, -9, side*7, -15, 1, cyanColor)
	}
	dot(0, 6, 6, color.RGBA{171, 71, 29, 255})
	dot(0, 4, 4, amberColor)
	dot(0, -2, 5, color.RGBA{232, 98, 29, 255})
	dot(0, -8, 5, amberColor)
	line(-3, -10, 3, -10, 3, cyanColor)
	dot(-1, -3, 1.5, color.RGBA{255, 224, 146, 255})
}
