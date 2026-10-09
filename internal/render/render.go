// Package render превращает посчитанные параметры в файлы для людей и для кода.
package render

import (
	"fmt"
	"strings"

	"mpskp/internal/variant"
)

var H = variant.Hex

func indName(s string) string {
	switch s {
	case "cathode":
		return "общий катод (FYS-5612AX)"
	case "anode":
		return "общий анод (FYS-5612BX)"
	}
	return "НЕ ЗАПОЛНЕНО — смотри табл. 1 ТЗ"
}

// Markdown — сводка варианта: её вставляешь в ПЗ и держишь перед глазами.
func Markdown(p *variant.Params) string {
	var b strings.Builder
	w := func(f string, a ...any) { fmt.Fprintf(&b, f+"\n", a...) }
	w("# Параметры варианта: %s, M = %d (ТЗ %d)", p.Group, p.M, p.Year)
	w("")
	if p.Student != "" {
		w("Студент: %s, руководитель: %s", p.Student, p.Checker)
		w("")
	}
	for _, x := range p.Warnings {
		w("> ⚠ %s", x)
	}
	if len(p.Warnings) > 0 {
		w("")
	}
	w("## Что писать")
	w("")
	w("| Программа | Что |")
	w("| --- | --- |")
	w("| 1 | обработчик IRQ клавиатуры (%s): скан матрицы, код в A или 0FFh |", p.KbInt)
	w("| 2 | %s (M %s) |", p.Prog2, map[bool]string{true: "чётное", false: "нечётное"}[p.M%2 == 0])
	w("| 3 | %s (M mod 3 = %d) |", p.Prog3, p.M%3)
	w("")
	w("## Выводы МК")
	w("")
	w("| Сигнал | Вывод |")
	w("| --- | --- |")
	w("| Строб Y1 (k = M mod 7 = %d) | %s |", p.K, p.Y1Pin)
	w("| Строб Y2 | %s |", p.Y2Pin)
	w("| IRQ клавиатуры | %s (%s), по спаду |", p.KbInt, p.KbIntPin)
	w("| Строб X2 от внешнего устройства | %s (%s), по спаду |", p.X2Int, p.X2IntPin)
	for _, d := range p.Devices {
		w("| CS: %s | %s |", d.Name, d.CS)
	}
	if p.CSMode == "decoder" {
		w("| CS_EN (разрешение дешифратора 74HC138, E2) | %s — перед обращением к внешним устройствам = 1 |", p.CSEnPin)
		w("| Адрес устройств | P2.5–P2.7 = A13–A15 → A0–A2 дешифратора; P2.0–P2.4 = A8–A12 |")
		w("| Свободные выходы дешифратора | %s |", p.UnusedCS)
	} else {
		w("| Свободная линия CS | %s (держим в 1) |", p.UnusedCS)
	}
	w("")
	w("## Карта внешней памяти (movx)")
	w("")
	w("| Устройство | CS | Окно |")
	w("| --- | --- | --- |")
	for _, d := range p.Devices {
		w("| %s | %s | %s–%s |", d.Name, d.CS, H(d.Base), H(d.Base+d.WinSize-1))
	}
	buf := p.Dev("Буфер (IDT7005)")
	w("")
	w("Буфер: %d байт, ячейки %s–%s, после %s указатель заворачивается на начало.", p.V, H(buf.Base), H(buf.Base+p.V-1), H(buf.Base+p.V-1))
	w("Отсчёт X2 внешнее устройство пишет правым портом IDT7005 (отдельный разъём: адрес, данные, строб записи) в ячейку обмена %s "+
		"вне кольца; тот же строб — прерывание МК: обработчик читает ячейку левым портом и кладёт отсчёт в кольцо процедурой записи.", H(buf.Base+buf.WinSize-1))
	w("")
	w("## Внутренняя память")
	w("")
	w("| Что | Где |")
	w("| --- | --- |")
	w("| «Голова» (2 байта, абсолютный адрес) | %s, %s |", H(p.Head), H(p.Head+1))
	w("| «Хвост» (2 байта) | %s, %s |", H(p.Tail), H(p.Tail+1))
	w("| Флаг пустоты | бит %d (%s) = %s |", p.Empty.Addr, H(p.Empty.Addr), p.Empty.Asm)
	w("| Флаг переполнения | бит %d (%s) = %s |", p.Ovf.Addr, H(p.Ovf.Addr), p.Ovf.Asm)
	w("| Стек | SP = 07h, до бит-области 20h — 24 байта |")
	w("")
	w("## Время (кварц %d МГц, машинный цикл %.3g мкс)", p.CrystalMHz, p.CycleUs)
	w("")
	w("| Интервал | Длительность | Таймер | Как отсчитывать |")
	w("| --- | --- | --- | --- |")
	w("| T1 — строб Y1 | %d мс | Timer%d | тик %d мс (загрузка %s) × %d, счётчик %d бит |", p.T1ms, p.Y1Timer, p.Y1.TickMs, H(p.Y1.Reload), p.Y1.Ticks, p.Y1.Counter)
	w("| T2 — строб Y2 | %d мкс | Timer%d | один отсчёт, загрузка %s |", p.T2us, p.Y2Timer, H(p.T2Reload))
	w("| T3 — индикация | %d мс | Timer%d | тик %d мс (загрузка %s) × %d, счётчик %d бит |", p.T3ms, p.T3Timer, p.T3.TickMs, H(p.T3.Reload), p.T3.Ticks, p.T3.Counter)
	w("")
	w("TMOD = 11h (оба таймера в режиме 1).")
	w("")
	w("## Пульт")
	w("")
	w("- Клавиатура: %s (%d столбца × %d строки).", p.Keyboard, p.Cols, p.Rows)
	w("- Индикатор: %s.", indName(p.Indicator))
	w("")
	w("## Ответы на п. 2.3 ТЗ")
	w("")
	w("- Минимальное время заполнения буфера при X1 = 0: V / %d = %d / %d = %.0f с.", p.X2Rate, p.V, p.X2Rate, p.FillTimeS)
	if p.G != 0 {
		w("- Функция: X1 ≠ 0 → Y2 = (G + M + X1 + X2) mod 256 = (%d + %d + X1 + X2) mod 256 = (%d + X1 + X2) mod 256; X1 = 0 → Y2 = 0.", p.G, p.M, p.G+p.M)
	} else {
		w("- Функция: X1 ≠ 0 → Y2 = (%d + X1 + X2) mod 256; X1 = 0 → Y2 = 0.", p.M)
	}
	return b.String()
}

