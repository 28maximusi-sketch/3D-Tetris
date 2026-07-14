// tetris3d.go - 3D Тетрис на Go
package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/eiannone/keyboard"
)

const (
	WIDTH  = 6
	DEPTH  = 6
	HEIGHT = 6
	EMPTY  = 0
	RECORD_FILE = "tetris3d_record.json"
)

var SHAPES = map[string][][3]int{
	"I": {{0,0,0},{1,0,0},{2,0,0},{3,0,0}},
	"O": {{0,0,0},{1,0,0},{0,1,0},{1,1,0}},
	"T": {{0,0,0},{1,0,0},{2,0,0},{1,1,0}},
	"L": {{0,0,0},{1,0,0},{2,0,0},{2,1,0}},
	"J": {{0,0,0},{1,0,0},{2,0,0},{0,1,0}},
	"S": {{1,0,0},{2,0,0},{0,1,0},{1,1,0}},
	"Z": {{0,0,0},{1,0,0},{1,1,0},{2,1,0}},
}

type Tetris3D struct {
	field        [][][]int
	score        int
	speed        int
	gameOver     bool
	paused       bool
	running      bool
	record       int
	currentShape [][3]int
	shapePos     [3]int // x,z,y
	dropTimer    time.Time
	lockInput    bool
	rand         *rand.Rand
}

func NewTetris3D() *Tetris3D {
	t := &Tetris3D{
		field:    make([][][]int, HEIGHT),
		score:    0,
		speed:    1,
		gameOver: false,
		paused:   false,
		running:  true,
		record:   loadRecord(),
		rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}
	for y := 0; y < HEIGHT; y++ {
		t.field[y] = make([][]int, WIDTH)
		for x := 0; x < WIDTH; x++ {
			t.field[y][x] = make([]int, DEPTH)
		}
	}
	t.spawnShape()
	return t
}

func loadRecord() int {
	file, err := os.Open(RECORD_FILE)
	if err != nil {
		return 0
	}
	defer file.Close()
	var data map[string]int
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&data); err != nil {
		return 0
	}
	if val, ok := data["record"]; ok {
		return val
	}
	return 0
}

func saveRecord(record int) {
	data := map[string]int{"record": record}
	file, _ := os.Create(RECORD_FILE)
	defer file.Close()
	encoder := json.NewEncoder(file)
	encoder.Encode(data)
}

func (t *Tetris3D) randomShape() [][3]int {
	keys := make([]string, 0, len(SHAPES))
	for k := range SHAPES {
		keys = append(keys, k)
	}
	name := keys[t.rand.Intn(len(keys))]
	return SHAPES[name]
}

func (t *Tetris3D) spawnShape() bool {
	shape := t.randomShape()
	xs := make([]int, len(shape))
	zs := make([]int, len(shape))
	for i, p := range shape {
		xs[i] = p[0]
		zs[i] = p[1]
	}
	cx := (max(xs) + min(xs)) / 2
	cz := (max(zs) + min(zs)) / 2
	startX := WIDTH/2 - cx
	startZ := DEPTH/2 - cz
	startY := HEIGHT - 1
	for _, p := range shape {
		x := startX + p[0]
		z := startZ + p[1]
		y := startY + p[2]
		if x < 0 || x >= WIDTH || z < 0 || z >= DEPTH || y < 0 || y >= HEIGHT || t.field[y][x][z] != EMPTY {
			t.gameOver = true
			if t.score > t.record {
				t.record = t.score
				saveRecord(t.record)
			}
			return false
		}
	}
	t.currentShape = shape
	t.shapePos = [3]int{startX, startZ, startY}
	return true
}

func (t *Tetris3D) collides(shape [][3]int, pos [3]int) bool {
	for _, p := range shape {
		x := pos[0] + p[0]
		z := pos[1] + p[1]
		y := pos[2] + p[2]
		if x < 0 || x >= WIDTH || z < 0 || z >= DEPTH || y < 0 || y >= HEIGHT {
			return true
		}
		if t.field[y][x][z] != EMPTY {
			return true
		}
	}
	return false
}

func (t *Tetris3D) lockShape() {
	for _, p := range t.currentShape {
		x := t.shapePos[0] + p[0]
		z := t.shapePos[1] + p[1]
		y := t.shapePos[2] + p[2]
		t.field[y][x][z] = 1
	}
	cleared := 0
	for y := 0; y < HEIGHT; y++ {
		full := true
		for x := 0; x < WIDTH; x++ {
			for z := 0; z < DEPTH; z++ {
				if t.field[y][x][z] == EMPTY {
					full = false
					break
				}
			}
			if !full {
				break
			}
		}
		if full {
			for y2 := y; y2 > 0; y2-- {
				t.field[y2] = t.field[y2-1]
			}
			t.field[0] = make([][]int, WIDTH)
			for x := 0; x < WIDTH; x++ {
				t.field[0][x] = make([]int, DEPTH)
			}
			cleared++
		}
	}
	if cleared > 0 {
		t.score += cleared
		t.speed = 1 + t.score/3
	}
	if !t.spawnShape() {
		t.gameOver = true
	}
}

func (t *Tetris3D) move(dx, dz, dy int) bool {
	newPos := [3]int{t.shapePos[0] + dx, t.shapePos[1] + dz, t.shapePos[2] + dy}
	if !t.collides(t.currentShape, newPos) {
		t.shapePos = newPos
		return true
	}
	return false
}

