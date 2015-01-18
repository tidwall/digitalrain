package main

import (
	"errors"
	"fmt"
	"github.com/gopherjs/gopherjs/js"
	"log"
	"math/rand"
	"time"
)

func main() {
	sheet := js.Global.Get("document").Call("createElement", "style")
	sheet.Set("innerHTML",
		"html, body { background: black; padding:0; margin:0; border:0; width:100%; height:100%; overflow:hidden;}")
	js.Global.Get("document").Get("head").Call("appendChild", sheet)
	js.Global.Get("document").Set("title", "Whoa")
	js.Global.Call("addEventListener", "load", func() {
		rain1, err := NewDigitalRain(js.Global.Get("document").Get("body"), 60, 2, 8, 0.25)
		if err != nil {
			log.Println(err.Error())
			return
		}
		js.Global.Call("addEventListener", "resize", func() {
			rain1.layout()
		})

		rain2, err := NewDigitalRain(js.Global.Get("document").Get("body"), 40, 2, 12, 1.0)
		if err != nil {
			log.Println(err.Error())
			return
		}
		js.Global.Call("addEventListener", "resize", func() {
			rain2.layout()
		})
	})
}

const (
	showBlueHeads = true
	//screenCols    = 65
	overlap    = 0 // 0-N, allowable drop overlaps for a column
	highColor  = "#7bB5C8"
	lowColor   = "#3b806d"
	githubLink = "http://github.com/tidwall/digitalrain"
)

type DigitalRain struct {
	parent, canvas  js.Object
	ctx             js.Object
	width, height   float64
	ratio           float64
	timestamp       time.Duration
	highGlyphCanvas js.Object
	lowGlyphCanvas  js.Object
	drops           []*waterDrop
	linkover        bool
	screenCols      int
	minSpeed        int
	maxSpeed        int
	brightness      float64
}

func NewDigitalRain(parent js.Object, screenCols int, minSpeed int, maxSpeed int, brightness float64) (*DigitalRain, error) {
	rain := &DigitalRain{parent: parent}
	rain.screenCols = screenCols
	rain.minSpeed = minSpeed
	rain.maxSpeed = maxSpeed
	rain.brightness = brightness
	if err := rain.start(); err != nil {
		return nil, err
	}
	return rain, nil
}
func (r *DigitalRain) start() error {
	rand.Seed(time.Now().UnixNano())
	var raf string
	for _, s := range []string{"requestAnimationFrame", "webkitRequestAnimationFrame", "mozRequestAnimationFrame"} {
		if js.Global.Get(s) != js.Undefined {
			raf = s
			break
		}
	}
	if raf == "" {
		return errors.New("requestAnimationFrame is not available")
	}
	defer r.layout()
	count := 0
	var f func(js.Object)
	f = func(timestampJS js.Object) {
		js.Global.Call(raf, f)
		if count%2 == 0 {
			r.loop(time.Duration(timestampJS.Float() * float64(time.Millisecond)))
		}
		count++
	}
	js.Global.Call(raf, f)
	return nil
}

func (r *DigitalRain) layout() {
	ratio := js.Global.Get("devicePixelRatio").Float()
	width := r.parent.Get("offsetWidth").Float() * ratio
	height := r.parent.Get("offsetHeight").Float() * ratio
	if r.canvas != nil && r.width == width && r.height == height && r.ratio == ratio {
		return
	}
	r.width, r.height, r.ratio = width, height, ratio
	if r.canvas != nil {
		r.parent.Call("removeChild", r.canvas)
	}
	r.canvas = js.Global.Get("document").Call("createElement", "canvas")
	r.ctx = r.canvas.Call("getContext", "2d")
	r.canvas.Set("width", r.width)
	r.canvas.Set("height", r.height)
	r.canvas.Get("style").Set("width", fmt.Sprintf("%.4fpx", r.width/r.ratio))
	r.canvas.Get("style").Set("height", fmt.Sprintf("%.4fpx", r.height/r.ratio))
	r.canvas.Get("style").Set("position", "absolute")
	r.parent.Call("appendChild", r.canvas)
	if r.highGlyphCanvas == nil {
		r.highGlyphCanvas = generateGlyphCanvas(highColor)
	}
	if r.lowGlyphCanvas == nil {
		r.lowGlyphCanvas = generateGlyphCanvas(lowColor)
	}

	r.canvas.Call("addEventListener", "click", func(ev js.Object) {
		if r.overLink(ev.Get("x").Int(), ev.Get("y").Int()) {
			js.Global.Set("location", githubLink)
		}
	})

	r.canvas.Call("addEventListener", "mousemove", func(ev js.Object) {
		if r.overLink(ev.Get("x").Int(), ev.Get("y").Int()) {
			r.canvas.Get("style").Set("cursor", "pointer")
			r.linkover = true
		} else {
			r.canvas.Get("style").Set("cursor", "default")
			r.linkover = false
		}
	})

	r.loop(r.timestamp)
}

