; Лапташкин В.И., А-08-23, 12, v1
$INCLUDE (vars.inc)             ; адреса, биты, константы варианта; в файл для робота mpscode вклеит его текст
DSEG AT 30h
; своих переменных нет: указатели и флаги буфера заданы в vars.inc
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
        mov SP, #07h            ; стек сразу за банком 0
        setb PIN_CSEN           ; разрешить дешифратор CS, иначе буфер не выбирается
        mov HEAD_L, #LOW(BUF_START)  ; «голова» — на первую ячейку буфера
        mov HEAD_H, #HIGH(BUF_START)
        mov TAIL_L, #LOW(BUF_START)  ; «хвост» — туда же: буфер пуст
        mov TAIL_H, #HIGH(BUF_START)
        setb F_EMPTY            ; отсчётов нет
        clr F_OVF               ; переполнения не было
        call BufRead ; %proc%
        jmp $ ; %stop%

; BufRead — выдать самый старый отсчёт из кольцевого буфера в IDT7005.
; Если буфер пуст, ничего не меняется и возвращается A = 0.
; Иначе читается ячейка «хвоста», «хвост» сдвигается (за последней ячейкой —
; снова первая), флаг переполнения снимается, а если «хвост» догнал «голову» —
; ставится флаг пустоты.
; Вход: нет. Выход: A — отсчёт (0 при пустом буфере); F_EMPTY, F_OVF.
; Портит: A и флаги; DPTR и B сохраняются в стеке.
BufRead:
        jb F_EMPTY, BrEmpty     ; читать нечего
        push DPL
        push DPH
        push B
        mov DPL, TAIL_L         ; адрес самого старого отсчёта
        mov DPH, TAIL_H
        movx A, @DPTR           ; взять отсчёт из буфера
        mov B, A                ; отложить, A нужен для сравнений
        inc DPTR                ; следующая ячейка
        mov A, DPL              ; вышли за конец буфера? сверяем оба байта
        cjne A, #LOW(BUF_END), BrSave
        mov A, DPH
        cjne A, #HIGH(BUF_END), BrSave
        mov DPTR, #BUF_START    ; заворот на начало кольца
BrSave:
        mov TAIL_L, DPL         ; запомнить новый «хвост»
        mov TAIL_H, DPH
        clr F_OVF               ; одно место освободилось — буфер не полон
        mov A, TAIL_L           ; «хвост» догнал «голову» — всё прочитано
        cjne A, HEAD_L, BrDone
        mov A, TAIL_H
        cjne A, HEAD_H, BrDone
        setb F_EMPTY
BrDone:
        mov A, B                ; вернуть отсчёт
        pop B
        pop DPH
        pop DPL
        ret
BrEmpty:
        clr A                   ; пустой буфер: условный ноль, указатели не трогаем
        ret
END
