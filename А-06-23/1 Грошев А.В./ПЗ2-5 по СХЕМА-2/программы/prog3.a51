; Грошев А.В., А-06-23, 1, v1
$INCLUDE (vars.inc)             ; адреса, биты, константы варианта; в файл для робота mpscode вклеит его текст
DSEG AT 30h
Y1Left: DS 1                    ; сколько тиков по 2 мс осталось до конца строба Y1
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
        setb PIN_CSEN           ; дешифратор адреса разрешён
        setb PIN_Y1             ; строб Y1 в пассивном состоянии
        mov Y1Left, #0
        anl TMOD, #0F0h         ; Timer0 — 16-битный счётчик (режим 1), Timer1 не трогаем
        orl TMOD, #01h
        clr TR0                 ; таймер пока стоит
        clr TF0
        setb ET0                ; прерывание по переполнению Timer0
        setb EA
        call Y1Start ; %proc%
        jmp $ ; %stop%

; Y1Start — запуск строба Y1 длительностью T1.
; Назначение: опустить PIN_Y1 в 0 и запустить Timer0 на Y1_TICKS тиков по 2 мс;
;   строб поднимает обработчик Timer0, процедура сразу возвращается.
; Вход: нет. Выход: PIN_Y1 = 0, Timer0 запущен, Y1Left = Y1_TICKS.
; Портит: ничего (меняет только TH0, TL0, TF0, TR0).
Y1Start:
        clr TR0                 ; повторный вызов начинает отсчёт заново
        mov TH0, #TICK_H        ; первый тик — ровно 2 мс
        mov TL0, #TICK_L
        clr TF0
        mov Y1Left, #Y1_TICKS
        clr PIN_Y1              ; начало строба
        setb TR0
        ret

; T0Isr — обработчик Timer0: прошёл очередной тик 2 мс.
; Назначение: перезарядить Timer0 с учётом тактов, натикавших после переполнения,
;   уменьшить счётчик и по его обнулению закончить строб Y1.
; Вход: Y1Left. Выход: Y1Left - 1; при нуле PIN_Y1 = 1 и Timer0 остановлен.
; Портит: ничего (A и PSW в стеке).
T0Isr:
        push ACC
        push PSW
        clr TR0                 ; на время перезарядки таймер стоит (7 тактов)
        mov A, TL0
        add A, #LOW((TICK_H SHL 8) + TICK_L + 7)   ; добавляем к уже натикавшему и
        mov TL0, A                                 ; учитываем 7 тактов остановки
        mov A, TH0
        addc A, #HIGH((TICK_H SHL 8) + TICK_L + 7)
        mov TH0, A
        setb TR0
        djnz Y1Left, T0Out      ; строб ещё идёт
        clr TR0                 ; T1 истекло: таймер больше не нужен
        setb PIN_Y1             ; конец строба
T0Out:
        pop PSW
        pop ACC
        reti
END
