package main

import (
	"context"
	"errors"
	"fmt"
	"image"
	"image/png"
	"math"
	"os"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestWindowMotionPreservesSimulation(t *testing.T) {
	g := newWindowGame(context.Background(), 50*time.Millisecond, 100)
	want := newSimulation()
	for i := 0; i < 100; i++ {
		want.step()
	}
	for i := 0; i < 400; i++ {
		g.advanceFrame()
	}
	if g.s.steps != 100 || !g.paused || g.phase != 1 {
		t.Fatalf("finite animation did not settle at limit: steps=%d paused=%v phase=%v", g.s.steps, g.paused, g.phase)
	}
	if g.s.position != want.position || g.s.facing != want.facing || len(g.s.dark) != len(want.dark) {
		t.Fatal("graphical simulation differs from core rules")
	}
	for p := range want.dark {
		if !g.s.dark[p] {
			t.Fatalf("tile %v has wrong color", p)
		}
	}
}

func TestWindowPauseAndSingleStep(t *testing.T) {
	g := newWindowGame(context.Background(), 50*time.Millisecond, 1)
	g.paused = true
	for i := 0; i < 10; i++ {
		g.advanceFrame()
	}
	if g.s.steps != 0 || g.phase != 0 {
		t.Fatal("paused simulation advanced")
	}
	g.singleStep()
	g.singleStep()
	if g.s.steps != 1 || g.s.position != (point{1, 0}) {
		t.Fatal("single step ignored limit or rules")
	}
}

func TestWindowCameraKeepsAntInsideBoard(t *testing.T) {
	g := newWindowGame(context.Background(), 0, 0)
	g.width, g.height, g.zoom = 640, 480, 64
	for i := 0; i < 20000; i++ {
		g.advanceFrame()
		x, y := g.antPosition()
		px := float64(g.width)/2 + (x-g.cameraX)*g.zoom
		py := float64(g.height-2)/2 + (y-g.cameraY)*g.zoom
		radius := 0.5 * g.zoom
		if px-radius < 28 || px+radius > float64(g.width-28) ||
			py-radius < 110 || py+radius > float64(g.height-112) {
			t.Fatalf("ant off board at step %d: (%v,%v)", g.s.steps, px, py)
		}
	}
}

func TestWindowInterpolation(t *testing.T) {
	g := newWindowGame(context.Background(), time.Second, 0)
	g.singleStep()
	g.phase = 0.5
	x, y := g.antPosition()
	if math.Abs(x-0.5) > 0.0001 || y != 0 {
		t.Fatalf("expected halfway between cells, got (%v,%v)", x, y)
	}
}

func TestMain(m *testing.M) {
	if path := os.Getenv("ANT_WINDOW_SMOKE_IMAGE"); path != "" {
		// Ebitengine must run on the main OS thread, not a test goroutine.
		if err := captureNativeWindow(path); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	os.Exit(m.Run())
}

func TestNativeWindowSmoke(t *testing.T) {
	path := os.Getenv("ANT_WINDOW_SMOKE_IMAGE")
	if path == "" {
		t.Skip("set ANT_WINDOW_SMOKE_IMAGE to capture a native smoke-test frame")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}

func captureNativeWindow(path string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	game := &windowSmoke{windowGame: newWindowGame(ctx, 50*time.Millisecond, 0), path: path}
	ebiten.SetWindowSize(1100, 760)
	ebiten.SetWindowTitle("ANT // visual verification")
	if err := ebiten.RunGame(game); err != nil {
		return err
	}
	if game.captureErr != nil {
		return game.captureErr
	}
	if !game.captured {
		return errors.New("native game exited without rendering the verification frame")
	}
	return nil
}

type windowSmoke struct {
	*windowGame
	path       string
	captured   bool
	captureErr error
}

func (g *windowSmoke) Update() error {
	if g.captured || g.captureErr != nil {
		return ebiten.Termination
	}
	return g.windowGame.Update()
}

func (g *windowSmoke) Draw(screen *ebiten.Image) {
	g.windowGame.Draw(screen)
	if g.ticks < 240 || g.captured || g.captureErr != nil {
		return
	}
	bounds := screen.Bounds()
	pixels := image.NewRGBA(bounds)
	screen.ReadPixels(pixels.Pix)
	file, err := os.Create(g.path)
	if err != nil {
		g.captureErr = err
		return
	}
	g.captureErr = errors.Join(png.Encode(file, pixels), file.Close())
	g.captured = g.captureErr == nil
}
