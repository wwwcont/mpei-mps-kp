package schgen

import (
	"crypto/sha1"
	"fmt"
	"sort"
	"strings"
)

const libName = "mps"

// labelFont — шрифт меток цепей. При шаге выводов 2,54 мм и шрифте 1,27 черта
// над ~{CS…} ложится на провод строкой выше и на печати пропадает.
const labelFont = 1.1

// Sheet — лист схемы, который собирается вызовами Sym/Wire/Label/...
type Sheet struct {
	LabelAtPin map[string]bool // имена, которые busMarks ставит у вывода, а не у шины (у шины место занято)
	lib        *Lib
	used       map[string]*Node // lib_id → символ для lib_symbols
	items      []*Node
	wires      [][2]Pt
	pinPts     []Pt
	syms       []*Comp
	seed       string
	n          int
	pwr        int
	Title      TitleBlock
	rootID     string
	project    string
	Roles      map[string]*Comp // роль → элемент (заполняет построитель)
	buses      [][2]Pt
	entries    [][2]Pt
	perp       bool
	A4         bool                   // лист А4 книжный (перечень элементов), иначе А3 альбомный
	J          Jitter                 // «почерк» листа (jitter.go); нулевой — как Plain
	Fixes      *Fixes                 // правки студента (fixes.go); nil — нет
	FixErrs    []string               // ошибки в правках (неизвестные обозначения и т.п.) — mpsgen падает с ними
	NC         []Pt                   // выводы, свободные намеренно (NoConnect)
	numCols    map[string][][]colItem // столбцы нумерации (Renumber) — для проверки NumberingDoubts
}

type TitleBlock struct {
	Date     string
	Comments map[int]string
}

func NewSheet(lib *Lib, seed string) *Sheet {
	s := &Sheet{lib: lib, used: map[string]*Node{}, seed: seed, project: "schematic"}
	s.rootID = s.uuid()
	return s
}

// uuid — детерминированный (одинаковый вход → одинаковый файл, удобно сравнивать).
func (s *Sheet) uuid() string {
	s.n++
	h := sha1.Sum([]byte(fmt.Sprintf("%s/%d", s.seed, s.n)))
	h[6] = h[6]&0x0f | 0x50
	h[8] = h[8]&0x3f | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", h[0:4], h[4:6], h[6:8], h[8:10], h[10:16])
}

// Comp — размещённый символ (одна часть корпуса).
type Comp struct {
	Ref, Value string
	Sym        string
	At         Pt
	Rot        int
	Unit       int
	PinBase    int // сквозная нумерация контактов у составного разъёма (часть XS1.2 начинается с 11)
	pins       map[string]Pt
	node       *Node
}

// Pin — точка подключения вывода num.
func (c *Comp) Pin(num string) Pt {
	p, ok := c.pins[num]
	if !ok {
		panic(fmt.Sprintf("%s (%s): нет вывода %s в части %d", c.Ref, c.Sym, num, c.Unit))
	}
	return p
}

// SymOpt — необязательные параметры размещения.
type SymOpt struct {
	Rot      int
	Unit     int
	RefAt    *Pt // абсолютное положение надписи позиционного обозначения
	ValAt    *Pt
	HideVal  bool
	HideRef  bool
	RefJust  string // left | right | ""
	ValJust  string
	Footnote string
}

