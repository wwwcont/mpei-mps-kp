; Рязанцев И.В., А-12-23, 17, v1
$INCLUDE (vars.inc)             ; адреса, биты, константы варианта; в файл для робота mpscode вклеит его текст
DSEG AT 30h
; переменные не нужны: указатели и флаги заданы в vars.inc
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
        mov HEAD_L, #LOW(BUF_START)     ; голова — в начало буфера
        mov HEAD_H, #HIGH(BUF_START)
        mov TAIL_L, #LOW(BUF_START)     ; хвост — туда же: буфер пуст
        mov TAIL_H, #HIGH(BUF_START)
        setb F_EMPTY            ; данных нет
        clr F_OVF               ; ничего не терялось
        mov A, #55h             ; пример отсчёта
        call BufWrite ; %proc%
        jmp $ ; %stop%

; BufWrite — запись отсчёта в кольцевой буфер в ячейку «головы».
; Если буфер заполнен целиком (голова = хвост и буфер не пуст), самый старый отсчёт выбрасывается:
; хвост сдвигается на следующую ячейку и ставится флаг переполнения.
; Вход: A — отсчёт. Выход: отсчёт в буфере, голова сдвинута, F_EMPTY = 0; F_OVF = 1, если отсчёт затёр самый старый.
; Портит: ничего (A, B, DPTR, PSW сохраняются).
BufWrite:
        push PSW                ; флаги вызывающей программы не трогаем
        push DPL                ; DPTR тоже сохраняем — он нужен для movx
        push DPH
        push B                  ; B будет временным местом для отсчёта
        mov B, A                ; отсчёт пока отложим в B
        jb F_EMPTY, BwStore     ; пустой буфер не может быть полным
        mov A, HEAD_L           ; полон ли: голова догнала хвост?
        cjne A, TAIL_L, BwStore
        mov A, HEAD_H
        cjne A, TAIL_H, BwStore
        mov DPL, TAIL_L         ; полон — выбрасываем самый старый отсчёт
        mov DPH, TAIL_H
        call BufNext
        mov TAIL_L, DPL
        mov TAIL_H, DPH
        setb F_OVF              ; отметили потерю данных
BwStore:
        mov DPL, HEAD_L
        mov DPH, HEAD_H
        mov A, B
        movx @DPTR, A           ; кладём отсчёт в ячейку головы
        call BufNext
        mov HEAD_L, DPL         ; голова смотрит на следующую свободную ячейку
        mov HEAD_H, DPH
        clr F_EMPTY             ; теперь в буфере точно что-то есть
        mov A, B                ; вернули отсчёт в A
        pop B                   ; восстанавливаем всё, что сохранили, в обратном порядке
        pop DPH
        pop DPL
        pop PSW
        ret                     ; запись закончена

; BufNext — следующая ячейка кольцевого буфера: после последней (BUF_END - 1) идёт первая (BUF_START).
; Вход: DPTR — адрес ячейки буфера. Выход: DPTR — адрес следующей ячейки. Портит: A, флаг C.
BufNext:
        inc DPTR
        mov A, DPL
        cjne A, #LOW(BUF_END), BnDone   ; ещё не вышли за конец — готово
        mov A, DPH
        cjne A, #HIGH(BUF_END), BnDone
        mov DPTR, #BUF_START    ; вышли за конец — заворачиваем на начало
BnDone:
        ret                     ; DPTR — следующая ячейка
END
