package schgen

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

// Проверка наложений на готовом листе (.kicad_sch): надписи друг на друге, надписи на чужих корпусах и проводах,
// провода сквозь корпуса, корпуса друг на друге. Замечание руководителя 06.10.2026: «взаимные пересечения УГО».
// Геометрия приближённая (ширина символа шрифта — charW от высоты), поэтому допуски небольшие, а ловим явное.

const charW = 0.92 // ширина знака / высота шрифта с промежутком: замер по PNG листа (GOST type A, «I/O7L» 1,27 мм — 5,84 мм)

type box struct{ x0, y0, x1, y1 float64 }

func (b box) w() float64 { return b.x1 - b.x0 }
func (b box) h() float64 { return b.y1 - b.y0 }
func (b box) shrink(d float64) box {
	return box{b.x0 + d, b.y0 + d, b.x1 - d, b.y1 - d}
}
func (b box) hit(c box) bool { return b.x0 < c.x1 && c.x0 < b.x1 && b.y0 < c.y1 && c.y0 < b.y1 }
func boxOf(pts ...Pt) box {
	b := box{math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)}
	for _, p := range pts {
		b.x0, b.y0 = math.Min(b.x0, p.X), math.Min(b.y0, p.Y)
		b.x1, b.y1 = math.Max(b.x1, p.X), math.Max(b.y1, p.Y)
	}
	return b
}

// segHit — отрезок a–b заходит внутрь прямоугольника (не только касается края).
func segHit(a, b Pt, r box) bool {
	if math.Max(a.X, b.X) <= r.x0 || math.Min(a.X, b.X) >= r.x1 || math.Max(a.Y, b.Y) <= r.y0 || math.Min(a.Y, b.Y) >= r.y1 {
		return false
	}
	if a.X == b.X || a.Y == b.Y { // наши провода — только горизонтали и вертикали
		return true
	}
	// наклонный: проверка по нескольким точкам
	for t := 0.0; t <= 1; t += 0.05 {
		p := Pt{a.X + (b.X-a.X)*t, a.Y + (b.Y-a.Y)*t}
		if p.X > r.x0 && p.X < r.x1 && p.Y > r.y0 && p.Y < r.y1 {
			return true
		}
	}
	return false
}

// textBox — прямоугольник надписи: точка привязки, угол 0/90/180/270, выравнивание, высота шрифта.
func textBox(txt string, at Pt, ang float64, hj, vj string, size float64) box {
	txt = strings.NewReplacer("~{", "", "}", "").Replace(txt)
	lines := strings.Split(txt, "\n")
	n := 0
	for _, l := range lines {
		n = max(n, len([]rune(l)))
	}
	w, h := float64(n)*size*charW, size*(1+1.6*float64(len(lines)-1))
	var x0, y0 float64
	switch hj {
	case "left":
		x0 = 0
	case "right":
		x0 = -w
	default:
		x0 = -w / 2
	}
	switch vj {
	case "top":
		y0 = 0
	case "bottom":
		y0 = -h
	default:
		y0 = -h / 2
	}
	// локальный прямоугольник (y вниз) → поворот на ang против часовой (90 — текст снизу вверх)
	c := []Pt{{x0, y0}, {x0 + w, y0}, {x0, y0 + h}, {x0 + w, y0 + h}}
	k := ((int(math.Round(ang))/90)%4 + 4) % 4
	for i := range c {
		x, y := c[i].X, c[i].Y
		for j := 0; j < k; j++ {
			x, y = y, -x
		}
		c[i] = Pt{at.X + x, at.Y + y}
	}
	return boxOf(c...)
}

func effects(n *Node) (size float64, hj, vj string, hidden bool) {
	size = 1.27
	hj, vj = "center", "center"
	e := n.Find("effects")
	if e == nil {
		return
	}
	if f := e.Find("font"); f != nil {
		if s := f.Find("size"); s != nil {
			size = s.Num(1)
		}
	}
	if j := e.Find("justify"); j != nil {
		for i := 0; ; i++ {
			a := j.Arg(i)
			if a == "" {
				break
			}
			switch a {
			case "left", "right":
				hj = a
			case "top", "bottom":
				vj = a
			}
		}
	}
	for _, k := range e.Kids {
		if (!k.IsList() && k.Atom == "hide") || (k.Head() == "hide" && k.Arg(0) == "yes") {
			hidden = true
		}
	}
	if h := n.Find("hide"); h != nil && h.Arg(0) == "yes" {
		hidden = true
	}
	return
}

