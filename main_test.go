//go:build !js

package main

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

const testW, testH = 560, 840

const howTo = "How to play: Use the arrow keys, or drag, to move the tiles. When two tiles with the same number touch, they merge into one!"

func testApp(t *testing.T, rows [size][size]int) (*app, *ui.Tester) {
	t.Helper()
	a := newApp(nil, testRNG())
	a.g = gameOf(rows)
	a.g.rng = a.rng
	return a, ui.NewTester(a.view, testW, testH)
}

func TestKeys(t *testing.T) {
	for _, tc := range []struct {
		key  ui.Key
		want [size]int
	}{
		{ui.KeyLeft, [size]int{4, 0, 0, 0}},
		{ui.KeyRight, [size]int{0, 0, 0, 4}},
		{ui.KeyA, [size]int{4, 0, 0, 0}},
		{ui.KeyD, [size]int{0, 0, 0, 4}},
		{ui.KeyH, [size]int{4, 0, 0, 0}},
		{ui.KeyL, [size]int{0, 0, 0, 4}},
	} {
		a, tt := testApp(t, [size][size]int{{0, 2, 2, 0}})
		tt.Key(0, tc.key)
		if got := withoutNew(a.g, board(a.g))[0]; got != tc.want {
			t.Errorf("key %v: row %v, want %v", tc.key, got, tc.want)
		}
		if a.g.score != 4 || !tt.HasText("4") {
			t.Errorf("key %v: score %d, texts %q", tc.key, a.g.score, tt.Texts())
		}
	}

	a, tt := testApp(t, [size][size]int{{2}, {2}})
	tt.Key(0, ui.KeyDown)
	if got := withoutNew(a.g, board(a.g)); got[3][0] != 4 {
		t.Errorf("after Down %v", got)
	}
	a, tt = testApp(t, [size][size]int{{}, {}, {2}, {2}})
	tt.Key(0, ui.KeyW)
	if got := withoutNew(a.g, board(a.g)); got[0][0] != 4 {
		t.Errorf("after W %v", got)
	}
}

func TestBestScore(t *testing.T) {
	a, tt := testApp(t, [size][size]int{{2, 2}})
	a.best = 2
	tt.Key(0, ui.KeyLeft)
	if a.best != 4 || !tt.HasText("4") {
		t.Errorf("best %d after scoring 4", a.best)
	}
	a.best = 100
	tt.Key(0, ui.KeyRight)
	if a.best != 100 {
		t.Errorf("best %d, want 100 kept", a.best)
	}
}

func TestNewGame(t *testing.T) {
	a, tt := testApp(t, [size][size]int{{2, 2, 4, 8}})
	tt.Key(0, ui.KeyLeft)
	old := a.g
	if err := tt.Click("New Game"); err != nil {
		t.Fatal(err)
	}
	if a.g == old || a.g.score != 0 || len(a.g.tiles) != 2 {
		t.Errorf("after New Game: score %d, %d tiles", a.g.score, len(a.g.tiles))
	}
	old = a.g
	tt.Key(0, ui.KeyR)
	if a.g == old {
		t.Error("R did not start a new game")
	}
	old = a.g
	tt.Key(ui.Cmd, ui.KeyN)
	if a.g == old {
		t.Error("Cmd+N did not start a new game")
	}
}

func TestWinMessage(t *testing.T) {
	a, tt := testApp(t, [size][size]int{{1024, 1024}})
	tt.Key(0, ui.KeyLeft)
	if !tt.HasText("You win!") {
		t.Fatalf("no win message: %q", tt.Texts())
	}
	if got := tt.Announcements(); len(got) != 1 {
		t.Errorf("announced %q", got)
	}
	tt.Key(0, ui.KeyRight)
	if a.g.tiles[0].Col != 0 {
		t.Error("the tiles moved under the win message")
	}
	if err := tt.Click("Keep going"); err != nil {
		t.Fatal(err)
	}
	if !a.g.keepPlaying || tt.HasText("You win!") {
		t.Fatalf("keep playing %v, texts %q", a.g.keepPlaying, tt.Texts())
	}
	tt.Key(0, ui.KeyRight)
	if withoutNew(a.g, board(a.g))[0][3] != 2048 {
		t.Errorf("the 2048 did not move right: %v", board(a.g))
	}
}

