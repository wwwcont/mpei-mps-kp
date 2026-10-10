; Казарин А.А., А-12-23, 8, v1
$INCLUDE (vars.inc)             ; адреса, биты, константы варианта; в файл для робота mpscode вклеит его текст
DSEG AT 30h
TEMP:   DS 1
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
        setb PIN_CSEN              ; разрешить выбор внешней памяти
        mov HEAD_L, #LOW BUF_START ; начать запись с первой ячейки
        mov HEAD_H, #HIGH BUF_START
        mov TAIL_L, #LOW BUF_START ; начать чтение с той же ячейки
        mov TAIL_H, #HIGH BUF_START
        setb F_EMPTY               ; исходный буфер пуст
        clr F_OVF                  ; переполнения ещё не было
        setb EX1                   ; разрешить внешний запрос X2
        setb EA                    ; разрешить прерывания глобально
        call BufRead ; %proc%
        jmp $ ; %stop%

; BufRead — чтение самого старого отсчёта из кольцевого буфера.
; Вход: нет. Выход: A — отсчёт; F_EMPTY — буфер опустел; F_OVF сброшен (буфер больше не полон).
; Портит: DPTR, флаги PSW; при пустом буфере возвращает A = 0.
BufRead:
        clr EX1                    ; не менять указатели во время чтения
        jb F_EMPTY, BufEmpty
        mov DPL, TAIL_L            ; указать на старейший отсчёт
        mov DPH, TAIL_H
        movx A, @DPTR              ; получить значение из XRAM
        mov TEMP, A                ; сохранить результат при обновлении указателя
        mov A, TAIL_L              ; увеличить младшую часть адреса
        add A, #1
        mov TAIL_L, A
        mov A, TAIL_H              ; перенести переполнение в старшую часть
        addc A, #0
        mov TAIL_H, A
        cjne A, #HIGH BUF_END, BufCompareHead
        mov A, TAIL_L
        cjne A, #LOW BUF_END, BufCompareHead
        mov TAIL_L, #LOW BUF_START ; вернуться к началу кольца
        mov TAIL_H, #HIGH BUF_START
BufCompareHead:
        mov A, TAIL_L              ; последний элемент прочитан?
        cjne A, HEAD_L, BufStillHasData
        mov A, TAIL_H
        cjne A, HEAD_H, BufStillHasData
        setb F_EMPTY               ; голова догнана хвостом: данных нет
BufStillHasData:
        clr F_OVF                  ; после чтения место снова свободно
        mov A, TEMP                ; вернуть отсчёт вызывающей программе
        setb EX1                   ; разрешить новые X2
        ret
BufEmpty:
        mov A, #0                  ; пустой буфер возвращает ноль
        setb EX1                   ; восстановить разрешение INT1
        ret
END
