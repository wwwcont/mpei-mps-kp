; Друговская Е.М., А-09-23, 3, v1
$INCLUDE (vars.inc)             ; адреса, биты, константы варианта; в файл для робота mpscode вклеит его текст
DSEG AT 30h
; здесь объявление переменных (счётчик тиков и т. п.)
CSEG AT 0h
        sjmp START
org 03h ; "заглушка" для INT0
        nop
        reti
org 0bh                         ; Timer0 — тик для отсчёта T3
        ljmp T0Isr
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
        ; TODO: CS_EN = 1; индикатор погасить (IND_OFF в ADR_IND); TMOD для Timer0; разрешить ET0 и EA
        mov A, #5               ; пример: показать «5»
        call Show ; %proc%
        jmp $ ; %stop%

; Show — вывод символа на индикатор на T3 (гашение — в обработчике Timer0).
; Вход: A — код по табл. 2 ТЗ (0–9 — цифры, 10 — «E», больше — погасить).
; TODO: что портит
Show:
        ; TODO: код → сегменты (a…g, dp = D0…D7; общий анод — инверсия), movx в ADR_IND, (пере)запуск отсчёта T3_TICKS
        ret

; T0Isr — тик Timer0: перезагрузка, счёт тиков, по окончании T3 — IND_OFF в индикатор.
T0Isr:
        ; TODO
        reti
END
