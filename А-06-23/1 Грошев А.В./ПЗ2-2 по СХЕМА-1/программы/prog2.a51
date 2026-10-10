; Грошев А.В., А-06-23, 1, v1
$INCLUDE (vars.inc)             ; адреса, биты, константы варианта; в файл для робота mpscode вклеит его текст
DSEG AT 30h
; здесь объявление переменных (если нужны)
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
        ; TODO: CS_EN = 1; буфер пуст: HEAD = TAIL = BUF_START, F_EMPTY = 1, F_OVF = 0
        mov A, #55h             ; пример отсчёта
        call BufWrite ; %proc%
        jmp $ ; %stop%

; BufWrite — запись отсчёта в кольцевой буфер; если буфер полон — затирается самый старый.
; Вход: A — отсчёт. Выход: F_EMPTY = 0; F_OVF = 1, если отсчёт затёр самый старый.
; TODO: что портит
BufWrite:
        ; TODO: полон (HEAD = TAIL, F_EMPTY = 0) — сдвинуть TAIL и F_OVF = 1; movx по адресу HEAD, HEAD дальше (BUF_END → BUF_START)
        ret
END