func (r *DigitalRain) overLink(x int, y int) bool {
	return x > int(r.width/r.ratio)-320 && y > int(r.height/r.ratio)-50
}

type waterDrop struct {
	col     int     // the column the drop it dropping
	row     float64 // the row of the bottom most cell
	start   float64 // the starting row
	speed   float64 // how many cells per second
	glyphs  []int   // random glyph table
	spedup  bool
	created time.Duration
}

func (r *DigitalRain) dropWaterAtCol(col int, speed float64, length int, start float64, created time.Duration) {
	wd := waterDrop{}
	wd.col = col
	wd.speed = speed
	wd.glyphs = make([]int, length)
	for i := 0; i < length; i++ {
		wd.glyphs[i] = rand.Int() % glyphsCount
	}
	r.drops = append(r.drops, &wd)

	wd.row = start
	wd.start = wd.row
	wd.created = created
}

func (r *DigitalRain) dropRandomWaterDrop(timestamp time.Duration) {
	col := rand.Int() % r.screenCols
	colcnt := 0
	for _, drop := range r.drops {
		if drop.col == col && int(drop.row)-len(drop.glyphs) < 0 {
			colcnt++
			if colcnt > overlap {
				return // no space
			}
		}
	}
	speed := float64(rand.Int()%(r.maxSpeed-r.minSpeed) + r.minSpeed)
	length := rand.Int()%30 + 10
	start := float64((rand.Int() % r.maxRows()) - r.maxRows()/2)
	r.dropWaterAtCol(col, speed, length, start, timestamp)
}
func (r *DigitalRain) maxRows() int {
	cellSize := r.width / float64(r.screenCols)
	return int((r.height / cellSize) + 2)
}
func (r *DigitalRain) drawGlyphAt(nidx int, col int, row float64, brightness float64, head bool) {
	if col < 0 || col > r.screenCols || row < -1 || row > float64(r.maxRows()) {
		return
	}
	r.drawGlyphElAt(r.lowGlyphCanvas, nidx, col, row, brightness)
	if head {
		r.drawGlyphElAt(r.highGlyphCanvas, nidx, col, row, brightness)
	}
}
func (r *DigitalRain) drawGlyphElAt(glyphCanvas js.Object, nidx int, col int, row float64, brightness float64) {
	if brightness <= 0.05 {
		return
	}
	if brightness > 1 {
		brightness = 1
	}
	cellSize := r.width / float64(r.screenCols)
	gy := int(nidx/glyphsCols) * glyphCellSize
	gx := int(nidx%glyphsCols) * glyphCellSize
	cx := cellSize*float64(col) + cellSize/2 - (cellSize*1.5)/2
	cy := cellSize * float64(row)
	r.ctx.Call("save")
	r.ctx.Set("globalAlpha", brightness)
	r.ctx.Call("drawImage", glyphCanvas, gx, gy, glyphCellSize, glyphCellSize, cx, cy, cellSize*1.5, cellSize*1.5)
	r.ctx.Call("restore")
}

func (r *DigitalRain) drawTitle(text string, color string, fontSize float64, y float64) float64 {
	ny := y + (fontSize * 1.5)
	pad := 15 * r.ratio
	x := r.width - pad
	y = r.height - pad - y
	r.ctx.Call("save")
	r.ctx.Set("font", fmt.Sprintf("%dpx Menlo, Consolas, Monospace, Helvetica, Arial, Sans-Serif", int(fontSize)))
	r.ctx.Set("textAlign", "right")
	r.ctx.Set("lineWidth", 0)
	r.ctx.Set("shadowColor", color)
	r.ctx.Set("shadowBlur", float64(fontSize))
	r.ctx.Set("fillStyle", color)
	r.ctx.Call("fillText", text, x, y)
	r.ctx.Call("restore")
	return ny
}

