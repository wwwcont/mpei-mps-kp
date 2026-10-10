; Давыдова Е.П., А-09-23, 2, v1
$INCLUDE (vars.inc)             ; адреса, биты, константы варианта; в файл для робота mpscode вклеит его текст
DSEG AT 30h
; своих переменных нет: указатели и флаги — по адресам из vars.inc
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
        setb PIN_CSEN           ; дешифратор включён — CS буфера будет работать
        mov HEAD_L, #LOW(BUF_START)  ; запись пойдёт в первую ячейку буфера
        mov HEAD_H, #HIGH(BUF_START)
        mov TAIL_L, #LOW(BUF_START)  ; читать тоже начнём с первой ячейки
        mov TAIL_H, #HIGH(BUF_START)
        setb F_EMPTY            ; буфер пуст
        clr F_OVF               ; переполнения не было
        call BufRead ; %proc%
        jmp $ ; %stop%

; BufRead — выдать самый старый отсчёт из кольцевого буфера (IDT7005, левый порт).
; Вход: «хвост» TAIL_H:TAIL_L, «голова» HEAD_H:HEAD_L, флаг F_EMPTY.
; Выход: A — отсчёт; «хвост» сдвинут на ячейку (после BUF_END-1 — на BUF_START);
;        F_OVF = 0; F_EMPTY = 1, если «хвост» догнал «голову».
;        Пустой буфер: A = 0, указатели и флаги не меняются.
; Портит: только A (PSW и DPTR сохраняются в стеке).
BufRead:
        jb F_EMPTY, BrNone      ; читать нечего
        push PSW
        push DPL
        push DPH
        mov DPL, TAIL_L
        mov DPH, TAIL_H
        movx A, @DPTR           ; самый старый отсчёт
        push ACC                ; сберечь его, пока двигаем «хвост»
        inc DPTR                ; следующая ячейка
        mov A, DPL
        cjne A, #LOW(BUF_END), BrKeep   ; ещё не вышли за буфер
        mov A, DPH
        cjne A, #HIGH(BUF_END), BrKeep
        mov DPTR, #BUF_START    ; вышли за последнюю ячейку — на начало кольца
BrKeep:
        mov TAIL_L, DPL
        mov TAIL_H, DPH
        clr F_OVF               ; одна ячейка освободилась — буфер не полон
        mov A, TAIL_L
        cjne A, HEAD_L, BrDone  ; «хвост» не совпал с «головой» — данные ещё есть
        mov A, TAIL_H
        cjne A, HEAD_H, BrDone
        setb F_EMPTY            ; прочитан последний отсчёт
BrDone:
        pop ACC                 ; вернуть отсчёт в A
        pop DPH
        pop DPL
        pop PSW
        ret
BrNone:
        clr A                   ; пусто: отдать 0, ничего не трогать
        ret
END