func TestGameOverMessage(t *testing.T) {
	a, tt := testApp(t, [size][size]int{
		{0, 2, 4, 8},
		{8, 16, 32, 64},
		{128, 256, 512, 1024},
		{32, 128, 64, 256},
	})
	tt.Key(0, ui.KeyLeft)
	if !a.g.over || !tt.HasText("Game over!") || tt.HasText("Keep going") {
		t.Fatalf("over %v, texts %q", a.g.over, tt.Texts())
	}
	if err := tt.Click("Try again"); err != nil {
		t.Fatal(err)
	}
	if a.g.over || tt.HasText("Game over!") || len(a.g.tiles) != 2 {
		t.Errorf("after Try again: over %v, texts %q", a.g.over, tt.Texts())
	}
}

func TestDrag(t *testing.T) {
	for _, tc := range []struct {
		dx, dy float32
		want   [size][size]int
	}{
		{-120, 10, [size][size]int{{4}}},
		{0, 120, [size][size]int{{}, {}, {}, {2, 0, 0, 2}}},
	} {
		a, tt := testApp(t, [size][size]int{{2, 0, 0, 2}})
		r, ok := tt.Find("Board")
		if !ok {
			t.Fatal("no board")
		}
		x, y := r.X+r.W/2, r.Y+r.H/2
		tt.Press(x, y)
		tt.Move(x+tc.dx/2, y+tc.dy/2)
		tt.Release(x+tc.dx, y+tc.dy)
		if got := withoutNew(a.g, board(a.g)); got != tc.want {
			t.Errorf("drag %v,%v: %v, want %v", tc.dx, tc.dy, got, tc.want)
		}
	}

	// A click is no swipe.
	a, tt := testApp(t, [size][size]int{{2, 0, 0, 2}})
	r, _ := tt.Find("Board")
	tt.ClickAt(r.X+r.W/2, r.Y+r.H/2)
	if len(a.g.tiles) != 2 {
		t.Errorf("a click moved the tiles: %v", board(a.g))
	}
}

// A touchpad swipe moves the tiles once, however long its momentum goes on.
func TestTouchpadSwipe(t *testing.T) {
	a := newApp(nil, testRNG())
	a.g = gameOf([size][size]int{{0, 0, 2, 0}})
	a.g.rng = a.rng
	now := time.Unix(1000, 0)
	a.clock = func() time.Time { return now }
	scroll := func(dx float32) {
		now = now.Add(16 * time.Millisecond)
		a.swipe(ui.InputEvent{Kind: ui.InputScroll, DX: dx, Precise: true}, 1)
	}
	for range 30 {
		scroll(8) // content moving left
	}
	if got := board(a.g); got[0][0] != 2 || len(a.g.tiles) != 2 {
		t.Fatalf("after a swipe: %v", got)
	}

	// Once the fingers rest, the next swipe moves the tiles again.
	now = now.Add(time.Second)
	for range 10 {
		scroll(-8)
	}
	if got := board(a.g); got[0][3] != 2 || len(a.g.tiles) != 3 {
		t.Errorf("after a second swipe: %v", got)
	}

	// A mouse wheel scrolls by lines, and moves nothing.
	now = now.Add(time.Second)
	if a.swipe(ui.InputEvent{Kind: ui.InputScroll, DX: 100}, 1) || len(a.g.tiles) != 3 {
		t.Error("a mouse wheel moved the tiles")
	}
}

