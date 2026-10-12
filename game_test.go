package main

import (
	"math/rand/v2"
	"testing"
)

func testRNG() *rand.Rand { return rand.New(rand.NewPCG(1, 2)) }

// gameOf returns a game with the tiles of rows, 0 for an empty cell.
func gameOf(rows [size][size]int) *game {
	g := &game{rng: testRNG()}
	for r := range size {
		for c := range size {
			if v := rows[r][c]; v != 0 {
				g.nextID++
				g.tiles = append(g.tiles, &tile{ID: g.nextID, Value: v, Row: r, Col: c})
			}
		}
	}
	return g
}

// board returns the values of g's tiles by cell.
func board(g *game) (rows [size][size]int) {
	for _, t := range g.tiles {
		rows[t.Row][t.Col] = t.Value
	}
	return rows
}

// withoutNew returns rows less the tile g added after its last move.
func withoutNew(g *game, rows [size][size]int) [size][size]int {
	t := g.tiles[0]
	for _, u := range g.tiles {
		if u.ID > t.ID && !u.fromMerge {
			t = u
		}
	}
	rows[t.Row][t.Col] = 0
	return rows
}

func TestMoveRow(t *testing.T) {
	for _, tc := range []struct {
		name   string
		in     [size]int
		d      direction
		want   [size]int
		gained int
	}{
		{"slides", [size]int{0, 0, 0, 2}, left, [size]int{2, 0, 0, 0}, 0},
		{"merges", [size]int{2, 0, 0, 2}, left, [size]int{4, 0, 0, 0}, 4},
		{"merges each pair once", [size]int{2, 2, 2, 2}, left, [size]int{4, 4, 0, 0}, 8},
		{"keeps a merged tile", [size]int{2, 2, 4, 0}, left, [size]int{4, 4, 0, 0}, 4},
		{"merges toward the wall first", [size]int{2, 2, 2, 0}, right, [size]int{0, 0, 2, 4}, 4},
		{"merges the far pair", [size]int{0, 2, 2, 2}, left, [size]int{4, 2, 0, 0}, 4},
		{"leaves unequal tiles", [size]int{2, 4, 8, 0}, right, [size]int{0, 2, 4, 8}, 0},
		{"merges big tiles", [size]int{1024, 1024, 0, 0}, right, [size]int{0, 0, 0, 2048}, 2048},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g := gameOf([size][size]int{tc.in})
			moved, gained := g.move(tc.d)
			if !moved || gained != tc.gained {
				t.Fatalf("moved %v, gained %d; want true, %d", moved, gained, tc.gained)
			}
			if got := withoutNew(g, board(g)); got[0] != tc.want {
				t.Errorf("row %v, want %v", got[0], tc.want)
			}
			if g.score != tc.gained {
				t.Errorf("score %d, want %d", g.score, tc.gained)
			}
			if len(g.tiles) != countTiles(tc.want)+1 {
				t.Errorf("%d tiles, want %d and a new one", len(g.tiles), countTiles(tc.want))
			}
		})
	}
}

func countTiles(row [size]int) (n int) {
	for _, v := range row {
		if v != 0 {
			n++
		}
	}
	return n
}

func TestMoveColumns(t *testing.T) {
	g := gameOf([size][size]int{
		{2, 0, 0, 0},
		{2, 4, 0, 0},
		{0, 4, 0, 0},
		{4, 4, 0, 0},
	})
	g.move(down)
	want := [size][size]int{
		{0, 0, 0, 0},
		{0, 0, 0, 0},
		{4, 4, 0, 0},
		{4, 8, 0, 0},
	}
	if got := withoutNew(g, board(g)); got != want {
		t.Errorf("after down %v, want %v", got, want)
	}
	if g.score != 12 {
		t.Errorf("score %d, want 12", g.score)
	}
}

// A move that changes nothing adds no tile and keeps the last move's
// merges showing.
func TestMoveBlocked(t *testing.T) {
	g := gameOf([size][size]int{{2, 4, 0, 0}, {4, 2, 0, 0}})
	g.ghosts = []*tile{{ID: 99, Value: 2}}
	moved, gained := g.move(left)
	if moved || gained != 0 || len(g.tiles) != 4 {
		t.Fatalf("moved %v, gained %d, %d tiles; want false, 0, 4", moved, gained, len(g.tiles))
	}
	if len(g.ghosts) != 1 {
		t.Errorf("a blocked move cleared the ghosts")
	}
}

