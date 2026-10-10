; Судьбинин Н.А., Аэ-21-23, 11, v1
$INCLUDE (vars.inc)             ; адреса, биты, константы варианта; в файл для робота mpscode вклеит его текст
DSEG AT 30h
; отдельных переменных нет: «голова», «хвост» и флаги — по адресам из vars.inc
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
        mov SP, #07h            ; стек сразу над банком 0
        setb PIN_CSEN           ; разрешаем дешифратор адреса — без него буфер не выбирается
        mov HEAD_L, #LOW(BUF_START)   ; «голова» — на первую ячейку буфера
        mov HEAD_H, #HIGH(BUF_START)
        mov TAIL_L, #LOW(BUF_START)   ; «хвост» — туда же: буфер пуст
        mov TAIL_H, #HIGH(BUF_START)
        setb F_EMPTY            ; отсчётов нет
        clr F_OVF               ; и ничего не терялось
        mov A, #55h             ; пример отсчёта
        call BufWrite ; %proc%
        jmp $ ; %stop%

; BufWrite — запись отсчёта в кольцевой буфер IDT7005.
; Буфер полон, когда «голова» догнала «хвост» и он не пуст (политика V: занято все BUF_SIZE ячеек);
; тогда самый старый отсчёт выбрасывается — «хвост» сдвигается вперёд, ставится F_OVF.
; Вход: A — отсчёт. Выход: отсчёт в ячейке бывшей «головы», «голова» на следующей ячейке,
; F_EMPTY = 0; F_OVF = 1, если затёрт самый старый отсчёт (иначе F_OVF не меняется).
; Портит A, B, DPTR, PSW.
BufWrite:
        mov B, A                ; отсчёт придержим в B, A нужен для сравнений
        jb F_EMPTY, BwPut       ; пустой буфер точно не полон
        mov A, HEAD_L
        cjne A, TAIL_L, BwPut   ; «голова» не на «хвосте» — место есть
        mov A, HEAD_H
        cjne A, TAIL_H, BwPut
        mov DPL, TAIL_L         ; буфер полон: выбрасываем самый старый отсчёт
        mov DPH, TAIL_H
        call BufNext            ; «хвост» на следующую ячейку
        mov TAIL_L, DPL
        mov TAIL_H, DPH
        setb F_OVF              ; отмечаем, что отсчёт потерян
BwPut:
        mov DPL, HEAD_L         ; адрес ячейки, куда пишем
        mov DPH, HEAD_H
        mov A, B
        movx @DPTR, A           ; кладём отсчёт в буфер
        call BufNext            ; «голова» на следующую ячейку
        mov HEAD_L, DPL
        mov HEAD_H, DPH
        clr F_EMPTY             ; теперь в буфере точно есть отсчёт
        ret

; BufNext — следующая ячейка кольцевого буфера.
; Вход: DPTR — адрес ячейки буфера. Выход: DPTR — адрес следующей; за последней ячейкой (BUF_END − 1)
; идёт первая (BUF_START). Портит A, PSW.
BufNext:
        inc DPTR
        mov A, DPL
        cjne A, #LOW(BUF_END), BnRet  ; младший байт не на границе — заворота нет
        mov A, DPH
        cjne A, #HIGH(BUF_END), BnRet ; старший тоже должен совпасть
        mov DPTR, #BUF_START    ; вышли за буфер — заворачиваем на начало
BnRet:
        ret
END
