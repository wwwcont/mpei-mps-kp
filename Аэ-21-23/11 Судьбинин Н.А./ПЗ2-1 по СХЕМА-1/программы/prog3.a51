; Судьбинин Н.А., Аэ-21-23, 11, v1
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
org 1bh                         ; Timer1 — конец строба Y2
        ; TODO: остановить таймер, PIN_Y2 = 1
        reti
org 23h ; "заглушка" для UART
        nop
        reti
org 2bh
START:
; Инициализация МК **********************************************
        mov SP, #07h
        ; TODO: CS_EN = 1; PIN_Y2 = 1 (строб пассивен); TMOD для Timer1; разрешить ET1 и EA
        mov X1, #3              ; пример входных данных
        mov X2, #200
        call Y2Out ; %proc%
        jmp $ ; %stop%

; Y2Out — расчёт Y2 = (G+M+X1+X2) mod 256 (X1 = 0 → Y2 = 0), запись в регистр Y2 и строб T2.
; Вход: X1, X2. Выход: регистр Y2; PIN_Y2 = 0 на T2 мкс (конец — в прерывании Timer1).
; TODO: что портит
Y2Out:
        ; TODO: расчёт (Y2_CONST = G+M), movx в ADR_Y2, затем Timer1 = T2_RELOAD_H:L и PIN_Y2 = 0
        ret
END
