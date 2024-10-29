//go:build js && wasm

package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"syscall/js"
	"time"
)

const (
	ColorGrey       = "#7f7f7f"
	ColorYellow     = "#ff0"
	ColorYellowDark = "#aa0"
	ColorBlue       = "#00f"
	ColorBlueDark   = "#00a"
	ColorRed        = "#f00"
	ColorRedDark    = "#a00"
	ColorGreen      = "#0f0"
	ColorGreenDark  = "#0a0"
	ColorOrange     = "#f70"
	ColorOrangeDark = "#d50"
	ColorPink       = "#f0f"
	ColorPinkDark   = "#a0a"
	ColorPurple     = "#f07"
	ColorPurpleDark = "#d05"
)

var LShapeMask = [][]bool{
	{true, true},
	{true, false},
	{true, false},
}

var LShapeMask2 = [][]bool{
	{true, true},
	{false, true},
	{false, true},
}

var IShapeMask = [][]bool{
	{true},
	{true},
	{true},
	{true},
}

var OShapeMask = [][]bool{
	{true, true},
	{true, true},
}

var PShapeMask = [][]bool{
	{true, false},
	{true, true},
	{true, false},
}

var SShapeMask = [][]bool{
	{false, true},
	{true, true},
	{true, false},
}

var SShapeMask2 = [][]bool{
	{true, false},
	{true, true},
	{false, true},
}

var game = struct {
	Grid             [20][10]string
	CurrentPosition  [2]int
	CurrentShapeMask [][]bool
	CurrentShape     int
	NextShape        int
	Score            int
	Paused           bool
}{
	NextShape: rand.Intn(7),
	Paused:    true,
}

func main() {
	fmt.Println("Hi from littie :0")
	gameGrid := GameGrid()
	document := Document()
	for range 20 {
		gameRow := document.Call("createElement", "game-row")
		for range 10 {
			gameSquare := document.Call("createElement", "game-square")
			gameRow.Call("appendChild", gameSquare)
		}
		gameGrid.Call("appendChild", gameRow)
	}

	QuerySelector("game-menu #play").Call("addEventListener", "click", js.FuncOf(MenuButtonHandler))
	QuerySelector("game-menu #restart").Call("addEventListener", "click", js.FuncOf(MenuButtonHandler))
	QuerySelector("game-menu #debug").Call("addEventListener", "click", js.FuncOf(MenuButtonHandler))
	QuerySelector("game-menu #back").Call("addEventListener", "click", js.FuncOf(MenuButtonHandler))
	QuerySelector("game-menu #settings").Call("addEventListener", "click", js.FuncOf(MenuButtonHandler))
	QuerySelector("game-menu #github").Call("addEventListener", "click", js.FuncOf(MenuButtonHandler))

	NextShape()

	document.Call("addEventListener", "keyup", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		event := args[0]
		key := event.Get("code").String()
		keyAction(key)
		return nil
	}))

	go func() {
		for {
			RenderGrid()
			RenderActiveShape()
			time.Sleep(100 * time.Millisecond)
		}
	}()

	for { // Move Tick
		time.Sleep(1 * time.Second)
		MoveTick()
	}

	//<-make(chan struct{})
}

func Document() js.Value {
	return js.Global().Get("document")
}

func QuerySelector(elementQuery string) js.Value {
	return Document().Call("querySelector", elementQuery)
}

func GameGrid() js.Value {
	return QuerySelector("game-grid")
}

func MenuButtonHandler(this js.Value, args []js.Value) interface{} {
	id := this.Get("id").String()
	switch id {
	case "play":
		ToggleMenu()
	case "restart":
		Restart()
	case "debug":
		byt, err := json.Marshal(game)
		if err != nil {
			byt = []byte("Parse err!")
		}
		QuerySelector("#debug-info").Set("innerHTML", string(byt))
		ShowMenuPage(1)
	case "back":
		ShowMenuPage(0)
	case "github":
		js.Global().Get("window").Call("open", "https://github.com/littie6amer/block-stacking")
	}
	return nil
}

func ToggleMenu() {
	menu := QuerySelector("game-menu")
	class := menu.Get("className").String()
	if class == "hide" {
		ShowMenuPage(0)
		menu.Set("className", "")
		game.Paused = true
	} else {
		menu.Set("className", "hide")
		game.Paused = false
	}
}

func ShowMenuPage(page int) {
	mainMenu, debug := QuerySelector("game-menu #main-menu"), QuerySelector("game-menu #debug-menu")
	switch page {
	case 0:
		mainMenu.Set("className", "")
		debug.Set("className", "hide")
	case 1:
		mainMenu.Set("className", "hide")
		debug.Set("className", "")
	}
}

func keyAction(key string) {
	if game.Paused && key != "Escape" {
		return
	}

	switch key {
	case "Escape":
		ToggleMenu()
	case "KeyA", "ArrowLeft":
		{
			valid := IsValidPosition([2]int{game.CurrentPosition[0] - 1, game.CurrentPosition[1]})
			//fmt.Println(valid)
			if !valid {
				return
			}
			game.CurrentPosition[0]--
		}
	case "KeyD", "ArrowRight":
		{
			valid := IsValidPosition([2]int{game.CurrentPosition[0] + 1, game.CurrentPosition[1]})
			//fmt.Println(valid)
			if !valid {
				return
			}
			game.CurrentPosition[0]++
		}
	case "KeyS", "ArrowDown":
		MoveTick()
	case "KeyW", "KeyR", "ArrowUp":
		game.CurrentShapeMask = RotateShape(game.CurrentShapeMask)
	case "KeyN":
		NextShape()
	case "Enter":
		PlaceShape()
	}
}