func (r *DigitalRain) drawTitles() {
	color := "59,128,109"
	//color := "123,181,200"
	y := float64(0)
	//y = r.drawTitle("HTML5 + Canvas, Written in Go", "rgba("+color+",.5)", 15*r.ratio, y)
	if r.linkover {
		y = r.drawTitle("github.com/tidwall/digitalrain", "rgba("+color+",1)", 15*r.ratio, y)
	} else {
		y = r.drawTitle("github.com/tidwall/digitalrain", "rgba("+color+",.5)", 15*r.ratio, y)
	}
	//y = r.drawTitle("Digital Rain", "rgba("+color+",.7)", 20*r.ratio, y)

}

func (r *DigitalRain) loop(timestamp time.Duration) {
	if timestamp == 0 || r.timestamp == 0 {
		r.timestamp = timestamp
		return
	}
	elapsed := timestamp - r.timestamp
	r.timestamp = timestamp

	r.dropRandomWaterDrop(timestamp)
	r.ctx.Call("clearRect", 0, 0, r.width, r.height)
	defer r.drawTitles()

	var drops []*waterDrop
	for _, drop := range r.drops {
		if !drop.spedup {
			if rand.Int()%250 == 0 {
				drop.speed *= float64(rand.Int()%3) + 0.8
				drop.spedup = true
			}
		}
		gbrightness := float64(r.brightness)
		age := (timestamp - drop.created)
		if age < time.Second {
			gbrightness = float64(age) / float64(time.Second)
		}

		drop.row += float64(elapsed) / float64(time.Second) * drop.speed
		gl := len(drop.glyphs)

		if int(drop.row)-gl > r.maxRows() {
			continue
		}
		drops = append(drops, drop)
		gcount := int(drop.row - drop.start)
		if gcount > len(drop.glyphs) {
			gcount = len(drop.glyphs)
		}
		for i := 0; i < gcount; i++ {
			glyph := drop.glyphs[i]
			brightness := 1 - (float64(i) / float64(gcount))
			// one in N change that the glyph will change, just because
			if rand.Int()%50 == 0 {
				glyph = rand.Int() % glyphsCount
				drop.glyphs[i] = glyph
			}
			row := drop.row - float64(i)
			r.drawGlyphAt(glyph, drop.col, row, brightness*gbrightness, i == 0)
		}
	}
	r.drops = drops
}

const (
	glyphs        = "02345789ABCEGIJMNPRVXYZ:>+*~｡､･ｦｰｱｲｳｴｵｶｷｸｺｻｼｾｿﾀﾁﾂﾄﾅﾆﾇﾈﾋﾌﾍﾎﾏﾐﾑﾓﾔﾕﾖﾗﾘﾙﾚﾛﾜﾝ"
	glyphsCols    = 18
	glyphsCount   = 72
	glyphCellSize = 100
	glyphFontSize = 86
)

func generateGlyphCanvas(color string) js.Object {
	glyphCanvas := js.Global.Get("document").Call("createElement", "canvas")
	glyphCanvas.Set("width", int(glyphCellSize*glyphsCols+glyphCellSize))
	glyphCanvas.Set("height", int(glyphCellSize*glyphsCount/glyphsCols+glyphCellSize))
	ctx := glyphCanvas.Call("getContext", "2d")
	col := 0
	row := 1
	for i, c := range glyphs {
		if col == 18 {
			row++
			col = 0
		}
		cellSize := float64(glyphCellSize)
		fontSize := float64(glyphFontSize)
		if i <= 36 {
			fontSize *= .87
		}
		cellSize = cellSize
		//ctx.Call("translate", cellSize*float64(col)+cellSize/2, cellSize*float64(row)+(fontSize-cellSize))
		ctx.Call("save")
		ctx.Set("textAlign", "center")
		ctx.Set("font", fmt.Sprintf("%dpx Monaco, Helvetica, Arial, Sans-Serif", int(fontSize)))
		ctx.Set("shadowColor", color)
		ctx.Set("shadowBlur", float64(fontSize)*.60)
		switch c {
		default:
			ctx.Call("translate", cellSize*float64(col)+cellSize/2, cellSize*float64(row)+(fontSize-cellSize))
		case '2', '4', '9':
			ctx.Call("scale", -1, 1)
			ctx.Call("translate", -(cellSize*float64(col) + cellSize/2), cellSize*float64(row)+(fontSize-cellSize))
		}
		for i := 0; i < 3; i++ {
			ctx.Set("fillStyle", color)
			//ctx.Call("fillText", string(c), cellSize*float64(col)+cellSize/2, cellSize*float64(row)+(fontSize-cellSize))
			ctx.Call("fillText", string(c), 0, 0)

		}
		ctx.Call("restore")
		col++
	}
	return glyphCanvas
}
