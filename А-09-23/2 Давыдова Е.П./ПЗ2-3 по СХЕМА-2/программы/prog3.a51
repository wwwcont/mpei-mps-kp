; Давыдова Е.П., А-09-23, 2, v1
$INCLUDE (vars.inc)             ; адреса, биты, константы варианта; в файл для робота mpscode вклеит его текст
DSEG AT 30h
X1:     DS 1                    ; цифра с клавиатуры (0 — Y2 пассивен)
X2:     DS 1                    ; отсчёт из буфера
CSEG AT 0h
        sjmp START
org 03h ; "заглушка" для INT0
        nop
        reti
org 0bh ; "заглушка" для Tmr0_ovf
        nop
        reti
org 13h ; "заглушка" для INT1
        nop
        reti
org 1bh                         ; Timer1 отсчитал T2 — конец строба Y2
        clr TR1                 ; таймер больше не нужен до следующего вывода
        setb PIN_Y2             ; строб в пассивную 1
        reti
org 23h ; "заглушка" для UART
        nop
        reti
org 2bh
START:
; Инициализация МК **********************************************
        mov SP, #07h
        setb PIN_CSEN           ; дешифратор включён — дойдёт CS регистра Y2
        setb PIN_Y2             ; строб Y2 пассивен
        mov TMOD, #10h          ; Timer1 — режим 1 (16 бит), Timer0 не используется
        setb ET1                ; прерывание по переполнению Timer1
        setb EA
        mov X1, #3              ; пример входных данных
        mov X2, #200
        call Y2Out ; %proc%
        jmp $ ; %stop%

; Y2Out — вычислить Y2, записать в регистр Y2 (74HC573) и выдать строб длительностью T2.
; Y2 = (G + M + X1 + X2) mod 256, если X1 не 0; при X1 = 0 — Y2 = 0.
; Вход: X1, X2 (ОЗУ).
; Выход: значение в регистре Y2; PIN_Y2 = 0, Timer1 запущен на T2 (строб снимает его прерывание).
; Портит: A (PSW и DPTR сохраняются в стеке).
Y2Out:
        push PSW
        push DPL
        push DPH
        mov A, X1
        jz Y2Put                ; X1 = 0 — выводим 0
        add A, X2               ; перенос отбрасывается: это и есть mod 256
        add A, #Y2_CONST        ; + (G + M)
Y2Put:
        mov DPTR, #ADR_Y2
        movx @DPTR, A           ; сначала значение в регистр
        clr TR1                 ; на случай незавершённого прошлого строба
        mov TH1, #T2_RELOAD_H   ; Timer1 отсчитает T2
        mov TL1, #T2_RELOAD_L
        clr TF1                 ; убрать старое переполнение
        clr PIN_Y2              ; начало строба
        setb TR1                ; пошёл отсчёт T2
        pop DPH
        pop DPL
        pop PSW
        ret
END