func (t *Tetris3D) rotate() {
	newShape := make([][3]int, len(t.currentShape))
	for i, p := range t.currentShape {
		newShape[i] = [3]int{-p[1], p[0], p[2]} // вращение вокруг Y
	}
	if !t.collides(newShape, t.shapePos) {
		t.currentShape = newShape
	}
}

func (t *Tetris3D) drop() {
	for t.move(0, 0, -1) {
	}
	t.lockShape()
}

func (t *Tetris3D) update() {
	if t.gameOver || t.paused {
		return
	}
	if time.Since(t.dropTimer) >= time.Second/time.Duration(t.speed) {
		if !t.move(0, 0, -1) {
			t.lockShape()
		}
		t.dropTimer = time.Now()
	}
}

func (t *Tetris3D) getFieldWithShape() [][][]int {
	copyField := make([][][]int, HEIGHT)
	for y := 0; y < HEIGHT; y++ {
		copyField[y] = make([][]int, WIDTH)
		for x := 0; x < WIDTH; x++ {
			copyField[y][x] = make([]int, DEPTH)
			copy(copyField[y][x], t.field[y][x])
		}
	}
	if t.currentShape != nil {
		for _, p := range t.currentShape {
			x := t.shapePos[0] + p[0]
			z := t.shapePos[1] + p[1]
			y := t.shapePos[2] + p[2]
			if x >= 0 && x < WIDTH && z >= 0 && z < DEPTH && y >= 0 && y < HEIGHT {
				copyField[y][x][z] = 2
			}
		}
	}
	return copyField
}

func (t *Tetris3D) draw() {
	clearScreen()
	fieldCopy := t.getFieldWithShape()
	fmt.Println(stringRepeat("═", WIDTH*2+10))
	fmt.Printf("  Счёт: %d   Рекорд: %d   Скорость: %d\n", t.score, t.record, t.speed)
	fmt.Println(stringRepeat("═", WIDTH*2+10))
	for y := HEIGHT - 1; y >= 0; y-- {
		fmt.Printf("  Y=%d  ", y)
		for z := 0; z < DEPTH; z++ {
			for x := 0; x < WIDTH; x++ {
				val := fieldCopy[y][x][z]
				if val == 2 {
					fmt.Print("\x1b[32m█ \x1b[0m")
				} else if val == 1 {
					fmt.Print("\x1b[36m█ \x1b[0m")
				} else {
					fmt.Print("· ")
				}
			}
			fmt.Print("  ")
		}
		fmt.Println()
	}
	fmt.Println(stringRepeat("═", WIDTH*2+10))
	status := "ИГРА"
	if t.paused {
		status = "ПАУЗА"
	} else if t.gameOver {
		status = "ИГРА ОКОНЧЕНА"
	}
	fmt.Printf("  %s  |  ←/→ - X  |  W/S - Z  |  Пробел - вращение  |  ↓ - падение\n", status)
	fmt.Println("  P - пауза  |  R - рестарт  |  Q - выход")
}

func clearScreen() {
	cmd := exec.Command("clear")
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", "cls")
	}
	cmd.Stdout = os.Stdout
	cmd.Run()
}

func stringRepeat(s string, n int) string {
	res := ""
	for i := 0; i < n; i++ {
		res += s
	}
	return res
}

func min(arr []int) int {
	m := arr[0]
	for _, v := range arr[1:] {
		if v < m {
			m = v
		}
	}
	return m
}

func max(arr []int) int {
	m := arr[0]
	for _, v := range arr[1:] {
		if v > m {
			m = v
		}
	}
	return m
}

func (t *Tetris3D) reset() {
	t.field = make([][][]int, HEIGHT)
	for y := 0; y < HEIGHT; y++ {
		t.field[y] = make([][]int, WIDTH)
		for x := 0; x < WIDTH; x++ {
			t.field[y][x] = make([]int, DEPTH)
		}
	}
	t.score = 0
	t.speed = 1
	t.gameOver = false
	t.paused = false
	t.spawnShape()
}

func (t *Tetris3D) handleInput() {
	for t.running {
		char, key, err := keyboard.GetKey()
		if err != nil {
			continue
		}
		if t.lockInput {
			continue
		}
		t.lockInput = true
		switch key {
		case keyboard.KeyArrowLeft:
			if !t.gameOver && !t.paused {
				t.move(-1, 0, 0)
			}
		case keyboard.KeyArrowRight:
			if !t.gameOver && !t.paused {
				t.move(1, 0, 0)
			}
		case keyboard.KeyArrowDown:
			if !t.gameOver && !t.paused {
				t.drop()
			}
		case keyboard.KeySpace:
			if !t.gameOver && !t.paused {
				t.rotate()
			}
		default:
			switch char {
			case 'w', 'W':
				if !t.gameOver && !t.paused {
					t.move(0, -1, 0)
				}
			case 's', 'S':
				if !t.gameOver && !t.paused {
					t.move(0, 1, 0)
				}
			case 'p', 'P':
				t.paused = !t.paused
			case 'r', 'R':
				t.reset()
			case 'q', 'Q':
				t.running = false
				return
			}
		}
		t.lockInput = false
	}
}

func (t *Tetris3D) run() {
	go t.handleInput()

	t.dropTimer = time.Now()
	ticker := time.NewTicker(time.Second / 30)
	defer ticker.Stop()

	for t.running {
		<-ticker.C
		t.update()
		t.draw()
	}
}

func main() {
	game := NewTetris3D()
	game.run()
	fmt.Println("Игра завершена.")
}
