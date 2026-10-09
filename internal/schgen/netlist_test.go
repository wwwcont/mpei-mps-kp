package schgen

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// Проверка электрики по netlist из kicad-cli: каждая ключевая цепь состоит ровно
// из ожидаемых выводов. ERC такие ошибки (слияние цепей, КЗ резистора) не ловит.
// Без kicad-cli тест пропускается.

var variants = []Variant{
	// А-12 M=14: 4×3, ОК
	{Cols: 4, Rows: 3, CS: CSPins{Buf: "P2.5", Y2: "P2.4", Kb: "P2.6", Ind: "P2.7"}, Y1: "P1.0", Y2: "P1.1", KbInt: "INT0", X2Int: "INT1"},
	// А-17 M=21: 3×4, INT клавиатуры на INT1
	{Cols: 3, Rows: 4, CS: CSPins{Buf: "P2.4", Y2: "P2.3", Kb: "P2.5", Ind: "P2.7"}, Y1: "P1.0", Y2: "P1.1", KbInt: "INT1", X2Int: "INT0"},
	// общий анод, стробы на краю P1
	{Cols: 4, Rows: 3, Anode: true, CS: CSPins{Buf: "P2.3", Y2: "P2.7", Kb: "P2.4", Ind: "P2.5"}, Y1: "P1.6", Y2: "P1.7", KbInt: "INT0", X2Int: "INT1"},
	// ТЗ-2026: А-12 M=14 — CS с дешифратора, общий анод
	{Cols: 4, Rows: 3, Anode: true, Decoder: true, CSEn: "P3.4", Filter: "33 н",
		CS: CSPins{Buf: "Y5", Y2: "Y6", Kb: "Y1", Ind: "Y7"}, Y1: "P1.0", Y2: "P1.1", KbInt: "INT0", X2Int: "INT1"},
	// ТЗ-2026: А-17 M=21 — 3×4, Y2/индикатор на Y1/Y7
	{Cols: 3, Rows: 4, Decoder: true, CSEn: "P3.4", Filter: "33 н",
		CS: CSPins{Buf: "Y6", Y2: "Y1", Kb: "Y2", Ind: "Y7"}, Y1: "P1.0", Y2: "P1.1", KbInt: "INT0", X2Int: "INT1"},
}

func TestNetlist(t *testing.T) {
	cli, err := exec.LookPath("kicad-cli")
	if err != nil {
		t.Skip("нет kicad-cli")
	}
	lib, err := LoadLib("../../masters/lib/mps.kicad_sym")
	if err != nil {
		t.Fatal(err)
	}
	for i, v := range variants {
		// A–D и три «авто»-сочетания признаков (MixStyle) — связи не должны зависеть ни от стиля, ни от «почерка»
		for _, st := range []string{"A", "B", "C", "D", "mix1", "mix2", "mix3"} {
			v.Style = PickStyle(st, fmt.Sprintf("тест|%d|%s", i, st))
			v.Jitter = nil
			if st != "A" && st != "B" {
				j := MakeJitter(fmt.Sprintf("тест|%d|%s", i, st), v.Rows)
				v.Jitter = &j
			}
			t.Run(fmt.Sprintf("%dx%d-%d-%s", v.Cols, v.Rows, i, st), func(t *testing.T) {
				dir := t.TempDir()
				sch := filepath.Join(dir, "schematic.kicad_sch")
				sh := Build(lib, v, "test")
				if err := os.WriteFile(sch, []byte(sh.String()), 0o644); err != nil {
					t.Fatal(err)
				}
				os.WriteFile(filepath.Join(dir, "schematic.kicad_pro"), []byte(Project), 0o644)
				net := filepath.Join(dir, "n.net")
				if out, err := exec.Command(cli, "sch", "export", "netlist", "-o", net, sch).CombinedOutput(); err != nil {
					t.Fatalf("%v: %s", err, out)
				}
				// ERC: любое нарушение — ошибка (правила, которые для учебной схемы не ошибка, выключены в Project)
				rpt := filepath.Join(dir, "erc.rpt")
				if out, err := exec.Command(cli, "sch", "erc", "--exit-code-violations", "-o", rpt, sch).CombinedOutput(); err != nil {
					b, _ := os.ReadFile(rpt)
					t.Errorf("ERC: %v\n%s\n%s", err, out, b)
				}
				nets := readNets(t, net)
				for name, want := range expected(v, sh.Roles) {
					got := nets.byPin[want[0]]
					if strings.Join(nets.members[got], " ") != strings.Join(sorted(want), " ") {
						t.Errorf("цепь %s:\n  ждём %v\n  есть %v (%s)", name, sorted(want), nets.members[got], got)
					}
				}
				// IDT7005: выводы на питании (R/WL, OER, SEM, BUSY, M/S)
				idt := sh.Roles["idt"].Ref
				for pin, want := range idtFixed(v) {
					if got := nets.byPin[fmt.Sprintf("%s.%d", idt, pin)]; got != want {
						t.Errorf("IDT7005 вывод %d: ждём %s, есть %s", pin, want, got)
					}
				}
				// ни один вывод микросхем не висит в цепи из одного вывода; свободны только выводы из списка NC
				nc := map[[2]int64]bool{}
				for _, p := range sh.NC {
					nc[[2]int64{int64(p.X*100 + 0.5), int64(p.Y*100 + 0.5)}] = true
				}
				for n, m := range nets.members {
					if len(m) == 1 && !strings.HasPrefix(n, "unconnected-") {
						t.Errorf("цепь %s из одного вывода %v", n, m)
					}
					if strings.HasPrefix(n, "unconnected-") {
						for _, rp := range m {
							if p, ok := sh.pinAt(rp); !ok || !nc[[2]int64{int64(p.X*100 + 0.5), int64(p.Y*100 + 0.5)}] {
								t.Errorf("вывод %s не подключён, а в списке свободных (NC) его нет", rp)
							}
						}
					}
				}
			})
		}
	}
}

