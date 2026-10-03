; Рязанцев И.В., А-12-23, 17, v1
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
org 1bh                         ; Timer1 — отсчитали T2, строб Y2 заканчивается
        clr TR1                 ; таймер больше не нужен
        setb PIN_Y2             ; строб в пассивную единицу
        reti
org 23h ; "заглушка" для UART
        nop
        reti
org 2bh
START:
; Инициализация МК **********************************************
        mov SP, #07h
        setb PIN_CSEN           ; дешифратор включён — регистр Y2 доступен по movx
        setb PIN_Y2             ; строб Y2 пока не активен
        anl TMOD, #0Fh          ; Timer1: режим 1 (16 бит), Timer0 не трогаем
        orl TMOD, #10h
        setb PT1                ; конец строба важнее остального — высокий приоритет
        setb ET1                ; прерывание от Timer1
        setb EA
        mov X1, #3              ; пример входных данных
        mov X2, #200
        call Y2Out ; %proc%
        jmp $ ; %stop%

; Y2Out — вычисление выхода Y2 и его выдача внешнему устройству.
; Y2 = (G + M + X1 + X2) mod 256, а при X1 = 0 — просто 0. Значение пишется в регистр Y2,
; после этого на P1.4 выдаётся строб 0 длиной T2; конец строба делает обработчик Timer1.
; Вход: X1, X2. Выход: регистр Y2, строб на PIN_Y2 запущен. Портит: ничего (A, DPTR, PSW сохраняются).
Y2Out:
        push PSW
        push ACC
        push DPL
        push DPH
        mov A, X1
        jz YoWrite              ; X1 = 0 — выдаём ноль
        add A, #Y2_CONST        ; G + M + X1; перенос отбрасывается — это и есть mod 256
        add A, X2
YoWrite:
        mov DPTR, #ADR_Y2
        movx @DPTR, A           ; сначала данные в регистр
        clr TR1                 ; на случай, если прошлый строб ещё идёт — перезапускаем отсчёт
        mov TH1, #T2_RELOAD_H   ; Timer1 отсчитает T2
        mov TL1, #T2_RELOAD_L
        clr TF1                 ; старый запрос не нужен
        clr PIN_Y2              ; строб пошёл
        setb TR1
        pop DPH
        pop DPL
        pop ACC
        pop PSW
        ret
END
