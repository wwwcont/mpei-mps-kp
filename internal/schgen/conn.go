package schgen

import "fmt"

// Размеры разъёма-таблицы (как в принятых схемах).
const (
	connRow  = 3.81 // высота строки
	connSigW = 12.7 // ширина графы с именем цепи
	connNumW = 7.62 // ширина графы с номером контакта
	connPin  = 2.54 // длина вывода слева
	connFont = 1.27
)

// ConnPin — строка разъёма.
type ConnPin struct {
	Signal string // имя цепи (можно с ~{…})
	Power  bool   // вывод питания разъёма (power_out: цепь запитана отсюда, PWR_FLAG не нужен)
}

// ConnStyle — вид таблицы разъёма.
type ConnStyle struct {
	NameHdr, NumHdr string // заголовки граф: «Сигнал»/«Вывод», «Цепь»/«№», «Цепь»/«Конт.»
	NumFirst        bool   // графа номера слева (Контакт | Цепь), иначе справа (Сигнал | Вывод)
}

var (
	ConnSignalPin = ConnStyle{NameHdr: "Сигнал", NumHdr: "Вывод"}               // принятая Ивана 2025
	ConnNumNet    = ConnStyle{NameHdr: "Цепь", NumHdr: "Конт.", NumFirst: true} // Осипова 2025 («№» — нет знака в шрифте листа, выходил квадрат)
	ConnContNet   = ConnStyle{NameHdr: "Цепь", NumHdr: "Конт.", NumFirst: true} // методичка Прил. А, Visio-работы
)

// AddConn регистрирует разъём-таблицу name. units — части разъёма (XS1.1, XS1.2… при нескольких),
// нумерация контактов сквозная. Точка подключения контакта i части u: (0, -(i)*connRow) от точки
// привязки части, вход слева.
func (l *Lib) AddConn(name string, units [][]ConnPin, cs ConnStyle) {
	stroke := func() *Node { return L("stroke", L("width", F(0)), L("type", A("default"))) }
	nofill := func() *Node { return L("fill", L("type", A("none"))) }
	x0 := connPin
	w := connSigW + connNumW
	xNum, xName := x0+connSigW, x0 // левый край графы номера / имени
	if cs.NumFirst {
		xNum, xName = x0, x0+connNumW
	}
	sym := L("symbol", Q(name),
		L("pin_numbers", L("hide", A("yes"))),
		L("pin_names", L("offset", F(0)), L("hide", A("yes"))),
		L("exclude_from_sim", A("no")), L("in_bom", A("yes")), L("on_board", A("yes")))
	prop := func(k, v string, x, y float64, hide bool) *Node {
		p := L("property", Q(k), Q(v), L("at", F(x), F(y), F(0)))
		if hide {
			p.Kids = append(p.Kids, L("hide", A("yes")))
		}
		p.Kids = append(p.Kids, L("effects", L("font", L("size", F(1.27), F(1.27)))))
		return p
	}
	top := connRow * 1.5
	sym.Kids = append(sym.Kids,
		prop("Reference", "XS", x0+w/2, top+1.27, false),
		prop("Value", name, x0, 0, true),
		prop("Footprint", "", 0, 0, true),
		prop("Datasheet", "", 0, 0, true),
		prop("Description", "Разъём", 0, 0, true))
	num := 1
	for u, rows := range units {
		n := len(rows)
		bottom := -(float64(n-1)*connRow + connRow/2)
		sub := L("symbol", Q(fmt.Sprintf("%s_%d_1", name, u+1)),
			L("rectangle", L("start", F(x0), F(top)), L("end", F(x0+w), F(bottom)), stroke(), nofill()))
		line := func(x1, y1, x2, y2 float64) {
			sub.Kids = append(sub.Kids, L("polyline", L("pts", L("xy", F(x1), F(y1)), L("xy", F(x2), F(y2))), stroke(), nofill()))
		}
		text := func(s string, x, y float64, just string) {
			eff := L("effects", L("font", L("size", F(connFont), F(connFont))))
			if just != "" {
				eff.Kids = append(eff.Kids, L("justify", A(just)))
			}
			sub.Kids = append(sub.Kids, L("text", Q(s), L("at", F(x), F(y), F(0)), eff))
		}
		sep := x0 + connSigW
		if cs.NumFirst {
			sep = x0 + connNumW
		}
		line(sep, top, sep, bottom)
		for i := 0; i < n; i++ {
			y := connRow/2 - float64(i)*connRow
			line(x0, y, x0+w, y)
		}
		nameW := connSigW
		text(cs.NameHdr, xName+nameW/2, connRow, "")
		text(cs.NumHdr, xNum+connNumW/2, connRow, "")
		for i, r := range rows {
			y := -float64(i) * connRow
			text(r.Signal, xName+0.635, y, "left")
			text(fmt.Sprint(num), xNum+connNumW/2, y, "")
			typ := "passive"
			if r.Power {
				typ = "power_out"
			}
			sub.Kids = append(sub.Kids, L("pin", A(typ), A("line"),
				L("at", F(0), F(y), F(0)), L("length", F(connPin)),
				L("name", Q(r.Signal), L("effects", L("font", L("size", F(connFont), F(connFont))))),
				L("number", Q(fmt.Sprint(num)), L("effects", L("font", L("size", F(connFont), F(connFont)))))))
			num++
		}
		sym.Kids = append(sym.Kids, sub)
	}
	l.Syms[name] = sym
}
