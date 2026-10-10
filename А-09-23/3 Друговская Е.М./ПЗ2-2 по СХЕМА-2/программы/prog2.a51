; Друговская Е.М., А-09-23, 3, v1
$INCLUDE (vars.inc)             ; адреса, биты, константы варианта; в файл для робота mpscode вклеит его текст
DSEG AT 30h
; переменные не нужны: указатели и флаги буфера заданы в vars.inc
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
        setb PIN_CSEN           ; дешифратор включён — буфер доступен по movx
        mov HEAD_L, #LOW(BUF_START)   ; писать будем с первой ячейки кольца
        mov HEAD_H, #HIGH(BUF_START)
        mov TAIL_L, #LOW(BUF_START)   ; читать — тоже с первой: голова = хвост
        mov TAIL_H, #HIGH(BUF_START)
        setb F_EMPTY            ; отсчётов в буфере нет
        clr F_OVF               ; ничего не потеряно
        mov A, #55h             ; пример отсчёта
        call BufWrite ; %proc%
        jmp $ ; %stop%

; BufWrite — запись отсчёта в кольцевой буфер в ячейку «головы».
; Если буфер уже полон (голова догнала хвост, буфер не пуст), самый старый
; отсчёт теряется: хвост уходит на следующую ячейку, F_OVF = 1.
; Вход: A — отсчёт.
; Выход: отсчёт в буфере, голова сдвинута, F_EMPTY = 0; F_OVF = 1 при затирании.
; Портит: DPTR, PSW (A сохраняется).
BufWrite:
        push ACC                ; отсчёт понадобится после проверок
        jb F_EMPTY, BwPut       ; пустой буфер полным быть не может
        mov A, HEAD_L
        cjne A, TAIL_L, BwPut   ; голова не на хвосте — место есть
        mov A, HEAD_H
        cjne A, TAIL_H, BwPut
        mov DPL, TAIL_L         ; буфер полон: выбрасываем самый старый отсчёт
        mov DPH, TAIL_H
        call PtrStep
        mov TAIL_L, DPL
        mov TAIL_H, DPH
        setb F_OVF              ; отмечаем, что отсчёт потерян
BwPut:
        mov DPL, HEAD_L         ; адрес ячейки «головы»
        mov DPH, HEAD_H
        pop ACC
        movx @DPTR, A           ; кладём отсчёт левым портом IDT7005
        call PtrStep
        mov HEAD_L, DPL         ; голова — на следующую свободную ячейку
        mov HEAD_H, DPH
        clr F_EMPTY             ; теперь в буфере точно что-то есть
        ret

; PtrStep — переход указателя буфера на следующую ячейку кольца.
; Вход: DPTR — адрес ячейки буфера.
; Выход: DPTR — следующая ячейка; после последней (BUF_END - 1) — BUF_START.
; Портит: PSW (A сохраняется).
PtrStep:
        push ACC
        inc DPTR
        mov A, DPL
        cjne A, #LOW(BUF_END), PsDone   ; не дошли до конца кольца
        mov A, DPH
        cjne A, #HIGH(BUF_END), PsDone
        mov DPTR, #BUF_START    ; вышли за последнюю ячейку — заворачиваем на начало
PsDone:
        pop ACC
        ret
END
