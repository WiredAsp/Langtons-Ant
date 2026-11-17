package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
)

const (
	enterScreen = "\x1b[?1049h\x1b[?25l\x1b[2J"
	leaveScreen = "\x1b[0m\x1b[?25h\x1b[?1049l"
	framePeriod = time.Second / 30
)

type viewport struct {
	center point
}

func trackAxis(center, position, size int) int {
	margin := min(3, (size-1)/2)
	low := center - size/2 + margin
	high := center - size/2 + size - 1 - margin
	if position < low {
		return center + position - low
	}
	if position > high {
		return center + position - high
	}
	return center
}

func (v *viewport) draw(output io.Writer, s *simulation, width, height, columns, rows int) error {
	if columns <= 0 || rows <= 0 {
		return fmt.Errorf("invalid terminal size: %dx%d", columns, rows)
	}
	var frame strings.Builder
	frame.WriteString("\x1b[H")
	if columns < 7 || rows < 7 {
		message := "Resize terminal"
		frame.WriteString(message[:min(len(message), max(0, columns-1))])
		frame.WriteString("\x1b[0m\x1b[J")
		_, err := io.WriteString(output, frame.String())
		return err
	}

	// Reserve the rightmost column and bottom row so drawing never scrolls.
	width = min(width, (columns-3)/2)
	height = min(height, rows-6)
	v.center.x = trackAxis(v.center.x, s.position.x, width)
	v.center.y = trackAxis(v.center.y, s.position.y, height)
	header := func(text, style string) {
		frame.WriteString(style)
		frame.WriteString(text[:min(len(text), columns-1)])
		frame.WriteString("\x1b[0m\x1b[K\r\n")
	}
	header("LANGTON'S ANT", "\x1b[1;96m")
	header(fmt.Sprintf("Step: %d | Position: (%d, %d)", s.steps, s.position.x, s.position.y), "\x1b[37m")
	header("Gold: ant | Dark: right | Light: left | Ctrl+C: quit", "\x1b[90m")
	border := "+" + strings.Repeat("--", width) + "+"
	header(border, "\x1b[36m")
	for row := 0; row < height; row++ {
		frame.WriteString("\x1b[36m|")
		for col := 0; col < width; col++ {
			p := point{v.center.x + col - width/2, v.center.y + row - height/2}
			switch {
			case p == s.position:
				frame.WriteString("\x1b[1;30;103m")
				frame.WriteByte("^>v<"[s.facing])
				frame.WriteByte(' ')
			case s.dark[p]:
				frame.WriteString("\x1b[40m  ")
			default:
				frame.WriteString("\x1b[47m  ")
			}
			frame.WriteString("\x1b[0m")
		}
		frame.WriteString("\x1b[36m|\x1b[0m\x1b[K\r\n")
	}
	frame.WriteString("\x1b[36m")
	frame.WriteString(border)
	frame.WriteString("\x1b[0m\x1b[J")
	_, err := io.WriteString(output, frame.String())
	return err
}

func advance(s *simulation, delay, elapsed time.Duration, pending *time.Duration, limit uint64) {
	budget := 256
	if delay > 0 {
		*pending += elapsed
		budget = int(min(int64(budget), int64(*pending/delay)))
	}
	for i := 0; i < budget && (limit == 0 || s.steps < limit); i++ {
		s.step()
		if delay > 0 {
			*pending -= delay
		}
	}
}

func animate(ctx context.Context, output io.Writer, s *simulation, width, height int,
	delay time.Duration, steps uint64, size func() (int, int, error)) (err error) {
	defer func() {
		_, restoreErr := io.WriteString(output, leaveScreen)
		err = errors.Join(err, restoreErr)
	}()
	if _, err = io.WriteString(output, enterScreen); err != nil {
		return err
	}
	var view viewport
	draw := func() error {
		columns, rows, err := size()
		if err != nil {
			return fmt.Errorf("read terminal size: %w", err)
		}
		return view.draw(output, s, width, height, columns, rows)
	}
	if err := draw(); err != nil {
		return err
	}
	ticker := time.NewTicker(framePeriod)
	defer ticker.Stop()
	last := time.Now()
	var pending time.Duration
	for steps == 0 || s.steps < steps {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
		now := time.Now()
		advance(s, delay, now.Sub(last), &pending, steps)
		last = now
		if err := draw(); err != nil {
			return err
		}
	}
	return nil
}
