package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"time"

	"golang.org/x/term"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := run(ctx, os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "ant:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, output io.Writer) error {
	flags := flag.NewFlagSet("ant", flag.ContinueOnError)
	flags.SetOutput(output)
	width := flags.Int("width", 31, "visible grid columns (1-200)")
	height := flags.Int("height", 15, "visible grid rows (1-100)")
	delay := flags.Duration("delay", 50*time.Millisecond, "time between steps (e.g. 50ms)")
	steps := flags.Uint64("steps", 0, "steps to simulate; 0 runs until Ctrl+C")
	plain := flags.Bool("plain", false, "print snapshots without terminal escape sequences")
	terminal := flags.Bool("terminal", false, "use the terminal animation instead of the graphical window")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected arguments: %s", strings.Join(flags.Args(), " "))
	}
	if *width < 1 || *width > 200 || *height < 1 || *height > 100 {
		return errors.New("width must be 1-200 and height must be 1-100")
	}
	if *delay < 0 {
		return errors.New("delay must not be negative")
	}

	s := newSimulation()
	if file, ok := output.(*os.File); !*plain && !*terminal && ok && file == os.Stdout && term.IsTerminal(int(file.Fd())) {
		return runWindow(ctx, *delay, *steps)
	}
	if file, ok := output.(*os.File); !*plain && ok && term.IsTerminal(int(file.Fd())) {
		return animate(ctx, output, s, *width, *height, *delay, *steps, func() (int, int, error) {
			return term.GetSize(int(file.Fd()))
		})
	}
	if err := renderPlain(output, s, *width, *height); err != nil {
		return err
	}

	timer := time.NewTimer(*delay)
	defer timer.Stop()
	for *steps == 0 || s.steps < *steps {
		select {
		case <-ctx.Done():
			return nil
		case <-timer.C:
		}
		s.step()
		if err := renderPlain(output, s, *width, *height); err != nil {
			return err
		}
		timer.Reset(*delay)
	}
	return nil
}

func renderPlain(output io.Writer, s *simulation, width, height int) error {
	var frame strings.Builder
	fmt.Fprintf(&frame, "Langton's ant | Step: %-20d\n", s.steps)
	frame.WriteString("Dark: #  Light: .  Ant: ^ > v <  |  Ctrl+C to stop\n")
	fmt.Fprintf(&frame, "Position: (%d, %d) | View follows ant", s.position.x, s.position.y)
	frame.WriteByte('\n')
	frame.WriteString("+" + strings.Repeat("--", width) + "+\n")
	for row := 0; row < height; row++ {
		frame.WriteByte('|')
		for col := 0; col < width; col++ {
			p := point{s.position.x + col - width/2, s.position.y + row - height/2}
			cell := byte('.')
			if s.dark[p] {
				cell = '#'
			}
			if p == s.position {
				cell = "^>v<"[s.facing]
			}
			frame.WriteByte(cell)
			frame.WriteByte(' ')
		}
		frame.WriteString("|\n")
	}
	frame.WriteString("+" + strings.Repeat("--", width) + "+\n")
	_, err := io.WriteString(output, frame.String())
	return err
}
