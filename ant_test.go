package main

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
)

func TestInitialState(t *testing.T) {
	s := newSimulation()
	if s.position != (point{0, 0}) || s.facing != north || s.steps != 0 {
		t.Fatalf("unexpected initial state: %+v", s)
	}
	if len(s.dark) != 1 || !s.dark[point{0, 0}] {
		t.Fatal("only the starting square should be dark")
	}
}

func TestTurningFlippingAndMovement(t *testing.T) {
	moves := []point{{0, -1}, {1, 0}, {0, 1}, {-1, 0}}
	for facing := north; facing <= west; facing++ {
		for _, dark := range []bool{false, true} {
			s := newSimulation()
			s.position = point{12, -9}
			s.facing = facing
			s.dark = map[point]bool{s.position: dark}
			start := s.position
			wantFacing := (facing + 3) % 4
			if dark {
				wantFacing = (facing + 1) % 4
			}
			s.step()
			wantPosition := point{start.x + moves[wantFacing].x, start.y + moves[wantFacing].y}
			if s.facing != wantFacing || s.position != wantPosition || s.steps != 1 {
				t.Errorf("facing=%d dark=%v: got %+v, want facing=%d position=%v",
					facing, dark, s, wantFacing, wantPosition)
			}
			if s.dark[start] == dark {
				t.Errorf("facing=%d dark=%v: square did not flip", facing, dark)
			}
		}
	}
}

func TestFirstFiveSteps(t *testing.T) {
	s := newSimulation()
	want := []struct {
		position point
		facing   direction
	}{
		{point{1, 0}, east},
		{point{1, -1}, north},
		{point{0, -1}, west},
		{point{0, 0}, south},
		{point{1, 0}, east},
	}
	for i, state := range want {
		s.step()
		if s.position != state.position || s.facing != state.facing {
			t.Fatalf("step %d: position=%v facing=%d; want %+v", i+1, s.position, s.facing, state)
		}
	}
	if len(s.dark) != 4 || !s.dark[point{0, 0}] {
		t.Fatalf("unexpected colors after revisiting origin: %v", s.dark)
	}
}

func TestFiniteRun(t *testing.T) {
	var out bytes.Buffer
	err := run(context.Background(), []string{"-steps=5", "-delay=0", "-width=3", "-height=3", "-plain"}, &out)
	if err != nil {
		t.Fatal(err)
	}
	text := out.String()
	if strings.Count(text, "Langton's ant") != 6 || !strings.Contains(text, "Step: 5") {
		t.Fatalf("expected initial frame plus five steps: %s", text)
	}
	if strings.Contains(text, "\x1b") {
		t.Fatal("plain output contains escape sequences")
	}
	if !strings.Contains(text, "Position: (1, 0)") {
		t.Fatal("final position missing")
	}
}

func TestInvalidArguments(t *testing.T) {
	for _, args := range [][]string{
		{"-width=0"}, {"-width=201"}, {"-height=0"}, {"-height=101"},
		{"-delay=-1ms"}, {"-steps=-1"}, {"unexpected"}, {"-unknown"},
	} {
		if err := run(context.Background(), args, &bytes.Buffer{}); err == nil {
			t.Errorf("expected error for %v", args)
		}
	}
}

func TestHelpAndCancellation(t *testing.T) {
	var out bytes.Buffer
	if err := run(context.Background(), []string{"-help"}, &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "-steps") {
		t.Fatal("help missing flag documentation")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	out.Reset()
	if err := run(ctx, []string{"-plain", "-delay=1h"}, &out); err != nil {
		t.Fatal(err)
	}
	if strings.Count(out.String(), "Langton's ant") != 1 {
		t.Fatal("cancelled run should stop after initial frame")
	}
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, errors.New("output failed")
}

func TestOutputFailure(t *testing.T) {
	for _, args := range [][]string{{}, {"-plain"}} {
		if err := run(context.Background(), args, failingWriter{}); err == nil {
			t.Fatal("expected output failure")
		}
	}
}
