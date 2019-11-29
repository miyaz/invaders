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
	_barWidth   = 10
	_blockWidth = 6
	_height     = 40
	_width      = 120
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
	Invaders  []invader
	Ball      point
	Vec       point
	Blocks    []point
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
		/*
			for i := range st.Blocks {
				if st.Blocks[i].Y >= 0 {
					drawLine(st.Blocks[i].X, st.Blocks[i].Y, "======")
				}
			}
		*/
		drawLine(st.BarX, _height-2, "-========-")
		if st.End == false {
			//drawInvader(st.Ball.X, st.Ball.Y)
			for i := range st.Invaders {
				drawInvader(st.Invaders[i])
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
	forms := invader.Forms
	form := forms[choice(2)]
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
				st.Ball.X += st.Vec.X
				st.Ball.Y += st.Vec.Y
				st.Invaders[i].Pos.X += st.Invaders[i].Vec.X
				st.Invaders[i].Pos.Y += st.Invaders[i].Vec.Y
				st = checkCollision(st)
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
	st.Ball.X, st.Ball.Y = _width/2, _height*2/3
	st.Vec.X, st.Vec.Y = 1, -1
	st.Life = 3
	//st.Blocks = initBlock()
	st.Invaders = initInvaders()

	return st
}

//ブロック初期化
func initBlock() []point {
	var blocks []point
	for r := 0; r < 5; r++ {
		for c := 0; c < 11; c++ {
			blocks = append(blocks,
				point{X: 2 + c*(_blockWidth+1), Y: 4 + r})
		}
	}

	return blocks
}

func initInvaders() []invader {
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
	invaders := []invader{}
	rows := strings.Count(form1, "\n") + 1
	cols := 6
	for i := 0; i < 20; i++ {
		invader := invader{
			Forms:    []string{form1, form2},
			Rows:     rows,
			Cols:     cols,
			Color:    colors[choice(len(colors))],
			Pos:      point{X: choice(_width - 10), Y: choice(_height - 5)},
			Vec:      point{X: plusminus[choice(2)], Y: plusminus[choice(2)]},
			Interval: ((i % 5) + 1) * 50,
		}
		invaders = append(invaders, invader)
	}
	return invaders
}

//衝突判定
func checkCollision(st state) state {
	//左右の壁
	if st.Ball.X-4 <= 0 || st.Ball.X+3 >= _width {
		st.Vec.X *= -1
	}
	//上下の壁
	if st.Ball.Y <= 2 {
		st.Vec.Y = 1
	}
	if st.Ball.Y >= _height {
		st.Vec.Y = -1
	}
	/*
		//ミス
		if st.Ball.Y >= _height {
			st.Life--
			st.Ball.X, st.Ball.Y = _width/2, _height*2/3
			st.Vec.Y = -1
			if st.Life <= 0 {
				hs := 0
				if st.HighScore < st.Score {
					hs = st.Score
				}
				st = initGame()
				st.HighScore = hs
			}
		}
	*/
	//バーとの衝突判定
	if st.Ball.X >= st.BarX && st.Ball.X <= st.BarX+_barWidth &&
		(st.Ball.Y == _height-2) {
		st.Vec.Y = -1
		if st.Ball.X <= st.BarX+(_barWidth/2) {
			st.Vec.X = -1
		} else {
			st.Vec.X = +1
		}
	}
	//バーが右の壁に到達
	if st.BarX+_barWidth > _width {
		st.BarX -= 3
	}
	//ブロックとの衝突判定
	for i := range st.Blocks {
		if st.Blocks[i].Y == st.Ball.Y {
			if st.Blocks[i].X <= st.Ball.X && st.Blocks[i].X+_blockWidth >= st.Ball.X {
				st.Vec.Y *= -1
				st.Blocks = remove(st.Blocks, i)
				st.Score++
				break
			}
		}
	}
	//ブロック全撃破
	if len(st.Blocks) == 0 {
		//st.Blocks = initBlock()
	}

	for i := range st.Invaders {
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
		//上下の壁
		if st.Invaders[i].Pos.Y <= 0 {
			st.Invaders[i].Pos.Y = 1
			st.Invaders[i].Vec.Y = 1
		}
		if st.Invaders[i].Pos.Y+st.Invaders[i].Rows >= _height {
			st.Invaders[i].Pos.Y = _height - st.Invaders[i].Rows
			st.Invaders[i].Vec.Y = -1
		}
		//バーとの衝突判定
		if st.Invaders[i].Pos.X+st.Invaders[i].Cols >= st.BarX && st.Invaders[i].Pos.X <= st.BarX+_barWidth &&
			st.Invaders[i].Pos.Y == _height-2-st.Invaders[i].Rows {
			st.Invaders[i].Vec.Y = -1
			if st.Invaders[i].Pos.X+(st.Invaders[i].Cols/2) <= st.BarX+(_barWidth/2) {
				st.Invaders[i].Vec.X = -1
			} else {
				st.Invaders[i].Vec.X = +1
			}
		}
		//バーが右の壁に到達
		if st.BarX+_barWidth > _width {
			st.BarX -= 3
		}
	}

	return st
}

//配列消去
func remove(s []point, i int) []point {
	s = s[:i+copy(s[i:], s[i+1:])]
	return s
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