// Asm — инклуд с EQU/BIT для всех трёх программ (синтаксис A51 / Keil).
func Asm(p *variant.Params) string {
	var b strings.Builder
	w := func(f string, a ...any) { fmt.Fprintf(&b, f+"\n", a...) }
	w("; Параметры варианта %s, M=%d (ТЗ %d). Сгенерировано mpsgen — не править руками.", p.Group, p.M, p.Year)
	w("")
	if p.G != 0 {
		w("VAR_M        EQU %d            ; номер варианта", p.M)
		w("VAR_G        EQU %d            ; номер группы", p.G)
		w("Y2_CONST     EQU %d            ; G+M: Y2 = (Y2_CONST + X1 + X2) mod 256", p.G+p.M)
	} else {
		w("VAR_M        EQU %d            ; номер варианта (для Y2 = M+X1+X2)", p.M)
	}
	w("")
	w("; --- адреса внешних устройств (movx) ---")
	w("ADR_IND      EQU %s        ; индикатор, CS %s", H(p.Dev("Индикатор").Base), p.Dev("Индикатор").CS)
	w("ADR_KB       EQU %s        ; клавиатура: запись столбцов / чтение строк, CS %s", H(p.Dev("Клавиатура").Base), p.Dev("Клавиатура").CS)
	w("ADR_Y2       EQU %s        ; регистр Y2, CS %s", H(p.Dev("Регистр Y2").Base), p.Dev("Регистр Y2").CS)
	buf := p.Dev("Буфер (IDT7005)")
	w("BUF_START    EQU %s        ; первая ячейка буфера, CS %s", H(buf.Base), buf.CS)
	w("BUF_END      EQU %s        ; первая ячейка ЗА буфером (точка заворота)", H(buf.Base+p.V))
	w("BUF_SIZE     EQU %d", p.V)
	w("BUF_X2IN     EQU %s        ; ячейка обмена: сюда внешнее устройство пишет X2 правым портом (строб X2 = прерывание INT1)", H(buf.Base+buf.WinSize-1))
	w("")
	w("; --- клавиатура %s и индикатор ---", p.Keyboard)
	w("KB_COLS      EQU %d             ; столбцы Col1…Col%d — биты D0…D%d при записи в ADR_KB (0 — столбец опрашивается)", p.Cols, p.Cols, p.Cols-1)
	w("KB_ROWS      EQU %d             ; строки Row1…Row%d — биты D0…D%d при чтении ADR_KB (0 — замкнута); старшие биты не определены — маскировать", p.Rows, p.Rows, p.Rows-1)
	off, kind := "00h", "общий катод: 1 зажигает сегмент"
	if p.Indicator == "anode" {
		off, kind = "0FFh", "общий анод: 0 зажигает сегмент"
	}
	w("IND_OFF      EQU %s           ; индикатор погашен (%s; D0…D7 = a…g, dp)", off, kind)
	w("")
	w("; --- внутреннее ОЗУ ---")
	w("HEAD_L       EQU %s          ; «голова», младший байт", H(p.Head))
	w("HEAD_H       EQU %s          ; «голова», старший байт", H(p.Head+1))
	w("TAIL_L       EQU %s          ; «хвост», младший байт", H(p.Tail))
	w("TAIL_H       EQU %s          ; «хвост», старший байт", H(p.Tail+1))
	w("F_EMPTY      BIT %s          ; флаг пустоты буфера (%s)", H(p.Empty.Addr), p.Empty.Asm)
	w("F_OVF        BIT %s          ; флаг переполнения буфера (%s)", H(p.Ovf.Addr), p.Ovf.Asm)
	w("")
	w("; --- выводы ---")
	w("PIN_Y1       BIT %s         ; строб Y1, активный 0", p.Y1Pin)
	w("PIN_Y2       BIT %s         ; строб Y2, активный 0", p.Y2Pin)
	if p.CSEnPin != "" {
		w("PIN_CSEN     BIT %s         ; CS_EN: разрешение дешифратора CS (1 — CS активны)", p.CSEnPin)
	}
	w("")
	w("; --- таймеры (кварц %d МГц) ---", p.CrystalMHz)
	w("T2_RELOAD_H  EQU %s          ; строб Y2 %d мкс, Timer%d", H(p.T2Reload>>8), p.T2us, p.Y2Timer)
	w("T2_RELOAD_L  EQU %s", H(p.T2Reload&0xFF))
	w("TICK_H       EQU %s          ; тик %d мс для Y1/T3, Timer%d", H(p.Y1.Reload>>8), p.Y1.TickMs, p.Y1Timer)
	w("TICK_L       EQU %s", H(p.Y1.Reload&0xFF))
	w("Y1_TICKS     EQU %d           ; T1 = %d мс", p.Y1.Ticks, p.T1ms)
	w("T3_TICKS     EQU %d          ; T3 = %d мс", p.T3.Ticks, p.T3ms)
	return b.String()
}

// VariantLine — строка для основной надписи схемы.
func VariantLine(groupFull string, m int) string {
	return fmt.Sprintf("Группа %s, Вариант %d, Э3", groupFull, m)
}