func MoveTick() {
	if game.Paused {
		return
	}

	yPlace := YPlacePosition()
	if yPlace == game.CurrentPosition[1]+1 {
		PlaceShape()
	} else {
		game.CurrentPosition[1]++
	}
}

func IsValidPosition(position [2]int) bool {
	shape := game.CurrentShapeMask
	x, y := position[0], position[1]

	if y == 0 {
		return true
	}

	shapeCols := len(game.CurrentShapeMask[0])
	shapeRows := len(game.CurrentShapeMask)

	for r := range shapeRows {
		for c := range shapeCols {
			if x+c > 9 || x+c < 0 {
				return false
			}
			if shape[r][c] && game.Grid[y+r][x+c] != "" {
				return false
			}
		}
	}
	return true
}

func RotateShape(shape [][]bool) [][]bool {
	rows := len(shape)
	cols := len(shape[0])

	rotated := make([][]bool, cols)
	for i := range rotated {
		rotated[i] = make([]bool, rows)
	}

	for r := range rows {
		for c := range cols {
			rotated[c][rows-1-r] = shape[r][c]
		}
	}
	return rotated
}

func YPlacePosition() int {
	shape := game.CurrentShapeMask
	shapeCols := len(game.CurrentShapeMask[0])
	shapeRows := len(game.CurrentShapeMask)
	yFloor := 21 - shapeRows
	highRow := 0

	for r := range shapeRows {
		for c := range shapeCols {
			for actualY := game.CurrentPosition[1]; actualY < 20; actualY++ {
				if !shape[r][c] || actualY <= game.CurrentPosition[1]+r || actualY >= yFloor || (r != shapeRows-1 && shape[r+1][c]) {
					continue
				}
				if actualY == 19 || game.Grid[actualY+1][game.CurrentPosition[0]+c] != "" {
					yFloor = (actualY + 1) //- (shapeRows - (r + 1))
					highRow = r
					break
				}
			}
		}
	}

	//for tryY := range 19 - game.CurrentPosition[1] /*0-19*/ {
	//	actualY := 19 - tryY // 0-19
	//
	//}

	//return yFloor
	return yFloor - (highRow + 1)
}

func Restart() {
	game.Paused = true
	game.Grid = [20][10]string{}
	game.Score = 0
	NextShape()
}

func PlaceShape() {
	shape := game.CurrentShapeMask
	shapeCols := len(game.CurrentShapeMask[0])
	shapeRows := len(game.CurrentShapeMask)

	yPlace := YPlacePosition()

	for r := range shapeRows {
		for c := range shapeCols {
			if shape[r][c] {
				game.Grid[yPlace+r][game.CurrentPosition[0]+c] = []string{ColorOrange, ColorPink, ColorYellow, ColorBlue, ColorPurple, ColorGreen, ColorRed}[game.CurrentShape]
			}
		}
	}
	CheckRows()
	NextShape()
}

func NextShape() {
	switch game.NextShape {
	case 0:
		game.CurrentShapeMask = LShapeMask
	case 1:
		game.CurrentShapeMask = LShapeMask2
	case 2:
		game.CurrentShapeMask = OShapeMask
	case 3:
		game.CurrentShapeMask = IShapeMask
	case 4:
		game.CurrentShapeMask = PShapeMask
	case 5:
		game.CurrentShapeMask = SShapeMask
	case 6:
		game.CurrentShapeMask = SShapeMask2
	}
	game.CurrentPosition = [2]int{4, 0}
	game.CurrentShape = game.NextShape
	fmt.Println("Switched to next shape:", game.NextShape)
	game.NextShape = rand.Intn(7)
}

func SetSquareColor(row, col int, hexColor string) {
	square := QuerySelector(fmt.Sprintf("game-grid :nth-child(%d) :nth-child(%d)", row+1, col+1))
	if square.Equal(js.Null()) {
		return
	}
	//fmt.Println(fmt.Sprintf("Set %d, %d to %s", row, col, hexColor))
	style := square.Get("style")
	style.Set("background-color", hexColor)
}

func CheckRows() {
	i := 19
	var newGrid [20][10]string
	for r := range 20 {
		r = 19 - r
		solved := true
		for c := range 10 {
			if game.Grid[r][c] == "" {
				solved = false
			} else if r == 1 {
				QuerySelector("#status").Set("innerHTML", fmt.Sprintf("You lost :(<br>Your last score was %d", game.Score))
				Restart()
				ToggleMenu()
				return
			}
		}
		if solved {
			game.Score++
			for c := range 10 {
				SetSquareColor(r, c, "#fff")
			}
		} else {
			newGrid[i] = game.Grid[r]
			i--
		}
	}
	game.Grid = newGrid
}

func RenderGrid() {
	for r := range 20 {
		for c := range 10 {
			color := game.Grid[r][c]
			if color == "" {
				color = ColorGrey
			}

			SetSquareColor(r, c, color)
		}
	}
}

func RenderActiveShape() {
	position := &game.CurrentPosition
	shape := game.CurrentShapeMask
	x, y := position[0], position[1]
	shapeCols := len(game.CurrentShapeMask[0])
	shapeRows := len(game.CurrentShapeMask)

	if x > 10-shapeCols {
		position[0] = 0
		x = 0
	}
	if x < 0 {
		position[0] = 10 - shapeCols
		x = 10 - shapeCols
	}

	for r := range shapeRows {
		for c := range shapeCols {
			if shape[r][c] == true {
				SetSquareColor(y+r, x+c, []string{ColorOrangeDark, ColorPinkDark, ColorYellowDark, ColorBlueDark, ColorPurpleDark, ColorGreenDark, ColorRedDark}[game.CurrentShape])
			}
		}
	}
}
