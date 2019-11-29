package main

import (
	"bufio"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/nsf/termbox-go"
)

const (
	_barWidth = 10
	_height   = 40
	_width    = 120
)

type point struct {
	X int
	Y int
}

var mu sync.Mutex

//ステータス
type state struct {
	BarX      int
	End       bool
	Invaders  map[int]*invader
	Life      int
	Score     int
	HighScore int
}

type invader struct {
	Forms    []string
	Rows     int
	Cols     int
	Color    termbox.Attribute
	Pos      point
	Vec      point
	Interval int
}

//タイマーイベント
func moveLoop(moveCh chan int, mover, ticker int) {
	t := time.NewTicker(time.Duration(ticker) * time.Millisecond)
	for {
		select {
		case <-t.C: //タイマーイベント
			moveCh <- mover
			break
		}
	}
	t.Stop()
}

//キーイベント
func keyEventLoop(kch chan termbox.Key) {
	for {
		switch ev := termbox.PollEvent(); ev.Type {
		case termbox.EventKey:
			kch <- ev.Key
		default:
		}
	}
}

//画面描画
func drawLoop(sch chan state) {
	for {
		st := <-sch
		mu.Lock()
		termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)
		for i := 0; i < _width; i++ {
			drawLine(i, 0, "-")
			drawLine(i, _height, "-")
		}
		for i := 0; i < _height; i++ {
			drawLine(0, i, "|")
			drawLine(_width, i, "|")
		}
		drawLine(st.BarX, _height-2, "-========-")
		if st.End == false {
			for k, _ := range st.Invaders {
				drawInvader(*st.Invaders[k])
			}
		} else {
			drawLine(0, _height/2, "|                                PUSH SPACE KEY")
		}
		termbox.Flush()
		mu.Unlock()
	}
}

//行を描画
func drawLine(x, y int, str string) {
	runes := []rune(str)
	for i := 0; i < len(runes); i++ {
		termbox.SetCell(x+i, y, runes[i], termbox.ColorDefault, termbox.ColorDefault)
	}
}

//インベーダーを描画
func drawInvader(invader invader) {
	formIdx := invader.Pos.X / 10 % 2
	forms := invader.Forms
	form := forms[formIdx]
	scanner := bufio.NewScanner(strings.NewReader(form))
	j := 0
	for scanner.Scan() {
		line := scanner.Text()
		runes := []rune(line)
		for i := 0; i < len(runes); i++ {
			termbox.SetCell(invader.Pos.X+i, invader.Pos.Y+j, runes[i], invader.Color, termbox.ColorDefault)
		}
		j++
	}
}

func choice(len int) int {
	rand.Seed(time.Now().UnixNano())
	i := rand.Intn(len)
	return i
}

//ゲームメイン処理
func controller(st state, stateCh chan state, keyCh chan termbox.Key, moveCh chan int) {
	for {
		select {
		case key := <-keyCh: //キーイベント
			mu.Lock()
			switch key {
			case termbox.KeyEsc, termbox.KeyCtrlC: //ゲーム終了
				st.End = true
				mu.Unlock()
				return
			case termbox.KeyArrowLeft: //ひだり
				if st.BarX-3 > 0 {
					st.BarX -= 3
				}
				break
			case termbox.KeyArrowRight: //みぎ
				if st.BarX+_barWidth+3 < _width {
					st.BarX += 3
				}
				break
			case termbox.KeySpace, termbox.KeyEnter: //ゲームスタート
				st.End = false
				break
			}
			mu.Unlock()
			stateCh <- st
			break
		case i := <-moveCh: //タイマーイベント
			mu.Lock()
			if st.End == false {
				if _, ok := st.Invaders[i]; ok {
					st.Invaders[i].Pos.X += st.Invaders[i].Vec.X
					st.Invaders[i].Pos.Y += st.Invaders[i].Vec.Y
					st = checkCollision(st, i)
				}
			}
			mu.Unlock()
			stateCh <- st
			break
		}
	}
}

//初期化
func initGame() state {
	st := state{End: true}
	st.BarX = _width/2 - _barWidth/2
	st.Life = 3
	st.Invaders = initInvaders()

	return st
}

func initInvaders() map[int]*invader {
	var colors = []termbox.Attribute{
		termbox.ColorRed,
		termbox.ColorGreen,
		termbox.ColorYellow,
		termbox.ColorBlue,
		termbox.ColorMagenta,
		termbox.ColorCyan,
		termbox.ColorWhite,
	}
	form1 := strings.TrimLeft(`
 ▚▄▄▞ 
▙█▟▙█▟
 ▞  ▚ `, "\n")
	form2 := strings.TrimLeft(`
 ▚▄▄▞ 
▟█▟▙█▙
▘▝▖▗▘▝`, "\n")
	plusminus := []int{-1, 1}
	invaders := map[int]*invader{}
	rows := strings.Count(form1, "\n") + 1
	cols := 6
	for i := 0; i < 20; i++ {
		invaders[i] = &invader{
			Forms:    []string{form1, form2},
			Rows:     rows,
			Cols:     cols,
			Color:    colors[choice(len(colors))],
			Pos:      point{X: choice(_width-cols) + 1, Y: choice(_height-rows) + 1},
			Vec:      point{X: plusminus[choice(2)], Y: plusminus[choice(2)]},
			Interval: ((i % 5) + 1) * 50,
		}
	}
	return invaders
}

