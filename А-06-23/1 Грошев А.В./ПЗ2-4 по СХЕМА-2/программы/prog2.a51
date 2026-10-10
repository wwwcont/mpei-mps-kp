; Грошев А.В., А-06-23, 1, v1
$INCLUDE (vars.inc)             ; адреса, биты, константы варианта; в файл для робота mpscode вклеит его текст
DSEG AT 30h
; переменные не нужны: «голова», «хвост» и флаги уже заданы в vars.inc
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
        mov HEAD_L, #LOW(BUF_START)   ; буфер пуст: «голова» и «хвост» на первой ячейке
        mov HEAD_H, #HIGH(BUF_START)
        mov TAIL_L, #LOW(BUF_START)
        mov TAIL_H, #HIGH(BUF_START)
        setb F_EMPTY            ; данных нет
        clr F_OVF               ; и ничего не терялось
        mov A, #55h             ; пробный отсчёт для записи
        call BufWrite ; %proc%
        jmp $ ; %stop%

; BufWrite — запись отсчёта в кольцевой буфер IDT7005.
; Назначение: положить байт в ячейку «головы» и сдвинуть «голову»; если буфер
;   полон (голова догнала хвост, буфер не пуст), самый старый отсчёт теряется:
;   «хвост» сдвигается вперёд и ставится флаг переполнения.
; Вход: A — отсчёт; HEAD, TAIL, F_EMPTY — текущее состояние буфера.
; Выход: HEAD (и при переполнении TAIL) сдвинуты; F_EMPTY = 0; F_OVF = 1 при потере.
; Портит: ничего (A, B, DPTR, PSW сохраняются).
BufWrite:
        push ACC
        push B
        push PSW
        push DPL
        push DPH
        mov B, A                ; отсчёт держим в B, A нужен для сравнений
        jb F_EMPTY, BwPut       ; пустой буфер полным быть не может
        mov A, HEAD_L
        cjne A, TAIL_L, BwPut   ; голова не на хвосте — место есть
        mov A, HEAD_H
        cjne A, TAIL_H, BwPut
        mov DPL, TAIL_L         ; места нет: выбрасываем самый старый отсчёт
        mov DPH, TAIL_H
        call NextCell
        mov TAIL_L, DPL
        mov TAIL_H, DPH
        setb F_OVF              ; отмечаем, что данные терялись
BwPut:
        mov DPL, HEAD_L
        mov DPH, HEAD_H
        mov A, B
        movx @DPTR, A           ; кладём отсчёт в ячейку «головы»
        call NextCell
        mov HEAD_L, DPL         ; «голова» смотрит на следующую свободную ячейку
        mov HEAD_H, DPH
        clr F_EMPTY             ; в буфере есть хотя бы один отсчёт
        pop DPH
        pop DPL
        pop PSW
        pop B
        pop ACC
        ret

; NextCell — следующая ячейка кольцевого буфера.
; Вход: DPTR — адрес ячейки внутри буфера.
; Выход: DPTR — следующая ячейка; после последней (BUF_END - 1) — снова BUF_START.
; Портит: A, флаги.
NextCell:
        inc DPTR
        mov A, DPL
        cjne A, #LOW(BUF_END), NcOut  ; младший байт не совпал — до конца не дошли
        mov A, DPH
        cjne A, #HIGH(BUF_END), NcOut
        mov DPTR, #BUF_START    ; вышли за буфер — заворачиваем на начало
NcOut:
        ret
END