// Sym ставит символ sym (имя в mps.kicad_sym) с обозначением ref.
func (s *Sheet) Sym(sym, ref, value string, at Pt, o SymOpt) *Comp {
	def, ok := s.lib.Syms[sym]
	if !ok {
		panic("нет символа " + sym)
	}
	if o.Unit == 0 {
		o.Unit = 1
	}
	libID := libName + ":" + sym
	if _, ok := s.used[libID]; !ok {
		d := def.Clone()
		d.Kids[1] = Q(libID)
		s.used[libID] = d
	}
	c := &Comp{Ref: ref, Value: value, Sym: sym, At: at, Rot: o.Rot, Unit: o.Unit, pins: map[string]Pt{}}
	n := L("symbol",
		L("lib_id", Q(libID)),
		L("at", F(at.X), F(at.Y), F(float64(o.Rot))),
		L("unit", F(float64(o.Unit))),
		L("exclude_from_sim", A("no")),
		L("in_bom", A(yn(!strings.HasPrefix(ref, "#")))),
		L("on_board", A(yn(!strings.HasPrefix(ref, "#")))),
		L("dnp", A("no")),
		L("uuid", Q(s.uuid())),
	)
	for _, p := range def.All("property") {
		key := p.Arg(0)
		val := p.Arg(1)
		pat := p.Find("at")
		pos := xform(at, 0, pat.Num(0), pat.Num(1))
		if o.Rot != 0 {
			pos = xform(at, o.Rot, pat.Num(0), pat.Num(1))
		}
		hide := p.Find("hide") != nil && p.Find("hide").Arg(0) == "yes"
		just := ""
		if ef := p.Find("effects"); ef != nil {
			if j := ef.Find("justify"); j != nil {
				just = j.Arg(0)
			}
		}
		switch key {
		case "Reference":
			val = ref
			if o.RefAt != nil {
				pos = *o.RefAt
			}
			hide = o.HideRef || strings.HasPrefix(ref, "#")
			if o.RefJust != "" {
				just = o.RefJust
			}
		case "Value":
			val = value
			if o.ValAt != nil {
				pos = *o.ValAt
			}
			hide = o.HideVal
			if o.ValJust != "" {
				just = o.ValJust
			}
		case "Footprint", "Datasheet", "Description":
			hide = true
		default:
			if strings.HasPrefix(key, "ki_") {
				continue
			}
			hide = true
		}
		pr := L("property", Q(key), Q(val), L("at", F(pos.X), F(pos.Y), F(float64(o.Rot%180))))
		if hide {
			pr.Kids = append(pr.Kids, L("hide", A("yes")))
		}
		// у символа, повёрнутого на 90°, KiCad зеркалит выравнивание подписи (на 270° — нет: проверено по PNG, VD1–VD3)
		if (o.Rot%360+360)%360 == 90 {
			switch just {
			case "left":
				just = "right"
			case "right":
				just = "left"
			}
		}
		fs := 1.27 // мелкие элементы — всегда 1,27 (иначе подписи налезают на соседей)
		if p := refPrefix(ref); p == "DD" || p == "XS" || p == "HG" {
			fs = s.refFont()
		}
		eff := L("effects", L("font", L("size", F(fs), F(fs))))
		if just != "" && just != "center" {
			eff.Kids = append(eff.Kids, L("justify", A(just)))
		}
		pr.Kids = append(pr.Kids, eff)
		n.Kids = append(n.Kids, pr)
	}
	for _, p := range Pins(def) {
		if p.Unit != 0 && p.Unit != o.Unit {
			continue
		}
		pt := xform(at, o.Rot, p.X, p.Y)
		c.pins[p.Num] = pt
		if !p.Hidden {
			s.pinPts = append(s.pinPts, pt)
		}
		n.Kids = append(n.Kids, L("pin", Q(p.Num), L("uuid", Q(s.uuid()))))
	}
	n.Kids = append(n.Kids, L("instances",
		L("project", Q(s.project),
			L("path", Q("/"+s.rootID),
				L("reference", Q(ref)),
				L("unit", F(float64(o.Unit)))))))
	c.node = n
	s.items = append(s.items, n)
	s.syms = append(s.syms, c)
	return c
}