//衝突判定
func checkCollision(st state, i int) state {
	//バーが右の壁に到達
	if st.BarX+_barWidth > _width {
		st.BarX -= 3
	}

	//左の壁
	if st.Invaders[i].Pos.X <= 0 {
		st.Invaders[i].Pos.X = 1
		st.Invaders[i].Vec.X = 1
	}
	//右の壁
	if st.Invaders[i].Pos.X+st.Invaders[i].Cols >= _width {
		st.Invaders[i].Pos.X = _width - st.Invaders[i].Cols
		st.Invaders[i].Vec.X = -1
	}
	//上の壁
	if st.Invaders[i].Pos.Y <= 0 {
		st.Invaders[i].Pos.Y = 1
		st.Invaders[i].Vec.Y = 1
	}
	//下の壁
	if st.Invaders[i].Pos.Y+st.Invaders[i].Rows >= _height {
		st.Invaders[i].Pos.Y = _height - st.Invaders[i].Rows
		st.Invaders[i].Vec.Y = -1
	}
	//インベーダー同士
	for o, _ := range st.Invaders {
		//左
		if st.Invaders[i].Vec.X < 0 && st.Invaders[i].Pos.X == st.Invaders[o].Pos.X+st.Invaders[o].Cols {
			if st.Invaders[i].Pos.Y <= st.Invaders[o].Pos.Y+st.Invaders[o].Rows &&
				st.Invaders[i].Pos.Y+st.Invaders[i].Rows >= st.Invaders[o].Pos.Y {
				if st.Invaders[i].Vec.X != st.Invaders[o].Vec.X {
					st.Invaders[i].Vec.X *= -1
					st.Invaders[o].Vec.X *= -1
				} else {
					st.Invaders[i].Vec.X *= -1
				}
			}
		}
		//右
		if st.Invaders[i].Vec.X > 0 && st.Invaders[i].Pos.X+st.Invaders[i].Cols == st.Invaders[o].Pos.X {
			if st.Invaders[i].Pos.Y <= st.Invaders[o].Pos.Y+st.Invaders[o].Rows &&
				st.Invaders[i].Pos.Y+st.Invaders[i].Rows >= st.Invaders[o].Pos.Y {
				if st.Invaders[i].Vec.X != st.Invaders[o].Vec.X {
					st.Invaders[i].Vec.X *= -1
					st.Invaders[o].Vec.X *= -1
				} else {
					st.Invaders[i].Vec.X *= -1
				}
			}
		}
		//上
		if st.Invaders[i].Vec.Y < 0 && st.Invaders[i].Pos.Y == st.Invaders[o].Pos.Y+st.Invaders[o].Rows {
			if st.Invaders[i].Pos.X <= st.Invaders[o].Pos.X+st.Invaders[o].Cols &&
				st.Invaders[i].Pos.X+st.Invaders[i].Cols >= st.Invaders[o].Pos.X {
				if st.Invaders[i].Vec.Y != st.Invaders[o].Vec.Y {
					st.Invaders[i].Vec.Y *= -1
					st.Invaders[o].Vec.Y *= -1
				} else {
					st.Invaders[i].Vec.Y *= -1
				}
			}
		}
		//下
		if st.Invaders[i].Vec.Y > 0 && st.Invaders[i].Pos.Y+st.Invaders[i].Rows == st.Invaders[o].Pos.Y {
			if st.Invaders[i].Pos.X <= st.Invaders[o].Pos.X+st.Invaders[o].Cols &&
				st.Invaders[i].Pos.X+st.Invaders[i].Cols >= st.Invaders[o].Pos.X {
				if st.Invaders[i].Vec.Y != st.Invaders[o].Vec.Y {
					st.Invaders[i].Vec.Y *= -1
					st.Invaders[o].Vec.Y *= -1
				} else {
					st.Invaders[i].Vec.Y *= -1
				}
			}
		}
	}

	//バーとの衝突判定
	if st.Invaders[i].Pos.X+st.Invaders[i].Cols >= st.BarX && st.Invaders[i].Pos.X <= st.BarX+_barWidth &&
		st.Invaders[i].Pos.Y == _height-2-st.Invaders[i].Rows {
		delete(st.Invaders, i)
	}

	//インベーダー全撃破
	if len(st.Invaders) == 0 {
		st.Invaders = initInvaders()
	}

	return st
}

//main
func main() {
	err := termbox.Init()
	if err != nil {
		panic(err)
	}
	st := initGame()
	stateCh := make(chan state)
	keyCh := make(chan termbox.Key)
	moveCh := make(chan int)

	go drawLoop(stateCh)
	go keyEventLoop(keyCh)
	for k, v := range st.Invaders {
		go func(idx, ticker int) {
			moveLoop(moveCh, idx, ticker)
		}(k, v.Interval)
	}

	controller(st, stateCh, keyCh, moveCh)
	termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)

	defer termbox.Close()
}
