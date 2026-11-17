package main

type point struct {
	x, y int
}

type direction int

const (
	north direction = iota
	east
	south
	west
)

type simulation struct {
	dark     map[point]bool
	position point
	facing   direction
	steps    uint64
}

func newSimulation() *simulation {
	return &simulation{
		dark: map[point]bool{{0, 0}: true},
	}
}

func (s *simulation) step() {
	if s.dark[s.position] {
		s.facing = (s.facing + 1) % 4
		delete(s.dark, s.position)
	} else {
		s.facing = (s.facing + 3) % 4
		s.dark[s.position] = true
	}

	switch s.facing {
	case north:
		s.position.y--
	case east:
		s.position.x++
	case south:
		s.position.y++
	case west:
		s.position.x--
	}
	s.steps++
}
