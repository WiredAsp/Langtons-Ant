package main

import (
	"bytes"
	"context"
	"errors"
	"regexp"
	"strings"
	"testing"
	"time"
)

var terminalEscape = regexp.MustCompile(`\x1b\[[0-9;?]*[A-Za-z]`)

func TestViewportFitsTerminalAndTracksAnt(t *testing.T) {
	s := newSimulation()
	var view viewport
	sizes := [][2]int{{80, 24}, {40, 12}, {7, 7}, {120, 40}, {1, 1}, {6, 6}}
	for step := 0; step < 12000; step++ {
		s.step()
		if step%37 != 0 {
			continue
		}
		for _, size := range sizes {
			var out bytes.Buffer
			if err := view.draw(&out, s, 200, 100, size[0], size[1]); err != nil {
				t.Fatal(err)
			}
			text := terminalEscape.ReplaceAllString(out.String(), "")
			lines := strings.Split(text, "\r\n")
			if len(lines) > max(1, size[1]-1) {
				t.Fatalf("frame exceeds %d rows: %d", size[1], len(lines))
			}
			for _, line := range lines {
				if len(line) > size[0]-1 {
					t.Fatalf("frame exceeds %d columns: %q", size[0], line)
				}
			}
			if size[0] >= 7 && size[1] >= 7 {
				if strings.Count(out.String(), "\x1b[1;30;103m") != 1 {
					t.Fatalf("ant missing or duplicated at step %d, size %v", step, size)
				}
			}
		}
	}
}

func TestCameraStaysStillAwayFromEdge(t *testing.T) {
	var view viewport
	s := newSimulation()
	for i := 0; i < 10; i++ {
		if err := view.draw(&bytes.Buffer{}, s, 31, 15, 80, 24); err != nil {
			t.Fatal(err)
		}
		if view.center != (point{}) {
			t.Fatalf("camera moved unnecessarily: %v", view.center)
		}
		s.step()
	}
}

func TestAdvanceTimingAndFiniteLimit(t *testing.T) {
	s := newSimulation()
	var pending time.Duration
	advance(s, 10*time.Millisecond, 25*time.Millisecond, &pending, 0)
	if s.steps != 2 || pending != 5*time.Millisecond {
		t.Fatalf("steps=%d pending=%v", s.steps, pending)
	}
	advance(s, 10*time.Millisecond, 5*time.Millisecond, &pending, 0)
	if s.steps != 3 || pending != 0 {
		t.Fatalf("steps=%d pending=%v", s.steps, pending)
	}
	advance(s, 0, time.Second, &pending, 5)
	if s.steps != 5 {
		t.Fatalf("finite run overshot: %d", s.steps)
	}
	s = newSimulation()
	advance(s, 0, 0, &pending, 0)
	if s.steps != 256 {
		t.Fatalf("unlimited speed must have bounded work per frame: %d", s.steps)
	}
}

func TestAnimationRestoresTerminal(t *testing.T) {
	size := func() (int, int, error) { return 80, 24, nil }
	t.Run("finite", func(t *testing.T) {
		var out bytes.Buffer
		s := newSimulation()
		if err := animate(context.Background(), &out, s, 31, 15, 0, 5, size); err != nil {
			t.Fatal(err)
		}
		if s.steps != 5 || !strings.HasPrefix(out.String(), enterScreen) || !strings.HasSuffix(out.String(), leaveScreen) {
			t.Fatal("animation did not complete and restore screen")
		}
	})
	t.Run("cancel", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		var out bytes.Buffer
		if err := animate(ctx, &out, newSimulation(), 31, 15, time.Hour, 0, size); err != nil {
			t.Fatal(err)
		}
		if !strings.HasSuffix(out.String(), leaveScreen) {
			t.Fatal("cancel did not restore terminal")
		}
	})
	t.Run("size failure", func(t *testing.T) {
		var out bytes.Buffer
		want := errors.New("size failed")
		err := animate(context.Background(), &out, newSimulation(), 31, 15, 0, 1,
			func() (int, int, error) { return 0, 0, want })
		if !errors.Is(err, want) || !strings.HasSuffix(out.String(), leaveScreen) {
			t.Fatal("size failure must be surfaced and terminal restored")
		}
	})
	t.Run("output failure", func(t *testing.T) {
		if err := animate(context.Background(), failingWriter{}, newSimulation(), 31, 15, 0, 1, size); err == nil {
			t.Fatal("output failure must be surfaced")
		}
	})
}

func TestRedirectedOutputIsPlain(t *testing.T) {
	var out bytes.Buffer
	if err := run(context.Background(), []string{"-steps=1", "-delay=0"}, &out); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "\x1b") {
		t.Fatal("redirected output must not contain terminal controls")
	}
}

func TestInvalidTerminalSize(t *testing.T) {
	var view viewport
	if err := view.draw(&bytes.Buffer{}, newSimulation(), 31, 15, 0, 0); err == nil {
		t.Fatal("invalid terminal size must be reported")
	}
}
