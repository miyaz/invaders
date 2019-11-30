package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/nsf/termbox-go"
)

const (
	_height = 40
	_width  = 120
)

type point struct {
	X int
	Y int
}

var mu sync.Mutex

//ステータス
type state struct {
	End       bool
	Player    *player
	Invaders  map[int]*invader
	Bullets   map[int]*bullet
	Life      int
	Combo     int
	Score     int
	HighScore int
}

type player struct {
	Form     string
	Rows     int
	Cols     int
	ColorMap map[string]termbox.Attribute
	Pos      point
}

type invader struct {
	Forms    []string
	Rows     int
	Cols     int
	Color    termbox.Attribute
	Pos      point
	Vec      point
	Interval int
	Life     int
}

type bullet struct {
	Form     string
	Rows     int
	Cols     int
	Color    termbox.Attribute
	Pos      point
	Vec      point
	Interval int
	CloseCh  chan bool
}

//タイマーイベント
func moveLoop(moveCh chan int, closeCh chan bool, mover, ticker int) {
	t := time.NewTicker(time.Duration(ticker) * time.Millisecond)
	defer t.Stop()
	for {
		select {
		case <-t.C: //タイマーイベント
			moveCh <- mover
			break
		case <-closeCh:
			return
		}
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
		if st.End == false {
			for k := range st.Invaders {
				drawInvader(*st.Invaders[k])
			}
			for k := range st.Bullets {
				drawBullet(*st.Bullets[k])
			}
		} else {
			displayMessage("PUSH ENTER KEY")
		}
		for i := 0; i < _width+1; i++ {
			drawLine(i, 1, "-")
			drawLine(i, _height, "-")
		}
		for i := 0; i < _height; i++ {
			drawLine(0, i, "|")
			drawLine(_width, i, "|")
		}
		drawLine(1, 0, "EXIT : ESC KEY")
		drawLine(30, 0, fmt.Sprintf("NumGoroutine: %03d", runtime.NumGoroutine()))
		drawLine(_width-65, 0, fmt.Sprintf("HighScore: %05d", st.HighScore))
		drawLine(_width-45, 0, fmt.Sprintf("Score: %05d", st.Score))
		drawLine(_width-30, 0, fmt.Sprintf("Combo: %03d", st.Combo))
		drawLine(_width-10, 0, fmt.Sprintf("Life: %02d", st.Life))
		drawPlayer(st.Player)
		termbox.Flush()
		mu.Unlock()
	}
}

func displayMessage(msg string) {
	frameWidth := int(float64(len(msg)) * 1.5)
	for i := 0; i < frameWidth; i++ {
		drawLine(_width/2-frameWidth/2+i, _height/2-1, "=")
		drawLine(_width/2-frameWidth/2+i, _height/2+1, "=")
	}
	drawLine(_width/2-len(msg)/2, _height/2, msg)
}

//行を描画
func drawLine(x, y int, str string) {
	runes := []rune(str)
	for i := 0; i < len(runes); i++ {
		termbox.SetCell(x+i, y, runes[i], termbox.ColorDefault, termbox.ColorDefault)
	}
}