// idtFixed — выводы IDT7005 на питании: правый порт выбран всегда (CER = 0) и только пишет (OER = 1);
// slave (M/S = 0), BUSY и SEM не используются. R/WL ← WR — в цепи WR.
func idtFixed(v Variant) map[int]string {
	m := map[int]string{22: "GND", 19: "+5V", 21: "+5V", 39: "+5V", 42: "+5V", 60: "+5V", 40: "GND"}
	if !v.Decoder { // ТЗ-2025: окно 2 КБ, A11/A12 обоих портов на земле
		m[26], m[25], m[55], m[56] = "GND", "GND", "GND", "GND"
	}
	return m
}

type netlist struct {
	byPin   map[string]string   // "DD3.9" → имя цепи
	members map[string][]string // имя цепи → выводы
}

func readNets(t *testing.T, path string) netlist {
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	root, err := Parse(string(b))
	if err != nil {
		t.Fatal(err)
	}
	n := netlist{byPin: map[string]string{}, members: map[string][]string{}}
	for _, nt := range root.Find("nets").All("net") {
		name := nt.Find("name").Arg(0)
		for _, nd := range nt.All("node") {
			p := nd.Find("ref").Arg(0) + "." + nd.Find("pin").Arg(0)
			n.byPin[p] = name
			n.members[name] = append(n.members[name], p)
		}
		sort.Strings(n.members[name])
	}
	return n
}

func sorted(s []string) []string {
	c := append([]string(nil), s...)
	sort.Strings(c)
	return c
}

