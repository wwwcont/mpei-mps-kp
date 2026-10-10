; Казарин А.А., А-12-23, 8, v1
$INCLUDE (vars.inc)             ; адреса, биты, константы варианта; в файл для робота mpscode вклеит его текст
DSEG AT 30h
KCOL:   DS 1                    ; текущий столбец 0…3
KROWS:  DS 1                    ; маска нажатых строк
KROWNO: DS 1                    ; номер строки для таблицы кодов
KCOUNT: DS 1                    ; число найденных клавиш
KCODE:  DS 1                    ; код единственной клавиши
CSEG AT 0h
        sjmp START
org 03h                         ; INT0 — нажатие клавиши: процедура вызывается отсюда
        call KbScan ; %proc%
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
        setb PIN_CSEN              ; включить дешифратор адресов
        mov DPTR, #ADR_KB          ; подготовить регистр столбцов
        clr A
        movx @DPTR, A              ; все столбцы активны для INT0
        setb IT0                   ; прерывание INT0 по спаду
        setb EX0                   ; разрешить клавиатурное прерывание
        setb EA                    ; разрешить глобальные прерывания
        jmp $ ; %stop%

; KbScan — сканирование матрицы клавиатуры по столбцам (обработчик INT0).
; Вход: нет. Выход: A — код единственной нажатой клавиши по табл. 3 ТЗ или 0FFh.
; Портит: A, DPTR, флаги PSW и временные переменные в DSEG.
KbScan:
        push PSW                  ; сохранить рабочие флаги и адрес данных
        push DPL
        push DPH
        mov KCOUNT, #0             ; начать подсчёт нажатий
        mov KCODE, #0FFh           ; значение на случай неоднозначного ввода
        mov KCOL, #0               ; начать с первого столбца
KbColumn:
        mov A, KCOL
        mov DPTR, #ColMask         ; выбрать маску столбца
        movc A, @A+DPTR
        mov DPTR, #ADR_KB
        movx @DPTR, A              ; опустить только выбранный столбец
        movx A, @DPTR              ; прочитать строки матрицы
        anl A, #((1 SHL KB_ROWS) - 1)
        cpl A
        anl A, #((1 SHL KB_ROWS) - 1)
        mov KROWS, A

        mov A, KROWS
        jnb ACC.0, KRow2           ; строка 1 замкнулась на столбец?
        mov KROWNO, #0
        call RecordKey
KRow2:
        mov A, KROWS
        jnb ACC.1, KRow3           ; строка 2 замкнулась на столбец?
        mov KROWNO, #1
        call RecordKey
KRow3:
        mov A, KROWS
        jnb ACC.2, KNextColumn     ; строка 3 замкнулась на столбец?
        mov KROWNO, #2
        call RecordKey
KNextColumn:
        inc KCOL                   ; перейти к следующему столбцу
        mov A, KCOL
        cjne A, #KB_COLS, KbColumn

        mov DPTR, #ADR_KB
        clr A
        movx @DPTR, A              ; вернуть матрицу в режим ожидания
        clr IE0                    ; убрать спады INT0 во время сканирования
        mov A, KCOUNT
        cjne A, #1, KbInvalid      ; принимать только одно нажатие
        mov A, KCODE
        pop DPH                    ; восстановить регистры, не меняя код в A
        pop DPL
        pop PSW
        ret
KbInvalid:
        mov A, #0FFh
        pop DPH                    ; восстановить регистры, не меняя код в A
        pop DPL
        pop PSW
        ret

; RecordKey — учесть одну найденную замкнутую точку матрицы.
; Вход: KCOL и KROWNO. Выход: KCOUNT увеличен; KCODE задан для первой клавиши.
; Портит: A, DPTR, флаги PSW.
RecordKey:
        inc KCOUNT                 ; каждое замыкание — отдельное нажатие
        mov A, KCOUNT
        cjne A, #1, RecordDone     ; код сохраняется только для первой клавиши
        mov A, KCOL
        rl A
        add A, KCOL                ; индекс столбца умножен на число строк
        add A, KROWNO
        mov DPTR, #KeyMap
        movc A, @A+DPTR            ; получить код клавиши по таблице раскладки
        mov KCODE, A
RecordDone:
        ret

; ColMask: при опросе один столбец имеет ноль, остальные — единицы.
ColMask:
        db 0FEh, 0FDh, 0FBh, 0F7h
; KeyMap расположена столбцами: 1/4/7, 2/5/8, 3/6/9, Т/С/0.
KeyMap:
        db 1, 4, 7, 2, 5, 8, 3, 6, 9, 10, 11, 0
END