func yn(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

// Power ставит символ питания (+5V / GND) в точку at.
func (s *Sheet) Power(kind string, at Pt, rot int) *Comp {
	s.pwr++
	o := SymOpt{Rot: rot, HideVal: kind == "GND"}
	// знак питания вбок: надпись за концом стрелки, а не на ней («+» прятался под стрелкой — проверка листов 09.10.2026),
	// и над проводом, а не посередине: иначе она свисает в промежуток к соседнему ряду и задевает надчёркивание метки под ней
	switch (rot%360 + 360) % 360 {
	case 270: // стрелка вправо
		o.ValAt, o.ValJust = ptr2(at.Add(3.302, -0.95)), "left"
	case 90: // стрелка влево
		o.ValAt, o.ValJust = ptr2(at.Add(-3.302, -0.95)), "right"
	}
	return s.Sym(kind, fmt.Sprintf("#PWR%02d", s.pwr), kind, at, o)
}

// Wire — ломаная из отрезков.
func (s *Sheet) Wire(pts ...Pt) {
	for i := 0; i+1 < len(pts); i++ {
		a, b := pts[i], pts[i+1]
		if a.eq(b) {
			continue
		}
		s.wires = append(s.wires, [2]Pt{a, b})
	}
}

// splitWires режет провода в точках, где на середину провода попадает вывод или
// конец другого провода: KiCad не считает вывод в середине провода подключённым.
func (s *Sheet) splitWires() {
	var cuts []Pt
	cuts = append(cuts, s.pinPts...)
	for _, w := range s.wires {
		cuts = append(cuts, w[0], w[1])
	}
	var out [][2]Pt
	for _, w := range s.wires {
		var in []Pt
		for _, c := range cuts {
			if onSegInner(w[0], w[1], c) {
				in = append(in, c)
			}
		}
		sort.Slice(in, func(i, j int) bool {
			return dist(w[0], in[i]) < dist(w[0], in[j])
		})
		prev := w[0]
		for _, c := range in {
			if !c.eq(prev) {
				out = append(out, [2]Pt{prev, c})
				prev = c
			}
		}
		out = append(out, [2]Pt{prev, w[1]})
	}
	s.wires = out
	for _, w := range s.wires {
		s.items = append(s.items, L("wire",
			L("pts", L("xy", F(w[0].X), F(w[0].Y)), L("xy", F(w[1].X), F(w[1].Y))),
			L("stroke", L("width", F(s.J.WireW)), L("type", A("default"))),
			L("uuid", Q(s.uuid()))))
	}
}

func dist(a, b Pt) float64 { return abs(a.X-b.X) + abs(a.Y-b.Y) }

// Bus — ломаная шины.
func (s *Sheet) Bus(pts ...Pt) {
	for i := 0; i+1 < len(pts); i++ {
		s.buses = append(s.buses, [2]Pt{pts[i], pts[i+1]})
	}
}

// BusEntry — наклонный отвод от точки at на (dx, dy) (один конец — на шине, другой — на проводе).
func (s *Sheet) BusEntry(at Pt, dx, dy float64) {
	s.entries = append(s.entries, [2]Pt{at, at.Add(dx, dy)})
}

// Perp — отводы к шине перпендикулярные (Т-образно, 90°) вместо 45°.
func (s *Sheet) SetPerpEntries(on bool) { s.perp = on }

// emitBuses выводит шины и отводы. В режиме 90° конец отвода на шине сдвигается к перпендикуляру
// от конца на проводе, а шина при необходимости удлиняется до нового конца.
func (s *Sheet) emitBuses() {
	onBus := func(p Pt) (int, bool) {
		for i, b := range s.buses {
			if p.eq(b[0]) || p.eq(b[1]) || onSegInner(b[0], b[1], p) {
				return i, true
			}
		}
		return -1, false
	}
	wireEnd := map[[2]int64]bool{}
	key := func(p Pt) [2]int64 { return [2]int64{int64(p.X*100 + 0.5), int64(p.Y*100 + 0.5)} }
	for _, w := range s.wires {
		wireEnd[key(w[0])], wireEnd[key(w[1])] = true, true
	}
	for i, e := range s.entries {
		if !s.perp {
			break
		}
		w, bp := e[0], e[1]
		if !wireEnd[key(w)] {
			w, bp = bp, w
		}
		// шина, к которой отвод встаёт перпендикулярно проводу (на стыке двух шин onBus может вернуть не ту)
		bi, ok := -1, false
		horizWire := true // направление провода, к которому подходит отвод
		for _, ww := range s.wires {
			if ww[0].eq(w) || ww[1].eq(w) {
				horizWire = abs(ww[0].Y-ww[1].Y) < 0.01
				break
			}
		}
		for i, b := range s.buses {
			if !(bp.eq(b[0]) || bp.eq(b[1]) || onSegInner(b[0], b[1], bp)) {
				continue
			}
			vert := abs(b[0].X-b[1].X) < 0.01
			if !ok || vert == horizWire {
				bi, ok = i, true
			}
		}
		if !ok {
			continue
		}
		b := s.buses[bi]
		var nb Pt
		if abs(b[0].X-b[1].X) < 0.01 { // вертикальная шина
			nb = Pt{b[0].X, w.Y}
		} else {
			nb = Pt{w.X, b[0].Y}
		}
		if _, ok := onBus(nb); !ok { // продлить шину до нового конца
			if abs(b[0].X-b[1].X) < 0.01 {
				if abs(nb.Y-b[0].Y) < abs(nb.Y-b[1].Y) {
					s.buses[bi][0] = nb
				} else {
					s.buses[bi][1] = nb
				}
			} else {
				if abs(nb.X-b[0].X) < abs(nb.X-b[1].X) {
					s.buses[bi][0] = nb
				} else {
					s.buses[bi][1] = nb
				}
			}
		}
		s.entries[i] = [2]Pt{w, nb}
	}
	s.busMarks()
	for _, b := range s.buses {
		s.items = append(s.items, L("bus",
			L("pts", L("xy", F(b[0].X), F(b[0].Y)), L("xy", F(b[1].X), F(b[1].Y))),
			L("stroke", L("width", F(s.J.BusW)), L("type", A("default"))),
			L("uuid", Q(s.uuid()))))
	}
	for _, e := range s.entries {
		s.items = append(s.items, L("bus_entry",
			L("at", F(e[0].X), F(e[0].Y)),
			L("size", F(round(e[1].X-e[0].X)), F(round(e[1].Y-e[0].Y))),
			L("stroke", L("width", F(0)), L("type", A("default"))),
			L("uuid", Q(s.uuid()))))
	}
}

// Label — локальная метка цепи. right — текст влево от точки.
func (s *Sheet) Label(name string, at Pt, right bool) {
	ang, just := 0.0, "left"
	if right {
		ang, just = 180, "right"
	}
	s.items = append(s.items, L("label", Q(name),
		L("at", F(at.X), F(at.Y), F(ang)),
		L("effects", s.labelFontExpr(), L("justify", A(just), A("bottom"))),
		L("uuid", Q(s.uuid()))))
}

// VLabel — метка на вертикальном проводе (текст снизу вверх).
func (s *Sheet) VLabel(name string, at Pt) {
	s.items = append(s.items, L("label", Q(name),
		L("at", F(at.X), F(at.Y), F(90)),
		L("effects", s.labelFontExpr(), L("justify", A("left"), A("bottom"))),
		L("uuid", Q(s.uuid()))))
}

// NoConnect — вывод, оставленный свободным намеренно. Крест KiCad на лист не ставим: в ГОСТ его нет, и он перечёркивает
// номер вывода (замечание руководителя 06.10.2026, «пересечения УГО»); ERC «вывод не подключён» выключен в Project,
// а что свободны только эти выводы — проверяет netlist-тест по списку NC.
func (s *Sheet) NoConnect(at Pt) { s.NC = append(s.NC, at) }

func (s *Sheet) junction(at Pt) {
	s.items = append(s.items, L("junction",
		L("at", F(at.X), F(at.Y)), L("diameter", F(s.J.JunctionD)), L("color", F(0), F(0), F(0), F(0)),
		L("uuid", Q(s.uuid()))))
}

// Text — свободный текст (примечания). Строки через \n.
func (s *Sheet) Text(txt string, at Pt, size float64) {
	s.items = append(s.items, L("text", Q(txt),
		L("exclude_from_sim", A("no")),
		L("at", F(at.X), F(at.Y), F(0)),
		L("effects", L("font", L("size", F(size), F(size))), L("justify", A("left"), A("top"))),
		L("uuid", Q(s.uuid()))))
}

// Graphic — произвольный графический примитив листа (полилиния/прямоугольник).
func (s *Sheet) Graphic(n *Node) { s.items = append(s.items, n) }

// autoJunctions ставит точки там, где сходятся ≥3 проводника или провод
// упирается в середину другого провода (ГОСТ: точки на ветвлениях обязательны).
func (s *Sheet) autoJunctions() {
	type key struct{ x, y int64 }
	k := func(p Pt) key { return key{int64(p.X*100 + 0.5), int64(p.Y*100 + 0.5)} }
	deg := map[key]int{}
	pos := map[key]Pt{}
	for _, w := range s.wires {
		for _, p := range w {
			deg[k(p)]++
			pos[k(p)] = p
		}
	}
	for _, p := range s.pinPts {
		if _, ok := deg[k(p)]; ok {
			deg[k(p)]++
		}
	}
	for kk, p := range pos {
		for _, w := range s.wires {
			if onSegInner(w[0], w[1], p) {
				deg[kk] += 2
			}
		}
	}
	var keys []key
	for kk, d := range deg {
		if d >= 3 {
			keys = append(keys, kk)
		}
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].x != keys[j].x {
			return keys[i].x < keys[j].x
		}
		return keys[i].y < keys[j].y
	})
	for _, kk := range keys {
		s.junction(pos[kk])
	}
}

