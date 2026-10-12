package main

import (
	"log"
	"math/rand/v2"
	"path/filepath"
	"strconv"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// The board's measures at a scale of 1, as in the original game; the view
// scales them all to fit the window.
const (
	pageW   = 500 // the width of the column holding everything
	boardW  = 500
	gap     = 15
	cellW   = (boardW - gap*(size+1)) / size
	cellGap = cellW + gap // from one cell to the next
	// pageH is the height of everything, to fit the window.
	pageH = 790
)

var (
	pageBg    = ui.Hex("#faf8ef")
	ink       = ui.Hex("#776e65")
	boardBg   = ui.Hex("#bbada0")
	cellBg    = ui.RGBA(238, 228, 218, 0.35)
	lightInk  = ui.Hex("#f9f6f2")
	buttonBg  = ui.Hex("#8f7a66")
	labelInk  = ui.Hex("#eee4da")
	riseInk   = ui.RGBA(119, 110, 101, 0.9)
	overBg    = ui.RGBA(238, 228, 218, 0.73)
	wonBg     = ui.RGBA(237, 194, 46, 0.5)
	superTile = ui.Hex("#3c3a32")
	white     = ui.Hex("#ffffff")
	black     = ui.Hex("#000000")
)

// tileColors are the backgrounds of tiles up to 2048.
var tileColors = map[int]ui.Color{
	2:    ui.Hex("#eee4da"),
	4:    ui.Hex("#ede0c8"),
	8:    ui.Hex("#f2b179"),
	16:   ui.Hex("#f59563"),
	32:   ui.Hex("#f67c5f"),
	64:   ui.Hex("#f65e3b"),
	128:  ui.Hex("#edcf72"),
	256:  ui.Hex("#edcc61"),
	512:  ui.Hex("#edc850"),
	1024: ui.Hex("#edc53f"),
	2048: ui.Hex("#edc22e"),
}

// tileGlow is how strongly the gold tiles glow, outside and along their
// edge.
var tileGlow = map[int][2]float32{
	128:  {0.238, 0.143},
	256:  {0.317, 0.190},
	512:  {0.397, 0.238},
	1024: {0.476, 0.286},
	2048: {0.556, 0.333},
}

// Timings of the original game.
var (
	slide    = ui.Animation{Duration: 100 * time.Millisecond, Ease: ui.EaseInOut}
	popDelay = 100 * time.Millisecond // tiles appear once the others have slid
	appear   = []ui.Keyframe{{Value: 0}, {At: 200 * time.Millisecond, Value: 1, Ease: ui.EaseInOut}}
	pop      = []ui.Keyframe{
		{Value: 0},
		{At: 100 * time.Millisecond, Value: 1.2, Ease: ui.EaseInOut},
		{At: 200 * time.Millisecond, Value: 1, Ease: ui.EaseInOut},
	}
	rise = []ui.Keyframe{{Value: 0}, {At: 600 * time.Millisecond, Value: 1, Ease: ui.EaseIn}}
	// The message over a finished game fades in after a moment, so the
	// last move shows first.
	messageIn  = []ui.Keyframe{{Value: 0}, {At: 800 * time.Millisecond, Value: 1, Ease: ui.EaseInOut}}
	messageLag = 1200 * time.Millisecond
)

// Swipes: a drag this far moves the tiles, and a touchpad scroll this far;
// the scroll then takes no more until the fingers rest this long, so the
// momentum after a swipe does not move them again.
const (
	dragMin   = 10
	scrollMin = 40
	scrollEnd = 180 * time.Millisecond
)

type tileKey int
type cellKey int
type riseKey int

type app struct {
	g    *game
	best int

	// rise counts the moves that scored, to show each move's points
	// rising from the score, and risen is the last one's.
	rise, risen int

	// The pointer pressed on the board, and the touchpad scrolling over it.
	pressing         bool
	pressX, pressY   float32
	scrollX, scrollY float32
	scrollLast       time.Time
	scrollSpent      bool
	clock            func() time.Time

	// announced is what screen readers last heard of the game's end.
	announced string

	rng   *rand.Rand
	store *store
}

func newApp(st *store, rng *rand.Rand) *app {
	a := &app{store: st, rng: rng, clock: time.Now}
	if st != nil {
		s := st.load()
		a.best = s.Best
		a.g = restoreGame(s.Game, rng)
	}
	if a.g == nil {
		a.g = newGame(rng)
	}
	a.best = max(a.best, a.g.score)
	return a
}

func (a *app) save() {
	if a.store == nil {
		return
	}
	s := saved{Best: a.best}
	if !a.g.over {
		s.Game = a.g.save()
	}
	if err := a.store.save(s); err != nil {
		log.Printf("saving the game: %v", err)
	}
}

func (a *app) move(d direction) {
	moved, gained := a.g.move(d)
	if !moved {
		return
	}
	if gained > 0 {
		a.rise++
		a.risen = gained
	}
	a.best = max(a.best, a.g.score)
	a.save()
}

func (a *app) restart() {
	a.g = newGame(a.rng)
	a.risen = 0
	a.save()
}

func (a *app) keepGoing() {
	a.g.keepPlaying = true
	a.save()
}

// keys move the tiles: the arrows, WASD, and Vim's HJKL.
var keys = []struct {
	key ui.Key
	dir direction
}{
	{ui.KeyUp, up}, {ui.KeyRight, right}, {ui.KeyDown, down}, {ui.KeyLeft, left},
	{ui.KeyW, up}, {ui.KeyD, right}, {ui.KeyS, down}, {ui.KeyA, left},
	{ui.KeyK, up}, {ui.KeyL, right}, {ui.KeyJ, down}, {ui.KeyH, left},
}

func (a *app) view(c *ui.Context) {
	for _, k := range keys {
		c.OnShortcut(0, k.key, func() { a.move(k.dir) })
	}
	c.OnShortcut(0, ui.KeyR, a.restart)
	c.OnShortcut(ui.Cmd, ui.KeyN, a.restart)

	a.announce(c)

	w, h := c.Size()
	s := min((w-32)/pageW, (h-32)/pageH)
	s = max(0.45, min(s, 1.6))

	ui.Column(c).Fill().Background(pageBg).Center().Children(func() {
		ui.Column(c).Width(pageW * s).Gap(20 * s).Children(func() {
			a.heading(c, s)
			a.intro(c, s)
			a.board(c, s)
			a.howTo(c, s)
		})
	})
}

// announce tells screen readers once that the game ended.
func (a *app) announce(c *ui.Context) {
	news := ""
	switch {
	case a.g.over:
		news = "Game over! You scored " + strconv.Itoa(a.g.score) + "."
	case a.g.terminated():
		news = "You win! You made the 2048 tile."
	}
	if news != a.announced {
		a.announced = news
		c.Announce(news)
	}
}

func (a *app) heading(c *ui.Context, s float32) {
	ui.Row(c).Justify(ui.SpaceBetween).AlignItems(ui.Center).Children(func() {
		ui.Text(c, "2048").FontSize(80 * s).FontWeight(700).TextColor(ink).NoWrap()
		ui.Row(c).Gap(5 * s).Children(func() {
			a.score(c, s, "SCORE", a.g.score, true)
			a.score(c, s, "BEST", a.best, false)
		})
	})
}

func (a *app) score(c *ui.Context, s float32, label string, n int, rises bool) {
	box := ui.Column(c.Key(label)).AlignItems(ui.Center).MinWidth(70*s).
		Padding(9*s, 25*s, 7*s).Radius(3 * s).Background(boardBg)
	box.Children(func() {
		ui.Text(c, label).FontSize(13 * s).FontWeight(700).TextColor(labelInk).NoWrap()
		ui.Text(c, strconv.Itoa(n)).FontSize(25 * s).FontWeight(700).TextColor(white).NoWrap().
			Label(label + " " + strconv.Itoa(n))
		if rises && a.risen > 0 {
			// Each move's points rise from the score and fade away.
			el := ui.Column(c.Key(riseKey(a.rise))).Absolute().Left(0).Right(0).Top(18 * s).
				AlignItems(ui.Center).PassThrough()
			p := el.Keyframes("rise", rise)
			el.Translate(0, -p*75*s).Opacity(1 - p)
			el.Children(func() {
				ui.Text(c, "+"+strconv.Itoa(a.risen)).FontSize(25 * s).FontWeight(700).TextColor(riseInk).NoWrap()
			})
		}
	})
}

func (a *app) intro(c *ui.Context, s float32) {
	ui.Row(c).Justify(ui.SpaceBetween).AlignItems(ui.Center).Gap(16 * s).Children(func() {
		ui.RichText(c,
			ui.Span{Text: "Join the numbers and get to the "},
			ui.Span{Text: "2048 tile!", Weight: 700},
		).FontSize(18 * s).TextColor(ink).Shrink(1)
		if button(c, s, "New Game").Clicked() {
			a.restart()
		}
	})
}

func (a *app) howTo(c *ui.Context, s float32) {
	ui.RichText(c,
		ui.Span{Text: "How to play: ", Weight: 700},
		ui.Span{Text: "Use the arrow keys, or drag, to move the tiles. When two tiles with the same number touch, they "},
		ui.Span{Text: "merge into one!", Weight: 700},
	).FontSize(18 * s).LineHeight(1.5).TextColor(ink)
}

// button is a button in the look of the game.
func button(c *ui.Context, s float32, label string) ui.Element {
	b := ui.ButtonBase(c.Key(label)).Label(label).Height(40 * s).PaddingX(20 * s).Radius(3 * s)
	bg := buttonBg
	if b.Pressed() {
		bg = bg.Mix(black, 0.08)
	} else if b.Hovered() {
		bg = bg.Mix(white, 0.06)
	}
	b.Background(bg).Cursor(ui.CursorPointer).Children(func() {
		ui.Text(c, label).FontSize(18 * s).FontWeight(700).TextColor(lightInk).NoWrap()
	})
	return b
}

func (a *app) board(c *ui.Context, s float32) {
	b := ui.Box(c.Key("board")).Size(boardW*s, boardW*s).Radius(6 * s).Background(boardBg).
		Label("Board").HandleInput(func(ev ui.InputEvent) bool { return a.swipe(ev, s) })
	b.Children(func() {
		for i := range size * size {
			ui.Box(c.Key(cellKey(i))).Absolute().
				Left((gap+float32(i%size)*cellGap)*s).Top((gap+float32(i/size)*cellGap)*s).
				Size(cellW*s, cellW*s).Radius(3 * s).Background(cellBg)
		}
		// The tiles merged away slide under the tiles they made.
		for _, t := range a.g.ghosts {
			a.tile(c, t, s)
		}
		for _, t := range a.g.tiles {
			a.tile(c, t, s)
		}
		if a.g.terminated() {
			a.message(c, s)
		}
	})
}

func (a *app) tile(c *ui.Context, t *tile, s float32) {
	el := ui.Column(c.Key(tileKey(t.ID))).Absolute().Left(gap*s).Top(gap*s).
		Size(cellW*s, cellW*s).Radius(3 * s).Center()
	// Positions animate in cells, so resizing the window does not.
	col := el.AnimateTo("col", float32(t.Col), slide)
	row := el.AnimateTo("row", float32(t.Row), slide)
	frames := appear
	if t.fromMerge {
		frames = pop
	}
	k := el.Keyframes("scale", frames, ui.KeyframeOptions{Delay: popDelay})
	el.Scale(k, k).Translate(col*cellGap*s, row*cellGap*s)
	if !t.fromMerge {
		el.Opacity(min(k, 1))
	}

	bg, fg := superTile, lightInk
	if v, ok := tileColors[t.Value]; ok {
		bg = v
		if t.Value <= 4 {
			fg = ink
		}
	}
	el.Background(bg)
	if glow, ok := tileGlow[t.Value]; ok {
		el.Shadow(0, 0, 30*s, 10*s, ui.RGBA(243, 215, 116, glow[0])).
			InsetShadow(0, 0, 0, 1*s, ui.RGBA(255, 255, 255, glow[1]))
	}
	el.Children(func() {
		label := strconv.Itoa(t.Value)
		ui.Text(c, label).FontSize(tileFontSize(len(label)) * s).FontWeight(700).TextColor(fg).NoWrap()
	})
}

// tileFontSize returns the size of a number of n digits on a tile.
func tileFontSize(n int) float32 {
	switch {
	case n <= 2:
		return 55
	case n == 3:
		return 45
	case n == 4:
		return 35
	case n == 5:
		return 30
	default:
		return 24
	}
}

// message covers a finished game: lost, or won for the player to keep
// going or start again.
func (a *app) message(c *ui.Context, s float32) {
	text, bg, fg := "Game over!", overBg, ink
	if !a.g.over {
		text, bg, fg = "You win!", wonBg, lightInk
	}
	m := ui.Column(c.Key("message")).Absolute().Left(0).Top(0).Size(boardW*s, boardW*s).
		Radius(6 * s).Background(bg).Center().Gap(40 * s)
	m.Opacity(m.Keyframes("in", messageIn, ui.KeyframeOptions{Delay: messageLag}))
	m.Children(func() {
		ui.Text(c, text).FontSize(60 * s).FontWeight(700).TextColor(fg).NoWrap()
		ui.Row(c).Gap(9 * s).Children(func() {
			if !a.g.over && button(c, s, "Keep going").Clicked() {
				a.keepGoing()
			}
			if button(c, s, "Try again").Clicked() {
				a.restart()
			}
		})
	})
}

// swipe moves the tiles as the pointer drags across the board, or as two
// fingers swipe across a touchpad over it.
func (a *app) swipe(ev ui.InputEvent, s float32) bool {
	if a.g.terminated() {
		a.pressing = false
		return false // the message's buttons take the pointer
	}
	switch ev.Kind {
	case ui.InputPointerDown:
		if ev.Button != 0 {
			return false
		}
		a.pressing, a.pressX, a.pressY = true, ev.X, ev.Y
		return true
	case ui.InputPointerUp:
		if !a.pressing {
			return false
		}
		a.pressing = false
		if d, ok := swipeDirection(ev.X-a.pressX, ev.Y-a.pressY, dragMin*s); ok {
			a.move(d)
		}
		return true
	case ui.InputScroll:
		if !ev.Precise {
			return false // a mouse wheel, by lines
		}
		now := a.clock()
		if now.Sub(a.scrollLast) > scrollEnd {
			a.scrollX, a.scrollY, a.scrollSpent = 0, 0, false
		}
		a.scrollLast = now
		if a.scrollSpent {
			return true
		}
		// The tiles go the way the content of a page would scroll.
		a.scrollX -= ev.DX
		a.scrollY -= ev.DY
		if d, ok := swipeDirection(a.scrollX, a.scrollY, scrollMin); ok {
			a.scrollSpent = true
			a.move(d)
		}
		return true
	}
	return false
}

// swipeDirection returns the direction of a swipe by dx, dy, if it went
// at least min along its main axis.
func swipeDirection(dx, dy, min float32) (direction, bool) {
	ax, ay := abs(dx), abs(dy)
	switch {
	case max(ax, ay) < min:
		return 0, false
	case ax > ay && dx > 0:
		return right, true
	case ax > ay:
		return left, true
	case dy > 0:
		return down, true
	default:
		return up, true
	}
}

func abs(x float32) float32 {
	if x < 0 {
		return -x
	}
	return x
}

func main() {
	var st *store
	if dir, err := mygo.App.Path(mygo.PathUserData); err == nil {
		st = &store{path: filepath.Join(dir, "state.json")}
	} else {
		log.Printf("no place to save the game: %v", err)
	}
	a := newApp(st, rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64())))
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:     "2048",
			Width:     560,
			Height:    840,
			MinWidth:  320,
			MinHeight: 500,
			// Opens where the user left it last time.
			StateKey: "main",
			Content:  ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