type olItem struct {
	kind  string // body | text | pin
	owner string // обозначение элемента или метка
	what  string // для сообщения
	b     box
	a, z  Pt   // для pin — отрезок вывода
	field bool // обозначение/тип элемента — проверяется и против своего корпуса
	at    *Pt  // точка привязки метки: свой провод (через неё проходящий) надпись не перечёркивает
}

// units — число частей символа (подсимволы NAME_u_s с u ≥ 1).
func units(def *Node) int {
	n := 0
	for _, sub := range def.All("symbol") {
		p := strings.Split(sub.Arg(0), "_")
		if len(p) >= 2 {
			n = max(n, atoi(p[len(p)-2]))
		}
	}
	return n
}

// Overlaps — список наложений на листе src (текст .kicad_sch).
func Overlaps(src string) ([]string, error) {
	root, err := Parse(src)
	if err != nil {
		return nil, err
	}
	libs := map[string]*Node{}
	if ls := root.Find("lib_symbols"); ls != nil {
		for _, s := range ls.All("symbol") {
			libs[s.Arg(0)] = s
		}
	}
	var items, pbodies, glines []olItem
	var wires [][2]Pt
	for _, w := range root.All("wire") {
		p := w.Find("pts").All("xy")
		wires = append(wires, [2]Pt{{p[0].Num(0), p[0].Num(1)}, {p[1].Num(0), p[1].Num(1)}})
	}
	var buses [][2]Pt
	for _, w := range root.All("bus") {
		p := w.Find("pts").All("xy")
		buses = append(buses, [2]Pt{{p[0].Num(0), p[0].Num(1)}, {p[1].Num(0), p[1].Num(1)}})
	}
	pn := 0
	for _, inst := range root.All("symbol") {
		lid := inst.Find("lib_id")
		if lid == nil {
			continue
		}
		def := libs[lid.Arg(0)]
		at := inst.Find("at")
		pos, rot := Pt{at.Num(0), at.Num(1)}, int(at.Num(2))
		unit := 1
		if u := inst.Find("unit"); u != nil {
			unit = int(u.Num(0))
		}
		ref := ""
		for _, p := range inst.All("property") {
			if p.Arg(0) == "Reference" {
				ref = p.Arg(1)
			}
		}
		power := strings.HasPrefix(ref, "#")
		owner := ref
		if power {
			pn++
			owner = fmt.Sprintf("%s#%d", ref, pn)
		}
		// надписи элемента
		for _, p := range inst.All("property") {
			name := p.Arg(0)
			if name != "Reference" && name != "Value" {
				continue
			}
			size, hj, vj, hidden := effects(p)
			if hidden || p.Arg(1) == "" {
				continue
			}
			pa := p.Find("at")
			txt := p.Arg(1)
			if name == "Reference" && def != nil && units(def) > 1 {
				txt += fmt.Sprintf(".%d", unit) // KiCad пишет часть корпуса через точку (DD3.1)
			}
			// KiCad поворачивает надпись вместе с элементом и держит её читаемой: 0 или 90 на листе
			ang := math.Mod(pa.Num(2)+float64(rot)+360, 180)
			if (rot%360+360)%360 == 90 { // KiCad зеркалит выравнивание у элемента, повёрнутого на 90° (см. Sheet.Sym)
				hj = map[string]string{"left": "right", "right": "left", "center": "center"}[hj]
			}
			items = append(items, olItem{kind: "text", owner: owner, what: fmt.Sprintf("надпись %q", txt), field: true,
				b: textBox(txt, Pt{pa.Num(0), pa.Num(1)}, ang, hj, vj, size)})
		}
		if def == nil {
			continue
		}
		pinNumHide, pinNameHide, nameOff := false, false, 0.508
		if n := def.Find("pin_numbers"); n != nil && n.Find("hide") != nil && n.Find("hide").Arg(0) == "yes" {
			pinNumHide = true
		}
		if n := def.Find("pin_names"); n != nil {
			if n.Find("hide") != nil && n.Find("hide").Arg(0) == "yes" {
				pinNameHide = true
			}
			if o := n.Find("offset"); o != nil {
				nameOff = o.Num(0)
			}
		}
		name := def.Arg(0)
		var body []Pt
		for _, sub := range def.All("symbol") {
			parts := strings.Split(sub.Arg(0), "_")
			if len(parts) < 2 {
				continue
			}
			u, st := atoi(parts[len(parts)-2]), atoi(parts[len(parts)-1])
			if (u != 0 && u != unit) || st > 1 {
				continue
			}
			_ = name
			for _, g := range sub.Kids {
				switch g.Head() {
				case "rectangle":
					s, e := g.Find("start"), g.Find("end")
					body = append(body, xform(pos, rot, s.Num(0), s.Num(1)), xform(pos, rot, e.Num(0), e.Num(1)))
				case "polyline":
					var pl []Pt
					for _, xy := range g.Find("pts").All("xy") {
						p := xform(pos, rot, xy.Num(0), xy.Num(1))
						body = append(body, p)
						pl = append(pl, p)
					}
					for k := 0; k+1 < len(pl); k++ {
						glines = append(glines, olItem{kind: "gline", owner: owner, a: pl[k], z: pl[k+1]})
					}
				case "circle":
					c, r := g.Find("center"), g.Find("radius").Num(0)
					body = append(body, xform(pos, rot, c.Num(0)-r, c.Num(1)-r), xform(pos, rot, c.Num(0)+r, c.Num(1)+r))
				case "text":
					size, hj, vj, h := effects(g)
					ta := g.Find("at")
					if !h && strings.TrimSpace(g.Arg(0)) != "" {
						c := xform(pos, rot, ta.Num(0), ta.Num(1))
						items = append(items, olItem{kind: "text", owner: owner, what: fmt.Sprintf("знак %q", g.Arg(0)),
							b: textBox(g.Arg(0), c, 0, hj, vj, size)})
					}
				case "pin":
					pa := g.Find("at")
					ln := 0.0
					if l := g.Find("length"); l != nil {
						ln = l.Num(0)
					}
					hid := false
					for _, k := range g.Kids {
						if (!k.IsList() && k.Atom == "hide") || (k.Head() == "hide" && k.Arg(0) == "yes") {
							hid = true
						}
					}
					if hid || ln == 0 {
						continue
					}
					ang := pa.Num(2)
					dx, dy := math.Cos(ang*math.Pi/180), math.Sin(ang*math.Pi/180)
					a := xform(pos, rot, pa.Num(0), pa.Num(1))
					z := xform(pos, rot, pa.Num(0)+dx*ln, pa.Num(1)+dy*ln)
					items = append(items, olItem{kind: "pin", owner: owner, what: "вывод", a: a, z: z, b: boxOf(a, z)})
					horiz := math.Abs(a.Y-z.Y) < 0.01
					if num := g.Find("number"); num != nil && !pinNumHide && !power {
						size, _, _, h := effects(num)
						if !h {
							mid := Pt{(a.X + z.X) / 2, (a.Y + z.Y) / 2}
							if horiz {
								items = append(items, olItem{kind: "text", owner: owner, what: "номер вывода " + num.Arg(0),
									b: textBox(num.Arg(0), mid.Add(0, -0.254), 0, "center", "bottom", size)})
							} else {
								items = append(items, olItem{kind: "text", owner: owner, what: "номер вывода " + num.Arg(0),
									b: textBox(num.Arg(0), mid.Add(-0.254, 0), 90, "center", "bottom", size)})
							}
						}
					}
					if nm := g.Find("name"); nm != nil && !pinNameHide && nm.Arg(0) != "" && nm.Arg(0) != "~" && nameOff > 0 {
						size, _, _, h := effects(nm)
						if !h {
							// имя — внутри корпуса за концом вывода
							ux, uy := (z.X-a.X)/ln, (z.Y-a.Y)/ln
							s := z.Add(ux*nameOff, uy*nameOff)
							var tb box
							switch {
							case horiz && ux > 0:
								tb = textBox(nm.Arg(0), s, 0, "left", "center", size)
							case horiz:
								tb = textBox(nm.Arg(0), s, 0, "right", "center", size)
							case uy < 0: // вывод снизу, имя вверх
								tb = textBox(nm.Arg(0), s, 90, "left", "center", size)
							default:
								tb = textBox(nm.Arg(0), s, 90, "right", "center", size)
							}
							items = append(items, olItem{kind: "text", owner: owner, what: "имя вывода " + nm.Arg(0), b: tb})
						}
					}
				}
			}
		}
		if len(body) > 0 && !power {
			items = append(items, olItem{kind: "body", owner: owner, what: "корпус", b: boxOf(body...)})
		}
		if len(body) > 0 && power {
			pbodies = append(pbodies, olItem{kind: "pbody", owner: owner, b: boxOf(body...)})
		}
	}
	for _, l := range root.All("label") {
		size, hj, vj, _ := effects(l)
		at := l.Find("at")
		// метка под 180°/270° KiCad хранит с выравниванием «right»: текст идёт от точки влево/вниз — как 0°/90° с этим выравниванием
		ap := Pt{at.Num(0), at.Num(1)}
		items = append(items, olItem{kind: "text", owner: "метка " + l.Arg(0), what: fmt.Sprintf("метка %q", l.Arg(0)), at: &ap,
			b: textBox(l.Arg(0), ap, math.Mod(at.Num(2), 180), hj, vj, size)})
	}
	for _, t := range root.All("text") {
		size, hj, vj, _ := effects(t)
		at := t.Find("at")
		items = append(items, olItem{kind: "text", owner: "текст", what: fmt.Sprintf("текст %q", firstLine(t.Arg(0))),
			b: textBox(t.Arg(0), Pt{at.Num(0), at.Num(1)}, at.Num(2), hj, vj, size)})
	}

	var out []string
	add := func(f string, a ...any) { out = append(out, fmt.Sprintf(f, a...)) }
	const tol = 0.15
	for i := range items {
		a := items[i]
		for j := i + 1; j < len(items); j++ {
			b := items[j]
			if a.owner == b.owner {
				// своё обозначение/тип — не на своём корпусе, выводах и надписях внутри корпуса
				f, o := a, b
				if b.field {
					f, o = b, a
				}
				// имена и номера выводов одного корпуса — не друг на друге
				if a.kind == "text" && b.kind == "text" && !a.field && !b.field &&
					strings.Contains(a.what, "вывод") && strings.Contains(b.what, "вывод") && a.b.shrink(tol).hit(b.b.shrink(tol)) {
					add("%s и %s (%s) наезжают друг на друга", a.what, b.what, a.owner)
				}
				if f.field && !o.field && f.kind == "text" {
					switch {
					case o.kind == "pin" && segHit(o.a, o.z, f.b.shrink(tol)):
						add("%s (%s) на своём выводе", f.what, f.owner)
					case o.kind != "pin" && f.b.shrink(tol).hit(o.b.shrink(tol)):
						add("%s (%s) на своём корпусе или %s", f.what, f.owner, o.what)
					}
				}
				continue
			}
			switch {
			case a.kind == "text" && b.kind == "text":
				// не слипаются: зазор ≥ 0,25 мм («49Y1stb» — номер линии вплотную к имени)
				if a.b.shrink(-0.125).hit(b.b.shrink(-0.125)) {
					add("%s (%s) наезжает на %s (%s)", a.what, a.owner, b.what, b.owner)
				}
			case a.kind == "text" && b.kind == "body", a.kind == "body" && b.kind == "text":
				t, o := a, b
				if a.kind == "body" {
					t, o = b, a
				}
				if t.b.shrink(tol).hit(o.b.shrink(tol)) {
					add("%s (%s) на корпусе %s, около (%.1f; %.1f) мм", t.what, t.owner, o.owner, t.b.x0, t.b.y0)
				}
			case a.kind == "text" && b.kind == "pin", a.kind == "pin" && b.kind == "text":
				t, p := a, b
				if a.kind == "pin" {
					t, p = b, a
				}
				if segHit(p.a, p.z, t.b.shrink(tol)) {
					add("%s (%s) на выводе %s", t.what, t.owner, p.owner)
				}
			case a.kind == "body" && b.kind == "body":
				if a.b.shrink(tol).hit(b.b.shrink(tol)) {
					add("корпус %s на корпусе %s", a.owner, b.owner)
				}
			case a.kind == "body" && b.kind == "pin", a.kind == "pin" && b.kind == "body":
				bd, p := a, b
				if a.kind == "pin" {
					bd, p = b, a
				}
				if segHit(p.a, p.z, bd.b.shrink(tol)) {
					add("вывод %s заходит в корпус %s", p.owner, bd.owner)
				}
			}
		}
		// провода и шины: сквозь корпуса и надписи
		if a.kind == "body" {
			for _, w := range append(append([][2]Pt{}, wires...), buses...) {
				// и вплотную вдоль края: провод ближе 0,6 мм к чужому корпусу читается как «пересечение УГО» (замечание Михалина: DD3.1)
				if segHit(w[0], w[1], a.b.shrink(-0.6)) {
					add("провод/шина (%.2f,%.2f)–(%.2f,%.2f) проходит через корпус %s", w[0].X, w[0].Y, w[1].X, w[1].Y, a.owner)
				}
			}
		}
		if a.kind == "text" {
			// метка стоит на своём проводе: нижние 0,3 мм прямоугольника не проверяем
			tb := a.b.shrink(-0.15) // надпись не вплотную к линиям (замечание Гольцова: «надписи налезают на линии»)
			if strings.HasPrefix(a.owner, "метка ") {
				tb.y1 -= 0.5 // метка стоит на своём проводе
			}
			for _, w := range wires {
				if a.at != nil && (a.at.eq(w[0]) || a.at.eq(w[1]) || onSegInner(w[0], w[1], *a.at)) {
					continue // свой провод
				}
				if segHit(w[0], w[1], tb) {
					add("%s (%s) перечёркнута проводом (%.2f,%.2f)–(%.2f,%.2f)", a.what, a.owner, w[0].X, w[0].Y, w[1].X, w[1].Y)
				}
			}
			for _, w := range buses {
				if segHit(w[0], w[1], tb) {
					add("%s (%s) перечёркнута шиной", a.what, a.owner)
				}
			}
		}
	}
	// вход в шину: провода с двух сторон в одной точке шины выглядят как пересечение шины (замечание Гольцова: «A15–AD4 … пересекают её»)
	type entry struct{ on, w Pt }
	var ents []entry
	for _, e := range root.All("bus_entry") {
		at, sz := e.Find("at"), e.Find("size")
		p0 := Pt{at.Num(0), at.Num(1)}
		p1 := p0.Add(sz.Num(0), sz.Num(1))
		onBus := func(p Pt) bool {
			for _, b := range buses {
				if p.eq(b[0]) || p.eq(b[1]) || onSegInner(b[0], b[1], p) {
					return true
				}
			}
			return false
		}
		switch {
		case onBus(p0):
			ents = append(ents, entry{p0, p1})
		case onBus(p1):
			ents = append(ents, entry{p1, p0})
		}
	}
	for i := range ents {
		for j := i + 1; j < len(ents); j++ {
			a, b := ents[i], ents[j]
			same := math.Abs(a.on.X-b.on.X) < 0.01 && math.Abs(a.on.Y-b.on.Y) < 1.0 || math.Abs(a.on.Y-b.on.Y) < 0.01 && math.Abs(a.on.X-b.on.X) < 1.0
			opposite := (a.w.X-a.on.X)*(b.w.X-b.on.X) < 0 || (a.w.Y-a.on.Y)*(b.w.Y-b.on.Y) < 0
			if same && opposite {
				add("входы в шину с двух сторон напротив друг друга около (%.1f; %.1f) мм — выглядит как пересечение шины", a.on.X, a.on.Y)
			}
		}
	}
	// имя вывода не пересекает линии внутри своего корпуса (разделители полей ГОСТ-УГО; Гольцов: «MPU, RAM перечёркнуты»)
	for _, it := range items {
		if it.kind != "text" || !strings.HasPrefix(it.what, "имя вывода") && !strings.HasPrefix(it.what, "знак ") {
			continue
		}
		for _, g := range glines {
			if g.owner == it.owner && segHit(g.a, g.z, it.b.shrink(0.1)) {
				add("%s (%s) пересекает линию внутри корпуса", it.what, it.owner)
			}
		}
	}
	// знак питания/земли не на чужом корпусе и не на выводе другого элемента
	for _, p := range pbodies {
		for _, it := range items {
			switch {
			case it.kind == "body" && p.b.shrink(0.1).hit(it.b.shrink(0.1)):
				add("знак питания %s на корпусе %s около (%.1f; %.1f) мм", p.owner, it.owner, p.b.x0, p.b.y0)
			case it.kind == "pin" && segHit(it.a, it.z, p.b.shrink(0.1)):
				add("знак питания %s на выводе %s около (%.1f; %.1f) мм", p.owner, it.owner, p.b.x0, p.b.y0)
			}
		}
	}
	// надпись не на знаке питания/земли (подпись DD5.1 под землёй дешифратора — проверка листов 09.10.2026)
	for _, p := range pbodies {
		for _, it := range items {
			if it.kind == "text" && it.owner != p.owner && it.b.shrink(0.1).hit(p.b) { // касание номера вывода у конца вывода — норма KiCad
				add("%s (%s) на знаке питания около (%.1f; %.1f) мм", it.what, it.owner, p.b.x0, p.b.y0)
			}
		}
	}
	// знак питания/земли не перечёркнут шиной
	for _, p := range pbodies {
		for _, w := range buses {
			if segHit(w[0], w[1], p.b.shrink(-0.3)) {
				add("шина проходит через знак питания %s около (%.1f; %.1f) мм", p.owner, p.b.x0, p.b.y0)
			}
		}
	}
	sort.Strings(out)
	return dedup(out), nil
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i] + "…"
	}
	return s
}

func dedup(s []string) []string {
	var r []string
	for i, x := range s {
		if i == 0 || x != s[i-1] {
			r = append(r, x)
		}
	}
	return r
}