// expected — ожидаемый состав ключевых цепей; элементы — по ролям (нумерация по ГОСТ считается сама).
func expected(v Variant, roles map[string]*Comp) map[string][]string {
	r := func(role string, pin any) string {
		c, ok := roles[role]
		if !ok {
			panic("нет роли " + role)
		}
		return fmt.Sprintf("%s.%v", c.Ref, pin)
	}
	mp := func(p string) string { return r("mcu", mcuPin(p)) }
	rc := func(role string, k int) string { return r(role, roles[role].PinBase+k) }
	e := map[string][]string{}
	dl := []string{"63", "64", "1", "2", "3", "4", "6", "7"}
	dr := []string{"10", "11", "12", "14", "15", "16", "17", "18"}
	outs244 := []int{18, 16, 14, 12}
	in244 := []int{2, 4, 6, 8}
	for i := 0; i < 8; i++ {
		m := []string{r("mcu", 39-i), r("latchA", 2+i), r("latchY2", 2+i), r("idt", dl[i]), r("latchInd", 2+i)}
		if i < v.Rows {
			m = append(m, r("buf", outs244[i]))
		}
		if i < 4 {
			m = append(m, r("kb173", 14-i))
		}
		e[fmt.Sprintf("AD%d", i)] = m
		e[fmt.Sprintf("A%d", i)] = []string{r("latchA", 19-i), r("idt", 44+i)}
		e[fmt.Sprintf("X2_A%d", i)] = []string{r("idt", 37-i), rc("xsX2", 10+i)}
		e[fmt.Sprintf("X2_%d", i)] = []string{r("idt", dr[i]), rc("xsX2", 2+i)}
		e[fmt.Sprintf("Y2_%d", i)] = []string{r("latchY2", 19-i), rc("xsY", 3+i)}
	}
	for i := 8; i <= 10; i++ {
		e[fmt.Sprintf("A%d", i)] = []string{r("mcu", 13+i), r("idt", 44+i)}
		e[fmt.Sprintf("X2_A%d", i)] = []string{r("idt", 37-i), rc("xsX2", 10+i)}
	}
	if v.Decoder {
		// ТЗ-2026: P2 целиком — адрес; A11, A12 — на оба порта IDT; A13..A15 — на дешифратор
		e["A11"] = []string{r("mcu", 24), r("idt", 55)}
		e["A12"] = []string{r("mcu", 25), r("idt", 56)}
		e["X2_A11"] = []string{r("idt", 26), rc("xsX2", 21)}
		e["X2_A12"] = []string{r("idt", 25), rc("xsX2", 22)}
		for i := 13; i <= 15; i++ {
			e[fmt.Sprintf("A%d", i)] = []string{r("mcu", 13+i), r("dec", i-12)}
		}
		e["CS_EN"] = []string{mp(v.CSEn), r("dec", 6)}
		yPin := []string{"15", "14", "13", "12", "11", "10", "9", "7"}
		mp = func(y string) string { return r("dec", yPin[y[1]-'0']) }
	}
	e["CSbuf"] = []string{mp(v.CS.Buf), r("idt", 59)}
	e["CSy2"] = []string{mp(v.CS.Y2), r("norY2", 2)}
	e["CSkb"] = []string{mp(v.CS.Kb), r("kb173", 9), r("kb173", 10), r("or", 1)}
	e["CSind"] = []string{mp(v.CS.Ind), r("norInd", 5)}
	e["WR"] = []string{r("mcu", 16), r("norY2", 3), r("norInd", 6), r("kb173", 7), r("idt", 61)}
	e["RD"] = []string{r("mcu", 17), r("idt", 62), r("or", 2)}
	e["ALE"] = []string{r("mcu", 30), r("latchA", 11)}
	e["Y1"] = []string{r("mcu", mcuPin(v.Y1)), rc("xsY", 1)}
	e["Y2"] = []string{r("mcu", mcuPin(v.Y2)), rc("xsY", 2)}
	intPin := map[string]string{"INT0": "12", "INT1": "13"}
	andOut := "12"
	andIn := []string{"1", "2", "13"}
	if v.Rows == 4 {
		andOut, andIn = "6", []string{"1", "2", "4", "5"}
	}
	e["INTkb"] = []string{r("mcu", intPin[v.KbInt]), r("and", andOut)}
	// строб X2 внешнего устройства: запись правым портом (R/WR) и прерывание МК
	e["INTx2"] = []string{r("mcu", intPin[v.X2Int]), rc("xsX2", 1), r("idt", 20)}
	e["OE244"] = []string{r("or", 3), r("buf", 1), r("buf", 19)}
	e["LoadY2"] = []string{r("norY2", 1), r("latchY2", 11)}
	e["LoadInd"] = []string{r("norInd", 4), r("latchInd", 11)}
	e["RST"] = []string{r("mcu", 9), r("cRst", 2), r("rRst", 1), r("kb173", 15)}
	e["XTAL1"] = []string{r("mcu", 19), r("zq", 2), r("cX1", 2)}
	e["XTAL2"] = []string{r("mcu", 18), r("zq", 1), r("cX2", 2)}

	for row := 0; row < v.Rows; row++ {
		e[fmt.Sprintf("Row%d", row+1)] = []string{r(fmt.Sprintf("ser%d", row), 2), r(fmt.Sprintf("ck%d", row), 1), r("buf", in244[row]), r("and", andIn[row])}
		line := []string{r(fmt.Sprintf("pull%d", row), 2), r(fmt.Sprintf("ser%d", row), 1)}
		for c := 0; c < v.Cols; c++ {
			line = append(line, r(fmt.Sprintf("sb%d_%d", c, row), 2))
		}
		e[fmt.Sprintf("RowLine%d", row+1)] = line
	}
	for c := 0; c < v.Cols; c++ {
		e[fmt.Sprintf("Col%d", c+1)] = []string{r("kb173", 3+c), r(fmt.Sprintf("vd%d", c), 1)}
		col := []string{r(fmt.Sprintf("vd%d", c), 2)}
		for row := 0; row < v.Rows; row++ {
			col = append(col, r(fmt.Sprintf("sb%d_%d", c, row), 1))
		}
		e[fmt.Sprintf("ColLine%d", c+1)] = col
	}
	seg := []string{"7", "6", "4", "2", "1", "9", "10", "5"}
	for i := 0; i < 8; i++ {
		rr := fmt.Sprintf("rseg%d", i)
		e[fmt.Sprintf("Q%d", i)] = []string{r("latchInd", 19-i), r(rr, 1)}
		e[fmt.Sprintf("seg%d", i)] = []string{r(rr, 2), r("hg", seg[i])}
	}
	return e
}

// pinAt — положение вывода «DD3.9» на листе.
func (s *Sheet) pinAt(refPin string) (Pt, bool) {
	i := strings.LastIndexByte(refPin, '.')
	ref, num := refPin[:i], refPin[i+1:]
	for _, c := range s.syms {
		if c.Ref == ref {
			if p, ok := c.pins[num]; ok {
				return p, true
			}
		}
	}
	return Pt{}, false
}
