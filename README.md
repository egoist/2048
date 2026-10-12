# 2048

The classic 2048 puzzle as a native desktop app, built with [MyGo](https://github.com/egoist/mygo).

## Run locally

You need Go 1.27.1 or newer. The MyGo CLI is pinned in `go.mod` as a tool, so there is nothing else to install.

```sh
go tool mygo dev     # run with live reload
go test ./...        # run the tests
go tool mygo build   # package the app into build/
```

## How to play

Move the tiles with the arrow keys, WASD or HJKL, by dragging across the board, or with a two-finger swipe on a touchpad. When two tiles with the same number touch, they merge into one. Reach the 2048 tile to win.

Press R or ⌘N to start a new game. Your game and best score are saved between launches.
