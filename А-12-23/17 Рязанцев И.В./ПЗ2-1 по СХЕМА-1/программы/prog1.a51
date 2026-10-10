; Рязанцев И.В., А-12-23, 17, v1
$INCLUDE (vars.inc)             ; адреса, биты, константы варианта; в файл для робота mpscode вклеит его текст
DSEG AT 30h
; переменные не нужны: всё держим в регистрах банка 0
CSEG AT 0h
        sjmp START
org 03h                         ; INT0 — нажатие клавиши: процедура вызывается отсюда
        call KbScan ; %proc%
        reti
org 0bh ; "заглушка" для Tmr0_ovf
        nop
        reti
org 13h ; "заглушка" для INT1
        nop
        reti
org 1bh ; "заглушка" для Tmr1_ovf
        nop
        reti
org 23h ; "заглушка" для UART
        nop
        reti
org 2bh
START:
; Инициализация МК **********************************************
        mov SP, #07h
        setb PIN_CSEN           ; разрешаем дешифратор, иначе клавиатура не выбирается
        mov DPTR, #ADR_KB
        clr A
        movx @DPTR, A           ; опускаем все столбцы: любое нажатие сразу опустит INT0
        setb IT0                ; INT0 срабатывает по спаду
        clr IE0                 ; забываем возможный ложный запрос до инициализации
        setb EX0                ; разрешаем прерывание от клавиатуры
        setb EA                 ; и прерывания вообще
        jmp $ ; %stop%

; KbScan — опрос матрицы клавиатуры 3x4 по столбцам (вызывается из обработчика INT0).
; Столбцы опускаются по одному, в каждом считаются замкнутые строки; код берётся из таблицы KeyMap.
; Вход: нет. Выход: A — код единственной нажатой клавиши (0…9, Т = 10, С = 11) или 0FFh,
; если не нажато ничего или нажато больше одной клавиши. Портит: только A (остальное сохраняется в стеке).
KbScan:
        push PSW
        push B
        push DPL
        push DPH
        push AR2
        push AR3
        push AR4
        push AR5
        push AR6
        push AR7
        mov DPTR, #ADR_KB
        mov R2, #0              ; сколько замкнутых контактов нашли
        mov R3, #0FFh           ; код последней найденной клавиши
        mov R4, #0              ; номер текущего столбца (0…KB_COLS-1)
        mov R5, #0FEh           ; маска: опущен только первый столбец
KsCol:
        mov A, R5
        movx @DPTR, A           ; опускаем текущий столбец, остальные подняты
        movx A, @DPTR           ; читаем строки
        anl A, #(1 SHL KB_ROWS) - 1     ; старшие биты шины не подключены
        xrl A, #(1 SHL KB_ROWS) - 1     ; теперь 1 — строка замкнута на этот столбец
        mov R7, A               ; замкнутые строки этого столбца
        mov R6, #0              ; номер строки
KsRow:
        mov A, R7
        jz KsNextCol            ; в этом столбце замкнутых строк больше нет
        clr C
        rrc A                   ; младшая строка уходит в перенос
        mov R7, A
        jnc KsRowSkip
        inc R2                  ; нашли ещё одну нажатую клавишу
        mov A, R6               ; индекс в таблице = строка * KB_COLS + столбец
        mov B, #KB_COLS
        mul AB
        add A, R4
        push DPL                ; адрес клавиатуры ещё понадобится
        push DPH
        mov DPTR, #KeyMap
        movc A, @A+DPTR         ; код клавиши по рис. 4 и табл. 3 ТЗ
        mov R3, A
        pop DPH
        pop DPL
KsRowSkip:
        inc R6
        sjmp KsRow
KsNextCol:
        mov A, R5
        rl A                    ; опускать будем следующий столбец
        mov R5, A
        inc R4
        cjne R4, #KB_COLS, KsCol
        clr A
        movx @DPTR, A           ; опрос закончен: все столбцы снова вниз, чтобы ловить следующее нажатие
        mov A, R3
        cjne R2, #1, KsBad      ; ровно одна клавиша — отдаём её код
        sjmp KsDone
KsBad:
        mov A, #0FFh            ; ничего или несколько клавиш сразу — ошибка
KsDone:
        clr IE0                 ; пока перебирали столбцы, INT0 дёргался — этот запрос лишний
        pop AR7
        pop AR6
        pop AR5
        pop AR4
        pop AR3
        pop AR2
        pop DPH
        pop DPL
        pop B
        pop PSW
        ret

; KeyMap — коды клавиш по строкам сверху вниз, в строке — по столбцам слева направо (рис. 4 ТЗ, 3x4).
KeyMap: DB 1, 2, 3
        DB 4, 5, 6
        DB 7, 8, 9
        DB 0, 10, 11
END
