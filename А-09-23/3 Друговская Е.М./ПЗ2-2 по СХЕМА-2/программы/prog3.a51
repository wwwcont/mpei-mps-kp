; Друговская Е.М., А-09-23, 3, v1
$INCLUDE (vars.inc)             ; адреса, биты, константы варианта; в файл для робота mpscode вклеит его текст
T0_LOAD EQU (TICK_H SHL 8) + TICK_L + 7   ; загрузка тика 5 мс + 7 циклов, пока таймер стоит в обработчике
DSEG AT 30h
T3Cnt:  DS 2                    ; сколько тиков по 5 мс осталось до гашения (младший, старший байт)
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
        setb PIN_CSEN           ; дешифратор включён — регистр индикатора доступен
        mov DPTR, #ADR_IND
        mov A, #IND_OFF
        movx @DPTR, A           ; после сброса индикатор тёмный
        mov TMOD, #11h          ; оба таймера — 16-разрядные (режим 1)
        setb ET0                ; прерывание Timer0 — отсчёт T3
        setb EA
        mov A, #5               ; пример: показать «5»
        call Show ; %proc%
        jmp $ ; %stop%

; Show — вывод символа на индикатор (общий анод) на время T3; гасит его обработчик Timer0.
; Вход: A — код по табл. 2 ТЗ: 0–9 — цифра, 10 — «E», больше — погасить.
; Выход: символ в регистре индикатора; для 0–10 заново запущен отсчёт T3.
; Портит: A, DPTR, PSW.
Show:
        cjne A, #11, $+3        ; C = 1, если код меньше 11
        jnc ShBlank             ; 11 и больше — показывать нечего
        mov DPTR, #SegTab
        movc A, @A+DPTR         ; код символа -> сегменты
        mov DPTR, #ADR_IND
        movx @DPTR, A           ; символ загорается
        clr TR0                 ; старый отсчёт (если шёл) отменяем
        clr ET0                 ; счётчик меняем без помех от прерывания
        mov T3Cnt, #LOW(T3_TICKS)
        mov T3Cnt+1, #HIGH(T3_TICKS)
        mov TH0, #TICK_H        ; первый тик 5 мс
        mov TL0, #TICK_L
        clr TF0
        setb ET0
        setb TR0                ; пошёл отсчёт T3
        ret
ShBlank:
        mov A, #IND_OFF
        mov DPTR, #ADR_IND
        movx @DPTR, A           ; гасим индикатор сразу
        clr TR0                 ; отсчёт T3 больше не нужен
        ret

; T0Isr — тик Timer0 (5 мс): перезагрузка таймера с поправкой, счёт тиков T3,
; по окончании T3 — индикатор гаснет и таймер останавливается.
; Вход: T3Cnt. Выход: T3Cnt уменьшен. Портит: ничего (A, PSW, DPTR сохраняются).
T0Isr:
        push ACC
        push PSW
        clr TR0                 ; на время перезагрузки таймер стоит
        mov A, TL0              ; TL0/TH0 уже набежали после переполнения — прибавляем к ним
        add A, #LOW(T0_LOAD)
        mov TL0, A
        mov A, TH0
        addc A, #HIGH(T0_LOAD)
        mov TH0, A
        setb TR0
        mov A, T3Cnt            ; T3Cnt := T3Cnt - 1 (16 бит)
        jnz T0Low
        dec T3Cnt+1
T0Low:
        dec T3Cnt
        mov A, T3Cnt
        orl A, T3Cnt+1
        jnz T0Exit              ; T3 ещё не кончилось
        clr TR0                 ; T3 вышло — таймер больше не нужен
        push DPL
        push DPH
        mov DPTR, #ADR_IND
        mov A, #IND_OFF
        movx @DPTR, A           ; гасим индикатор
        pop DPH
        pop DPL
T0Exit:
        pop PSW
        pop ACC
        reti

; Сегменты для общего анода (0 — сегмент горит; D0…D7 = a…g, dp): цифры 0–9 и «E».
SegTab: DB 0C0h, 0F9h, 0A4h, 0B0h, 099h
        DB 092h, 082h, 0F8h, 080h, 090h
        DB 086h
END