//プレイヤーを描画
func drawPlayer(player *player) {
	scanner := bufio.NewScanner(strings.NewReader(player.Form))
	j := 0
	for scanner.Scan() {
		line := scanner.Text()
		runes := []rune(line)
		for i := 0; i < len(runes); i++ {
			color := termbox.ColorDefault
			mapkey := strconv.Itoa(i) + "-" + strconv.Itoa(j)
			if _, ok := player.ColorMap[mapkey]; ok {
				color = player.ColorMap[mapkey]
			}
			termbox.SetCell(player.Pos.X+i, player.Pos.Y+j, runes[i], color, termbox.ColorDefault)
		}
		j++
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

//弾丸を描画
func drawBullet(bullet bullet) {
	runes := []rune(bullet.Form)
	for i := 0; i < len(runes); i++ {
		termbox.SetCell(bullet.Pos.X+i, bullet.Pos.Y, runes[i], bullet.Color, termbox.ColorDefault)
	}
}

func choice(len int) int {
	rand.Seed(time.Now().UnixNano())
	i := rand.Intn(len)
	return i
}

//ゲームメイン処理
func controller(st state, stateCh chan state, keyCh chan termbox.Key, moveCh chan int) {
	bulletCh := make(chan int)
	bulletCnt := 0
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
				if st.Player.Pos.X-3 > 0 {
					st.Player.Pos.X -= 3
				}
				break
			case termbox.KeyArrowRight: //みぎ
				if st.Player.Pos.X+st.Player.Cols+3 < _width {
					st.Player.Pos.X += 3
				}
				break
			case termbox.KeyEnter: //ゲームスタート
				if st.End {
					st.End = false
					st.Score = 0
				}
				break
			case termbox.KeySpace: //発射
				bulletCnt++
				st.Bullets[bulletCnt] = fire(st.Player.Pos.X + st.Player.Cols/2)
				go func(closeCh chan bool, key, ticker int) {
					moveLoop(bulletCh, closeCh, key, ticker)
				}(st.Bullets[bulletCnt].CloseCh, bulletCnt, st.Bullets[bulletCnt].Interval)
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
		case i := <-bulletCh: //タイマーイベント
			mu.Lock()
			if st.End == false {
				if _, ok := st.Bullets[i]; ok {
					st.Bullets[i].Pos.Y += st.Bullets[i].Vec.Y
					st = checkHit(st, i)
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
	st.Life = 3
	st.Player = initPlayer()
	st.Invaders = initInvaders()
	st.Bullets = map[int]*bullet{}

	return st
}

func initPlayer() *player {
	form := strings.TrimLeft(`
  ▙▉▟
▞▓░▒░▓▚`, "\n")
	colormap := map[string]termbox.Attribute{
		"2-0": termbox.ColorGreen,
		"3-0": termbox.ColorBlue,
		"4-0": termbox.ColorGreen,
		"0-1": termbox.ColorGreen,
		"1-1": termbox.ColorCyan,
		"2-1": termbox.ColorMagenta,
		"3-1": termbox.ColorRed,
		"4-1": termbox.ColorMagenta,
		"5-1": termbox.ColorCyan,
		"6-1": termbox.ColorGreen,
	}
	rows := strings.Count(form, "\n") + 1
	cols := utf8.RuneCountInString(strings.Split(form, "\n")[rows-1])
	player := player{
		Form:     form,
		Rows:     rows,
		Cols:     cols,
		ColorMap: colormap,
		Pos:      point{X: _width/2 - cols/2, Y: _height - rows},
	}

	return &player
}

func initInvaders() map[int]*invader {
	var colors = []termbox.Attribute{
		termbox.ColorGreen,
		termbox.ColorYellow,
		termbox.ColorBlue,
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
	cols := utf8.RuneCountInString(strings.Split(form1, "\n")[rows-1])
	for i := 0; i < 20; i++ {
		invaders[i] = &invader{
			Forms:    []string{form1, form2},
			Rows:     rows,
			Cols:     cols,
			Color:    colors[choice(len(colors))],
			Pos:      point{X: choice(_width-cols) + 1, Y: choice(_height*2/3) + 2},
			Vec:      point{X: plusminus[choice(2)], Y: plusminus[choice(2)]},
			Interval: ((i % 5) + 1) * 50,
			Life:     3,
		}
	}
	return invaders
}

func fire(x int) *bullet {
	return &bullet{
		Form:     "▘",
		Rows:     1,
		Cols:     1,
		Color:    termbox.ColorDefault,
		Pos:      point{X: x, Y: _height - 3},
		Vec:      point{X: 0, Y: -1},
		Interval: 50,
		CloseCh:  make(chan bool),
	}
}

//衝突判定
func checkCollision(st state, i int) state {
	//Playerが右の壁に到達
	if st.Player.Pos.X+st.Player.Cols > _width {
		st.Player.Pos.X -= 3
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
	if st.Invaders[i].Pos.Y <= 1 {
		st.Invaders[i].Pos.Y = 2
		st.Invaders[i].Vec.Y = 1
	}
	//下の壁
	if st.Invaders[i].Pos.Y+st.Invaders[i].Rows >= _height {
		st.Invaders[i].Pos.Y = _height - st.Invaders[i].Rows
		st.Invaders[i].Vec.Y = -1
	}
	//インベーダー同士
	for o := range st.Invaders {
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

	//Playerとの衝突判定
	if st.Invaders[i].Pos.X+st.Invaders[i].Cols >= st.Player.Pos.X &&
		st.Invaders[i].Pos.X <= st.Player.Pos.X+st.Player.Cols &&
		st.Invaders[i].Pos.Y == _height-2-st.Invaders[i].Rows {
		st.Invaders[i].Vec.Y = -1
		if st.Invaders[i].Pos.X+(st.Invaders[i].Cols/2) <= st.Player.Pos.X+(st.Player.Cols/2) {
			st.Invaders[i].Vec.X = -1
		} else {
			st.Invaders[i].Vec.X = +1
		}
		st.Life--
		if st.Life <= 0 {
			st = resetGame(st)
		}
	}

	return st
}

func checkHit(st state, i int) state {
	bullet := st.Bullets[i]
	//命中判定
	for o, invader := range st.Invaders {
		if bullet.Pos.X+bullet.Cols >= invader.Pos.X && bullet.Pos.X <= invader.Pos.X+invader.Cols &&
			bullet.Pos.Y+bullet.Rows >= invader.Pos.Y && bullet.Pos.Y <= invader.Pos.Y+invader.Rows {
			close(st.Bullets[i].CloseCh)
			delete(st.Bullets, i)
			st.Invaders[o].Life--
			st.Combo++
			switch st.Invaders[o].Life {
			case 2:
				st.Invaders[o].Color = termbox.ColorMagenta
			case 1:
				st.Invaders[o].Color = termbox.ColorRed
			case 0:
				delete(st.Invaders, o)
				st.Score += st.Combo
				if st.HighScore < st.Score {
					st.HighScore = st.Score
				}
			}
			//インベーダー全撃破
			if len(st.Invaders) == 0 {
				st = resetGame(st)
			}
			return st
		}
	}

	//外れたので消す
	if st.Bullets[i].Pos.Y <= 1 {
		st.Combo = 0
		close(st.Bullets[i].CloseCh)
		delete(st.Bullets, i)
	}

	return st
}

func resetGame(st state) state {
	hs := st.HighScore
	s := st.Score
	if st.HighScore < st.Score {
		hs = st.Score
	}
	for k := range st.Bullets {
		close(st.Bullets[k].CloseCh)
	}
	st = initGame()
	st.HighScore = hs
	st.Score = s
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
			moveLoop(moveCh, nil, idx, ticker)
		}(k, v.Interval)
	}

	controller(st, stateCh, keyCh, moveCh)
	termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)

	defer termbox.Close()
}
