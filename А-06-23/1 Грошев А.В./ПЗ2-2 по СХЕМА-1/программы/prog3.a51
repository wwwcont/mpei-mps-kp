; Грошев А.В., А-06-23, 1, v1
$INCLUDE (vars.inc)             ; адреса, биты, константы варианта; в файл для робота mpscode вклеит его текст
DSEG AT 30h
; здесь объявление переменных (счётчик тиков и т. п.)
CSEG AT 0h
        sjmp START
org 03h ; "заглушка" для INT0
        nop
        reti
org 0bh                         ; Timer0 — тик для отсчёта T1
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
        ; TODO: CS_EN = 1; PIN_Y1 = 1 (строб пассивен); TMOD для Timer0; разрешить ET0 и EA
        call Y1Start ; %proc%
        jmp $ ; %stop%

; Y1Start — начало строба Y1 = 0 длительностью T1 на PIN_Y1; конец — в обработчике Timer0.
; Вход, выход: нет.
; TODO: что портит
Y1Start:
        ; TODO: PIN_Y1 = 0, запустить отсчёт Y1_TICKS тиков по TICK_H:TICK_L (в цикле не ждать!)
        ret

; T0Isr — тик Timer0: перезагрузка, счёт тиков, по окончании T1 — PIN_Y1 = 1.
T0Isr:
        ; TODO
        reti
END
