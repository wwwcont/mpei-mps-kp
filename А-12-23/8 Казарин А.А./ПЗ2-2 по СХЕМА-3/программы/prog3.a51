; Казарин А.А., А-12-23, 8, v1
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
        clr TR1                    ; остановить отсчёт длительности
        setb PIN_Y2                ; вернуть строб в пассивное состояние
        reti
org 23h ; "заглушка" для UART
        nop
        reti
org 2bh
START:
; Инициализация МК **********************************************
        mov SP, #07h
        setb PIN_CSEN              ; включить дешифратор адреса
        setb PIN_Y2                ; строб неактивен до первого вызова
        mov TMOD, #11h             ; таймеры в режиме 1
        clr TR1                    ; Timer1 остановлен
        setb ET1                   ; разрешить прерывание Timer1
        setb EA                    ; разрешить прерывания глобально
        mov X1, #0                 ; пример: нулевой ввод держит Y2 пассивным
        mov X2, #0
        call Y2Out ; %proc%
        jmp $ ; %stop%

; Y2Out — расчёт Y2 = (G+M+X1+X2) mod 256 (X1 = 0 → Y2 = 0), запись в регистр Y2 и строб T2.
; Вход: X1, X2. Выход: регистр Y2; PIN_Y2 = 0 на T2 мкс (конец — в прерывании Timer1).
; Портит: A, DPTR, флаги PSW, Timer1.
Y2Out:
        mov A, X1                  ; нулевой X1 требует нулевого выхода
        jz Y2Zero
        add A, X2                  ; сложение выполняется по модулю 256
        add A, #Y2_CONST           ; прибавить G + M из vars.inc
        sjmp Y2Write
Y2Zero:
        clr A                      ; пассивное значение при X1 = 0
Y2Write:
        mov DPTR, #ADR_Y2          ; выбрать внешний регистр Y2
        movx @DPTR, A              ; записать значение до начала стробы
        mov TH1, #T2_RELOAD_H      ; загрузить длительность T2
        mov TL1, #T2_RELOAD_L
        clr TF1                    ; сбросить старое переполнение
        clr PIN_Y2                 ; активная часть стробы начинается
        setb TR1                   ; Timer1 завершит строб по прерыванию
        ret
END