// The tiles a merge takes slide into its cell, keeping their IDs, and the
// tile they make is new.
func TestMergeGhosts(t *testing.T) {
	g := gameOf([size][size]int{{0, 2, 0, 2}})
	a, b := g.tiles[0], g.tiles[1]
	g.move(left)
	if len(g.ghosts) != 2 {
		t.Fatalf("%d ghosts, want 2", len(g.ghosts))
	}
	for _, gh := range g.ghosts {
		if gh != a && gh != b {
			t.Errorf("ghost %d is not one of the merged tiles", gh.ID)
		}
		if gh.Row != 0 || gh.Col != 0 {
			t.Errorf("ghost %d at %d,%d, want 0,0", gh.ID, gh.Row, gh.Col)
		}
	}
	var made *tile
	for _, t := range g.tiles {
		if t.fromMerge {
			made = t
		}
	}
	if made == nil || made.Value != 4 || made.ID == a.ID || made.ID == b.ID {
		t.Fatalf("merged tile %+v", made)
	}
}

func TestNewTiles(t *testing.T) {
	twos, fours := 0, 0
	rng := testRNG()
	for range 2000 {
		g := newGame(rng)
		if len(g.tiles) != 2 {
			t.Fatalf("a new game has %d tiles, want 2", len(g.tiles))
		}
		if g.tiles[0].Row == g.tiles[1].Row && g.tiles[0].Col == g.tiles[1].Col {
			t.Fatal("two tiles in one cell")
		}
		for _, t := range g.tiles {
			switch t.Value {
			case 2:
				twos++
			case 4:
				fours++
			}
		}
	}
	if twos+fours != 4000 {
		t.Fatalf("tiles other than 2 and 4")
	}
	if share := float64(fours) / 4000; share < 0.08 || share > 0.12 {
		t.Errorf("%.3f of new tiles are 4s, want about 0.1", share)
	}
}

func TestWin(t *testing.T) {
	g := gameOf([size][size]int{{1024, 1024, 0, 0}})
	g.move(left)
	if !g.won || !g.terminated() {
		t.Fatalf("won %v, terminated %v after making 2048", g.won, g.terminated())
	}
	if moved, _ := g.move(right); moved {
		t.Error("a won game took a move before keep going")
	}
	g.keepPlaying = true
	if g.terminated() {
		t.Error("terminated while keeping going")
	}
	if moved, _ := g.move(right); !moved {
		t.Error("no move after keep going")
	}
}

func TestGameOver(t *testing.T) {
	// The move fills the last cell; whatever comes, nothing can merge.
	g := gameOf([size][size]int{
		{0, 2, 4, 8},
		{8, 16, 32, 64},
		{128, 256, 512, 1024},
		{32, 128, 64, 256},
	})
	g.move(left)
	// The new tile went in the last cell, as a 2 or a 4 beside 8 and 16.
	if !g.over || !g.terminated() {
		t.Fatalf("over %v with %v", g.over, board(g))
	}
	if moved, _ := g.move(right); moved {
		t.Error("a lost game took a move")
	}

	g = gameOf([size][size]int{
		{2, 4, 2, 4},
		{4, 2, 4, 2},
		{2, 4, 2, 4},
		{4, 2, 4, 4},
	})
	if !g.movesAvailable() {
		t.Error("no moves available with two 4s side by side")
	}
}

func TestSaveRestore(t *testing.T) {
	g := gameOf([size][size]int{{2, 0, 4, 0}, {0, 8, 0, 0}})
	g.score, g.won, g.keepPlaying = 120, true, true
	r := restoreGame(g.save(), testRNG())
	if r == nil {
		t.Fatal("not restored")
	}
	if board(r) != board(g) || r.score != 120 || !r.won || !r.keepPlaying || r.nextID != g.nextID {
		t.Errorf("restored %v score %d won %v keep %v next %d", board(r), r.score, r.won, r.keepPlaying, r.nextID)
	}
	if moved, _ := r.move(left); !moved {
		t.Error("the restored game takes no move")
	}

	for name, s := range map[string]*savedGame{
		"empty":     {},
		"outside":   {Tiles: []tile{{ID: 1, Value: 2, Row: 4}}},
		"shared":    {Tiles: []tile{{ID: 1, Value: 2}, {ID: 2, Value: 4}}},
		"not power": {Tiles: []tile{{ID: 1, Value: 6}}},
		"over": {Tiles: []tile{
			{1, 2, 0, 0, false}, {2, 4, 0, 1, false}, {3, 2, 0, 2, false}, {4, 4, 0, 3, false},
			{5, 4, 1, 0, false}, {6, 2, 1, 1, false}, {7, 4, 1, 2, false}, {8, 2, 1, 3, false},
			{9, 2, 2, 0, false}, {10, 4, 2, 1, false}, {11, 2, 2, 2, false}, {12, 4, 2, 3, false},
			{13, 4, 3, 0, false}, {14, 2, 3, 1, false}, {15, 4, 3, 2, false}, {16, 2, 3, 3, false},
		}},
	} {
		if restoreGame(s, testRNG()) != nil {
			t.Errorf("restored a %s save", name)
		}
	}
}
