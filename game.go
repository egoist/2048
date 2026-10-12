package main

import "math/rand/v2"

// size is the number of cells on each side of the board.
const size = 4

// winValue is the tile that wins the game.
const winValue = 2048

type direction int

const (
	up direction = iota
	right
	down
	left
)

// vector returns the step a tile takes moving in d, in rows and columns.
func (d direction) vector() (dr, dc int) {
	switch d {
	case up:
		return -1, 0
	case right:
		return 0, 1
	case down:
		return 1, 0
	default:
		return 0, -1
	}
}

// tile is a numbered tile on the board. Its ID stays the same while it
// slides, so the view can animate it from where it was.
type tile struct {
	ID    int `json:"id"`
	Value int `json:"value"`
	Row   int `json:"row"`
	Col   int `json:"col"`

	// fromMerge is set on a tile two others made, which pops in where
	// they met, rather than growing in as a new tile does.
	fromMerge bool
}

// game is the state of one game of 2048.
type game struct {
	tiles []*tile
	// ghosts are the tiles the last move merged away. They slide into the
	// cell of their merge and stay under the merged tile until the next
	// move, so the merge reads as two tiles meeting.
	ghosts []*tile

	score int
	// won is set when a 2048 tile first appears, and keepPlaying once the
	// player chooses to carry on past it.
	won, keepPlaying bool
	over             bool

	nextID int
	rng    *rand.Rand
}

func newGame(rng *rand.Rand) *game {
	g := &game{rng: rng}
	g.addRandomTile()
	g.addRandomTile()
	return g
}

// terminated reports whether the board takes no more moves: the game is
// lost, or won and the player has not chosen to keep going.
func (g *game) terminated() bool {
	return g.over || (g.won && !g.keepPlaying)
}

// grid returns the live tiles by cell.
func (g *game) grid() (cells [size][size]*tile) {
	for _, t := range g.tiles {
		cells[t.Row][t.Col] = t
	}
	return cells
}

func (g *game) emptyCells() [][2]int {
	cells := g.grid()
	var empty [][2]int
	for r := range size {
		for c := range size {
			if cells[r][c] == nil {
				empty = append(empty, [2]int{r, c})
			}
		}
	}
	return empty
}

// addRandomTile puts a 2, or one time in ten a 4, in a random empty cell.
func (g *game) addRandomTile() {
	empty := g.emptyCells()
	if len(empty) == 0 {
		return
	}
	cell := empty[g.rng.IntN(len(empty))]
	value := 2
	if g.rng.Float64() >= 0.9 {
		value = 4
	}
	g.nextID++
	g.tiles = append(g.tiles, &tile{ID: g.nextID, Value: value, Row: cell[0], Col: cell[1]})
}

// move slides every tile as far as it goes in d, merging each pair of
// equal tiles that meet once, and reports whether anything moved and the
// points the merges made. A move that changes the board adds a tile.
func (g *game) move(d direction) (moved bool, gained int) {
	if g.terminated() {
		return false, 0
	}
	cells := g.grid()
	var ghosts []*tile
	// joined marks the cells whose tile this move made, which take no
	// other: 2 2 4 moved left makes 4 4, not 8.
	var joined [size][size]bool
	dr, dc := d.vector()

	// Tiles nearest the wall they move toward go first, so each lands
	// against the one ahead of it.
	order := [size]int{0, 1, 2, 3}
	reversed := [size]int{3, 2, 1, 0}
	rows, cols := order, order
	if dr == 1 {
		rows = reversed
	}
	if dc == 1 {
		cols = reversed
	}

	for _, r := range rows {
		for _, c := range cols {
			t := cells[r][c]
			if t == nil {
				continue
			}
			// Find the farthest empty cell in the way, and what is past it.
			fr, fc := r, c
			for {
				nr, nc := fr+dr, fc+dc
				if !inside(nr, nc) || cells[nr][nc] != nil {
					break
				}
				fr, fc = nr, nc
			}
			nr, nc := fr+dr, fc+dc
			cells[r][c] = nil
			if inside(nr, nc) {
				if next := cells[nr][nc]; next.Value == t.Value && !joined[nr][nc] {
					g.nextID++
					cells[nr][nc] = &tile{ID: g.nextID, Value: t.Value * 2, Row: nr, Col: nc, fromMerge: true}
					joined[nr][nc] = true
					t.Row, t.Col = nr, nc
					ghosts = append(ghosts, next, t)
					gained += t.Value * 2
					if t.Value*2 == winValue {
						g.won = true
					}
					moved = true
					continue
				}
			}
			cells[fr][fc] = t
			if fr != r || fc != c {
				t.Row, t.Col = fr, fc
				moved = true
			}
		}
	}

	if !moved {
		return false, 0
	}
	g.ghosts = ghosts
	g.tiles = g.tiles[:0]
	for r := range size {
		for c := range size {
			if t := cells[r][c]; t != nil {
				g.tiles = append(g.tiles, t)
			}
		}
	}
	g.score += gained
	g.addRandomTile()
	if !g.movesAvailable() {
		g.over = true
	}
	return true, gained
}

// movesAvailable reports whether some move would change the board.
func (g *game) movesAvailable() bool {
	cells := g.grid()
	for r := range size {
		for c := range size {
			t := cells[r][c]
			if t == nil {
				return true
			}
			if r+1 < size && cells[r+1][c] != nil && cells[r+1][c].Value == t.Value {
				return true
			}
			if c+1 < size && cells[r][c+1] != nil && cells[r][c+1].Value == t.Value {
				return true
			}
		}
	}
	return false
}

func inside(r, c int) bool {
	return r >= 0 && r < size && c >= 0 && c < size
}

// savedGame is a game as it is stored between launches.
type savedGame struct {
	Tiles       []tile `json:"tiles"`
	Score       int    `json:"score"`
	Won         bool   `json:"won"`
	KeepPlaying bool   `json:"keepPlaying"`
	NextID      int    `json:"nextId"`
}

func (g *game) save() *savedGame {
	s := &savedGame{Score: g.score, Won: g.won, KeepPlaying: g.keepPlaying, NextID: g.nextID}
	for _, t := range g.tiles {
		s.Tiles = append(s.Tiles, tile{ID: t.ID, Value: t.Value, Row: t.Row, Col: t.Col})
	}
	return s
}

// restoreGame returns the game s saved, or nil when s does not hold a
// board that can be played.
func restoreGame(s *savedGame, rng *rand.Rand) *game {
	if s == nil || len(s.Tiles) == 0 {
		return nil
	}
	g := &game{score: s.Score, won: s.Won, keepPlaying: s.KeepPlaying, nextID: s.NextID, rng: rng}
	var seen [size][size]bool
	for _, t := range s.Tiles {
		if !inside(t.Row, t.Col) || seen[t.Row][t.Col] || t.Value < 2 || t.Value&(t.Value-1) != 0 {
			return nil
		}
		seen[t.Row][t.Col] = true
		g.nextID = max(g.nextID, t.ID)
		g.tiles = append(g.tiles, &tile{ID: t.ID, Value: t.Value, Row: t.Row, Col: t.Col})
	}
	if !g.movesAvailable() {
		return nil
	}
	return g
}