func TestSaved(t *testing.T) {
	st := &store{path: filepath.Join(t.TempDir(), "state.json")}
	a := newApp(st, testRNG())
	a.g = gameOf([size][size]int{{2, 2}})
	a.g.rng = a.rng
	a.move(left)
	b := newApp(st, testRNG())
	if board(b.g) != board(a.g) || b.g.score != 4 || b.best != 4 {
		t.Fatalf("reopened %v score %d best %d; want %v 4 4", board(b.g), b.g.score, b.best, board(a.g))
	}

	// A lost game is not kept, but its best score is.
	b.g = gameOf([size][size]int{
		{0, 2, 4, 8},
		{8, 16, 32, 64},
		{128, 256, 512, 1024},
		{32, 128, 64, 256},
	})
	b.g.rng, b.g.score = b.rng, 5000
	b.move(left)
	c := newApp(st, testRNG())
	if c.g.over || c.g.score != 0 || c.best != 5000 {
		t.Errorf("after a lost game: over %v score %d best %d", c.g.over, c.g.score, c.best)
	}

	// A damaged save starts a new game.
	os.WriteFile(st.path, []byte("{"), 0o644)
	if d := newApp(st, testRNG()); len(d.g.tiles) != 2 || d.best != 0 {
		t.Errorf("from a damaged save: %d tiles, best %d", len(d.g.tiles), d.best)
	}
}

// The board fits windows of any size.
func TestFits(t *testing.T) {
	for _, win := range [][2]int{{320, 500}, {560, 840}, {1200, 900}, {900, 1400}} {
		_, tt := testApp(t, [size][size]int{{2}})
		tt.SetSize(win[0], win[1])
		r, ok := tt.Find("Board")
		if !ok {
			t.Fatal("no board")
		}
		if r.X < 0 || r.Y < 0 || r.X+r.W > float32(win[0]) || r.Y+r.H > float32(win[1]) {
			t.Errorf("at %v the board is at %+v", win, r)
		}
		if h, ok := tt.Find(howTo); !ok || h.Y+h.H > float32(win[1]) {
			t.Errorf("at %v the help is at %+v", win, h)
		}
	}
}

// TestSnapshots writes pictures of the game to $SNAPSHOTS, to look at.
func TestSnapshots(t *testing.T) {
	dir := os.Getenv("SNAPSHOTS")
	if dir == "" {
		t.Skip("SNAPSHOTS is not set")
	}
	shoot := func(name string, tt *ui.Tester) {
		time.Sleep(2100 * time.Millisecond) // past every animation
		tt.Frame()
		f, err := os.Create(filepath.Join(dir, name+".png"))
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		png.Encode(f, tt.Image())
	}
	a, tt := testApp(t, [size][size]int{
		{2, 4, 8, 16},
		{32, 64, 128, 256},
		{512, 1024, 2048, 4096},
		{8192, 16384, 0, 0},
	})
	a.g.score, a.best = 20484, 31337
	tt.SetScale(2)
	shoot("tiles", tt)
	tt.Key(0, ui.KeyRight)
	time.Sleep(150 * time.Millisecond)
	tt.Frame()
	f, _ := os.Create(filepath.Join(dir, "rise.png"))
	png.Encode(f, tt.Image())
	f.Close()

	a, tt = testApp(t, [size][size]int{{1024, 1024, 4, 2}, {2, 8}})
	tt.SetScale(2)
	tt.Key(0, ui.KeyLeft)
	shoot("win", tt)

	a, tt = testApp(t, [size][size]int{
		{0, 2, 4, 8},
		{8, 16, 32, 64},
		{128, 256, 512, 1024},
		{32, 128, 64, 256},
	})
	tt.Key(0, ui.KeyLeft)
	shoot("over", tt)

	_, tt = testApp(t, [size][size]int{{2, 0, 0, 2}, {4}})
	tt.SetSize(340, 520)
	shoot("small", tt)
}