func onSegInner(a, b, p Pt) bool {
	if p.eq(a) || p.eq(b) {
		return false
	}
	const e = 0.01
	if abs(a.X-b.X) < e && abs(p.X-a.X) < e {
		return p.Y > min(a.Y, b.Y)+e && p.Y < max(a.Y, b.Y)-e
	}
	if abs(a.Y-b.Y) < e && abs(p.Y-a.Y) < e {
		return p.X > min(a.X, b.X)+e && p.X < max(a.X, b.X)-e
	}
	return false
}

func abs(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}

// String — готовый файл .kicad_sch.
func (s *Sheet) String() string {
	s.splitWires()
	s.autoJunctions()
	s.emitBuses()
	s.shift()
	root := L("kicad_sch",
		L("version", A("20260306")),
		L("generator", Q("mpsgen")),
		L("generator_version", Q("10.0")),
		L("uuid", Q(s.rootID)),
		s.paperNode(),
	)
	tb := L("title_block")
	if s.Title.Date != "" {
		tb.Kids = append(tb.Kids, L("date", Q(s.Title.Date)))
	}
	var ck []int
	for k := range s.Title.Comments {
		ck = append(ck, k)
	}
	sort.Ints(ck)
	for _, k := range ck {
		tb.Kids = append(tb.Kids, L("comment", F(float64(k)), Q(s.Title.Comments[k])))
	}
	root.Kids = append(root.Kids, tb)
	ls := L("lib_symbols")
	var ids []string
	for id := range s.used {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		ls.Kids = append(ls.Kids, s.used[id])
	}
	root.Kids = append(root.Kids, ls)
	root.Kids = append(root.Kids, s.items...)
	root.Kids = append(root.Kids, L("sheet_instances", L("path", Q("/"), L("page", Q("1")))))
	root.Kids = append(root.Kids, L("embedded_fonts", A("no")))
	return root.String()
}

