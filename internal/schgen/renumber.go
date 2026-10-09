package schgen

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Renumber присваивает позиционные обозначения по ГОСТ 2.710: в пределах буквенного
// кода — по расположению на листе сверху вниз, столбцами слева направо.
// Столбец — элементы, перекрывающиеся по горизонтали (по корпусам, без ножек выводов).
// Части одного корпуса (DD3.1, DD3.2) получают номер по первой встреченной части.
func (s *Sheet) Renumber() {
	s.numCols = map[string][][]colItem{}
	groups := map[string][]*Comp{}
	for _, c := range s.syms {
		if strings.HasPrefix(c.Ref, "#") {
			continue
		}
		p := refPrefix(c.Ref)
		groups[p] = append(groups[p], c)
	}
	for prefix, cs := range groups {
		type box struct {
			c         *Comp
			x0, x1, y float64
		}
		var bs []box
		for _, c := range cs {
			x0, x1 := c.At.X, c.At.X
			for _, p := range c.pins {
				x0, x1 = min(x0, p.X), max(x1, p.X)
			}
			// корпус без ножек выводов: столбцы — по телам элементов, а не по концам выводов (иначе цепочки перекрытий
			// сливают в один «столбец» соседние узлы, и порядок номеров глазу неочевиден)
			if x1-x0 > 6 {
				x0, x1 = x0+2.54, x1-2.54
			}
			bs = append(bs, box{c, x0 - 0.5, x1 + 0.5, c.At.Y})
		}
		// столбец — элементы с одним центром по горизонтали (±2,54 мм); столбцы — слева направо по центру, в столбце — сверху вниз.
		// Так читает руководитель: «у вас принято „сверху вниз и слева направо“, тогда DD2 — это должно быть DD1» (Михалин 08.10.2026:
		// регистр столбцов левее МК — отдельный, более левый столбец, хотя корпуса перекрываются по горизонтали).
		cx := func(b box) float64 { return (b.x0 + b.x1) / 2 }
		sort.SliceStable(bs, func(i, j int) bool { return cx(bs[i]) < cx(bs[j]) })
		var cols [][]box
		for _, b := range bs {
			if len(cols) == 0 || cx(b)-cx(cols[len(cols)-1][0]) > 2.54 {
				cols = append(cols, nil)
			}
			cols[len(cols)-1] = append(cols[len(cols)-1], b)
		}
		s.numCols[prefix] = nil
		for _, col := range cols {
			var refs []colItem
			for _, b := range col {
				refs = append(refs, colItem{b.c, b.x0, b.x1, b.y})
			}
			s.numCols[prefix] = append(s.numCols[prefix], refs)
		}
		newRef := map[string]string{}
		n := 0
		for _, col := range cols {
			sort.SliceStable(col, func(i, j int) bool { return col[i].y < col[j].y })
			for _, b := range col {
				if _, ok := newRef[b.c.Ref]; !ok {
					n++
					newRef[b.c.Ref] = fmt.Sprintf("%s%d", prefix, n)
				}
			}
		}
		for _, c := range cs {
			c.setRef(newRef[c.Ref])
		}
	}
}

type colItem struct {
	c         *Comp
	x0, x1, y float64
}

// NumberingDoubts — места, где порядок «по столбцам» неочевиден глазу: соседние столбцы почти касаются (зазор < 2,54 мм),
// или в столбце, слитом по цепочке перекрытий, нижний элемент правее и не перекрывается с верхним, а верхний левее.
// Такие места правят раскладкой (элементы разносят), а не правилом.
func (s *Sheet) NumberingDoubts() []string {
	var out []string
	cx := func(b colItem) float64 { return (b.x0 + b.x1) / 2 }
	// порядок при «широких» столбцах: центры ближе 7,62 мм — один столбец (так может прочитать глаз)
	order := func(items []colItem, tol float64) []string {
		sort.SliceStable(items, func(i, j int) bool { return cx(items[i]) < cx(items[j]) })
		var cols [][]colItem
		for _, it := range items {
			if len(cols) == 0 || cx(it)-cx(cols[len(cols)-1][len(cols[len(cols)-1])-1]) > tol {
				cols = append(cols, nil)
			}
			cols[len(cols)-1] = append(cols[len(cols)-1], it)
		}
		var refs []string
		seen := map[string]bool{}
		for _, col := range cols {
			sort.SliceStable(col, func(i, j int) bool { return col[i].y < col[j].y })
			for _, it := range col {
				if !seen[it.c.Ref] {
					seen[it.c.Ref] = true
					refs = append(refs, it.c.Ref)
				}
			}
		}
		return refs
	}
	for prefix, cols := range s.numCols {
		var all []colItem
		for _, c := range cols {
			all = append(all, c...)
		}
		a := order(append([]colItem(nil), all...), 2.54)
		b := order(append([]colItem(nil), all...), 7.62)
		for i := range a {
			if a[i] != b[i] {
				out = append(out, fmt.Sprintf("%s: номера зависят от того, как читать столбцы: %v или %v", prefix, a, b))
				break
			}
		}
	}
	sort.Strings(out)
	return dedup(out)
}

func (c *Comp) setRef(ref string) {
	c.Ref = ref
	c.node.Walk(func(n *Node) {
		switch {
		case n.Head() == "property" && n.Arg(0) == "Reference":
			n.Kids[2] = Q(ref)
		case n.Head() == "reference":
			n.Kids[1] = Q(ref)
		}
	})
}

func refPrefix(ref string) string { return strings.TrimRight(ref, "0123456789") }

func refNum(ref string) int {
	n, _ := strconv.Atoi(ref[len(refPrefix(ref)):])
	return n
}
