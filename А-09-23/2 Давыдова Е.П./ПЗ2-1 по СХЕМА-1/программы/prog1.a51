; Давыдова Е.П., А-09-23, 2, v1
$INCLUDE (vars.inc)             ; адреса, биты, константы варианта; в файл для робота mpscode вклеит его текст
DSEG AT 30h
; переменных в ОЗУ нет: опрос работает на регистрах R2…R6 и B
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
        setb PIN_CSEN           ; разрешить дешифратор, чтобы дошёл CS клавиатуры
        mov DPTR, #ADR_KB
        clr A
        movx @DPTR, A           ; все столбцы в 0: любое нажатие опустит строку и ~INT0
        setb IT0                ; INT0 — по спаду
        clr IE0                 ; забыть спад, возможный при установке столбцов
        setb EX0                ; разрешить прерывание от клавиатуры
        setb EA                 ; общее разрешение прерываний
        jmp $ ; %stop%

; KbScan — опрос матрицы 4x3 по столбцам (вызывается из обработчика INT0).
; Проходит Col1…Col4, в каждом читает строки и считает замкнутые контакты.
; Вход: нет.
; Выход: A — код клавиши по табл. 3 ТЗ, если нажата ровно одна; 0FFh — ни одной или несколько.
; Портит: только A (PSW, B, DPTR, R2…R6 сохраняются в стеке).
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
        mov R2, #0FFh           ; код найденной клавиши (пока нет)
        mov R3, #0              ; сколько контактов замкнуто
        mov R4, #0              ; номер столбца 0…3
        mov R5, #0FEh           ; маска: 0 только в бите текущего столбца
KbCol:
        mov DPTR, #ADR_KB
        mov A, R5
        movx @DPTR, A           ; опустить один столбец, остальные держать в 1
        movx A, @DPTR           ; прочитать строки через 74HC244
        cpl A                   ; теперь 1 — строка замкнута
        anl A, #(1 SHL KB_ROWS) - 1 ; старшие линии шины к строкам не подключены
        mov B, A                ; биты замкнутых строк этого столбца
        mov A, R4
        mov R6, A               ; индекс в KeyTab: строка * KB_COLS + столбец, начинаем с Row1
KbRow:
        mov A, B
        jz KbNext               ; в столбце больше нет замкнутых строк
        clr C
        rrc A                   ; младшая строка уходит в C
        mov B, A
        jnc KbSkip              ; эта строка не замкнута
        inc R3                  ; ещё один замкнутый контакт
        mov A, R6
        mov DPTR, #KeyTab
        movc A, @A+DPTR         ; код клавиши по раскладке рис. 4
        mov R2, A
KbSkip:
        mov A, R6
        add A, #KB_COLS         ; та же колонка, следующая строка
        mov R6, A
        sjmp KbRow
KbNext:
        mov A, R5
        rl A                    ; ноль переезжает на следующий столбец
        mov R5, A
        inc R4
        cjne R4, #KB_COLS, KbCol
        mov DPTR, #ADR_KB
        clr A
        movx @DPTR, A           ; снова все столбцы в 0 — ждём следующее нажатие
        clr IE0                 ; спады ~INT0 во время перебора — не новое нажатие
        mov A, R2
        cjne R3, #1, KbBad      ; ровно одна клавиша — код, иначе ошибка
        sjmp KbOut
KbBad:
        mov A, #0FFh            ; ничего не нажато (помеха) или нажато несколько
KbOut:
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

; KeyTab — коды клавиш (табл. 3 ТЗ) в порядке строка * 4 + столбец, раскладка 4x3 рис. 4:
; Row1: 1 2 3 Т, Row2: 4 5 6 С, Row3: 7 8 9 0; Т = 10, С = 11.
KeyTab: DB 1, 2, 3, 10
        DB 4, 5, 6, 11
        DB 7, 8, 9, 0
END