func (s *Sheet) paperNode() *Node {
	if s.A4 {
		return L("paper", Q("A4"), A("portrait"))
	}
	return L("paper", Q("A3"))
}

func (s *Sheet) refFont() float64 {
	if s.J.RefFont > 0 {
		return s.J.RefFont
	}
	return 1.27
}

func (s *Sheet) labelFont() float64 {
	if s.J.LabelFont > 0 {
		return s.J.LabelFont
	}
	return labelFont
}

// labelFontExpr — (font (size …) [italic]) для меток цепей.
func (s *Sheet) labelFontExpr() *Node {
	f := L("font", L("size", F(s.labelFont()), F(s.labelFont())))
	if s.J.LabelItalic {
		f.Kids = append(f.Kids, A("italic"))
	}
	return f
}

// mark — отметка «что уже есть на листе» для moveSince.
type mark struct{ items, wires, pins, buses, entries, nc int }

func (s *Sheet) mark() mark {
	return mark{len(s.items), len(s.wires), len(s.pinPts), len(s.buses), len(s.entries), len(s.NC)}
}

// moveSince сдвигает всё, что добавлено на лист после m (один блок): узлы, провода, точки выводов, шины — и элементы
// (их положение нужно нумерации и рамкам узлов). Только для блоков, связанных с остальным метками и знаками питания.
func (s *Sheet) moveSince(m mark, dx, dy float64) {
	if dx == 0 && dy == 0 {
		return
	}
	mv := func(p Pt) Pt { return Pt{round(p.X + dx), round(p.Y + dy)} }
	moved := map[*Node]bool{}
	for _, it := range s.items[m.items:] {
		moved[it] = true
		it.Walk(func(n *Node) {
			switch n.Head() {
			case "at", "xy", "start", "end", "mid", "center":
				n.Kids[1] = F(round(n.Num(0) + dx))
				n.Kids[2] = F(round(n.Num(1) + dy))
			}
		})
	}
	for i := m.wires; i < len(s.wires); i++ {
		s.wires[i] = [2]Pt{mv(s.wires[i][0]), mv(s.wires[i][1])}
	}
	for i := m.pins; i < len(s.pinPts); i++ {
		s.pinPts[i] = mv(s.pinPts[i])
	}
	for i := m.buses; i < len(s.buses); i++ {
		s.buses[i] = [2]Pt{mv(s.buses[i][0]), mv(s.buses[i][1])}
	}
	for i := m.nc; i < len(s.NC); i++ {
		s.NC[i] = mv(s.NC[i])
	}
	for i := m.entries; i < len(s.entries); i++ {
		s.entries[i] = [2]Pt{mv(s.entries[i][0]), mv(s.entries[i][1])}
	}
	for _, c := range s.syms {
		if !moved[c.node] {
			continue
		}
		c.At = mv(c.At)
		for k, p := range c.pins {
			c.pins[k] = mv(p)
		}
	}
}

// shift сдвигает весь чертёж (кроме lib_symbols) на J.ShiftX/ShiftY — один раз, после всех расчётов связей.
func (s *Sheet) shift() {
	dx, dy := s.J.ShiftX, s.J.ShiftY
	if dx == 0 && dy == 0 {
		return
	}
	for _, it := range s.items {
		it.Walk(func(n *Node) {
			switch n.Head() {
			case "at", "xy", "start", "end", "mid", "center":
				n.Kids[1] = F(round(n.Num(0) + dx))
				n.Kids[2] = F(round(n.Num(1) + dy))
			}
		})
	}
	s.J.ShiftX, s.J.ShiftY = 0, 0
}

func ptr2(p Pt) *Pt { return &p }
