package main

import (
	"bufio"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/nsf/termbox-go"
)

const (
	_refreshSpan     = 1000
	_fireTimeSpan    = 100
	_invaderTimeSpan = 500
	_barWidth        = 10
	_blockWidth      = 6
	_height          = 25
	_width           = 80
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
	Ball      point
	Vec       point
	Blocks    []point
	Life      int
	Score     int
	HighScore int
}

//タイマーイベント
func fireTimerLoop(ftch chan bool) {
	for {
		ftch <- true
		time.Sleep(time.Duration(_fireTimeSpan) * time.Millisecond)
	}
}

//タイマーイベント
func invaderTimerLoop(itch chan bool) {
	for {
		itch <- true
		time.Sleep(time.Duration(_invaderTimeSpan) * time.Millisecond)
	}
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
		drawLine(1, 0, "EXIT : ESC KEY")
		drawLine(_width-50, 0, fmt.Sprintf("HighScore : %05d", st.HighScore))
		drawLine(_width-30, 0, fmt.Sprintf("Score : %05d", st.Score))
		drawLine(_width-10, 0, fmt.Sprintf("Life : %02d", st.Life))
		drawLine(0, 1, "--------------------------------------------------------------------------------")
		for i := range st.Blocks {
			if st.Blocks[i].Y >= 0 {
				drawLine(st.Blocks[i].X, st.Blocks[i].Y, "======")
			}
		}
		drawLine(st.BarX, _height-2, "-========-")
		if st.End == false {
			drawInvader(st.Ball.X, st.Ball.Y)
		} else {
			drawLine(0, _height/2, "                                  PUSH SPACE KEY")
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
func drawInvader(x, y int) {
	invader := `
 ▚▄▄▞
▟█▟▙█▙
▘▝▖▗▘▝
`
	scanner := bufio.NewScanner(strings.NewReader(invader))
	j := 0
	for scanner.Scan() {
		line := scanner.Text()
		runes := []rune(line)
		for i := 0; i < len(runes); i++ {
			termbox.SetCell(x+i-3, y+j-2, runes[i], termbox.ColorDefault, termbox.ColorDefault)
		}
		j++
	}
}

//ゲームメイン処理
func controller(stateCh chan state, keyCh chan termbox.Key, fireCh, invaderCh chan bool) {
	st := initGame()
	t := time.NewTicker(time.Duration(_refreshSpan) * time.Millisecond)
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
		case <-fireCh: //タイマーイベント
			mu.Lock()
			if st.End == false {
				st.Ball.X += st.Vec.X
				st.Ball.Y += st.Vec.Y
				st = checkCollision(st)
			}
			mu.Unlock()
			stateCh <- st
			break
		case <-t.C: //タイマーイベント
			break
			//default:
			//	break
		}
	}
	t.Stop()
}

//初期化
func initGame() state {
	st := state{End: true}
	st.BarX = _width/2 - _barWidth/2
	st.Ball.X, st.Ball.Y = _width/2, _height*2/3
	st.Vec.X, st.Vec.Y = 1, -1
	st.Life = 3
	st.Blocks = initBlock()
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

//衝突判定
func checkCollision(st state) state {
	//左右の壁
	if st.Ball.X <= 0 || st.Ball.X+3 >= _width {
		st.Vec.X *= -1
	}
	//上下の壁
	if st.Ball.Y <= 2 {
		st.Vec.Y = 1
	}
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
		st.Blocks = initBlock()
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
	stateCh := make(chan state)
	keyCh := make(chan termbox.Key)
	fireCh := make(chan bool)
	invaderCh := make(chan bool)

	go drawLoop(stateCh)
	go keyEventLoop(keyCh)
	go fireTimerLoop(fireCh)
	go invaderTimerLoop(invaderCh)

	controller(stateCh, keyCh, fireCh, invaderCh)
	termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)

	defer termbox.Close()
}
