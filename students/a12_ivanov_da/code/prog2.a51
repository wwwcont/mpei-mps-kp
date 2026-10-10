; Иванов Д.А., А-12-23, 5, v1
$INCLUDE (vars.inc)             ; параметры варианта формирует генератор
DSEG AT 30h
; дополнительные ячейки ОЗУ не нужны
CSEG AT 0h
        sjmp START
org 03h                         ; заглушка INT0
        nop
        reti
org 0bh                         ; заглушка Timer0
        nop
        reti
org 13h                         ; заглушка INT1
        nop
        reti
org 1bh                         ; заглушка Timer1
        nop
        reti
org 23h                         ; заглушка UART
        nop
        reti
org 2bh
START:
; Инициализация МК **********************************************
        mov SP, #07h            ; стек не пересекается с указателями буфера
        setb PIN_CSEN           ; разрешить дешифратор внешней памяти
        mov DPTR, #BUF_START    ; пустой буфер начинается с первой ячейки кольца
        mov HEAD_L, DPL
        mov HEAD_H, DPH
        mov TAIL_L, DPL
        mov TAIL_H, DPH
        setb F_EMPTY            ; после сброса данных нет
        clr F_OVF
        mov A, #55h             ; тестовый отсчёт для вызова процедуры
        call BufWrite ; %proc%
        jmp $ ; %stop%

; BufWrite — запись байта в кольцевой буфер с политикой полной ёмкости V.
; Вход: A — новый отсчёт. Выход: F_EMPTY = 0; F_OVF = 1 только при вытеснении старого байта.
; Портит: A, B, DPTR, R6, R7, PSW.
BufWrite:
        mov B, A                ; сохранить отсчёт на время проверки указателей
        clr F_OVF               ; обычная запись снимает прежний признак переполнения
        jb F_EMPTY, BW_STORE    ; в пустом кольце совпадение указателей не означает полноту
        mov A, HEAD_L
        cjne A, TAIL_L, BW_STORE
        mov A, HEAD_H
        cjne A, TAIL_H, BW_STORE
        mov DPL, TAIL_L         ; буфер полон: удалить самый старый отсчёт
        mov DPH, TAIL_H
        inc DPTR
        mov R6, DPL             ; сохранить полученный адрес для сравнения с концом
        mov R7, DPH
        mov DPTR, #BUF_END
        mov A, R6
        cjne A, DPL, BW_TAIL_KEEP
        mov A, R7
        cjne A, DPH, BW_TAIL_KEEP
        mov DPTR, #BUF_START    ; после BUF_END кольцо продолжается с начала
        sjmp BW_TAIL_SAVE
BW_TAIL_KEEP:
        mov DPL, R6
        mov DPH, R7
BW_TAIL_SAVE:
        mov TAIL_L, DPL
        mov TAIL_H, DPH
        setb F_OVF              ; новый байт вытеснил самый старый
BW_STORE:
        mov DPL, HEAD_L         ; «голова» указывает на место новой записи
        mov DPH, HEAD_H
        mov A, B
        movx @DPTR, A           ; записать отсчёт левым портом IDT7005
        inc DPTR
        mov R6, DPL             ; проверить необходимость заворота головы
        mov R7, DPH
        mov DPTR, #BUF_END
        mov A, R6
        cjne A, DPL, BW_HEAD_KEEP
        mov A, R7
        cjne A, DPH, BW_HEAD_KEEP
        mov DPTR, #BUF_START
        sjmp BW_HEAD_SAVE
BW_HEAD_KEEP:
        mov DPL, R6
        mov DPH, R7
BW_HEAD_SAVE:
        mov HEAD_L, DPL         ; сохранить адрес следующей свободной ячейки
        mov HEAD_H, DPH
        clr F_EMPTY             ; после записи буфер заведомо не пуст
        ret
END
