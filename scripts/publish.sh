#!/usr/bin/env bash
# Выкладывает результаты сборки в ветку results и пишет сводку запуска (GitHub Actions).
# Использование: RUN_KIND=СХЕМА|ПЗ1|ПЗ2 [RUN_ROOT=_пробы] scripts/publish.sh build/<группа>/<вариант> [...]
#
# Раскладка ветки results (и Яндекс-диска — один в один):
#   <группа>/<M> <Фамилия И.О.>/СХЕМА-N/            — КМ-1: схема PNG/PDF/KiCad, перечни, params, vars.inc, ERC, sheet.json, meta.json
#   <группа>/<M> <Фамилия И.О.>/ПЗ1-K по СХЕМА-N/    — КМ-2: «Фамилия ИО ПЗ1.docx», PDF «просмотр», рисунки
#   <группа>/<M> <Фамилия И.О.>/ПЗ2-K по СХЕМА-N/    — КМ-3: «Фамилия ИО ПЗ2.docx», PDF «просмотр», программы/
#   _пробы/<группа>/<M> <Фамилия И.О.>/СХЕМА-N/      — джоба «Проба варианта» (не сдача)
# У каждого вида свой счётчик, прошлые прогоны не перетираются. «по СХЕМА-N» — по какой схеме собрано (build/…/.schema).
# Папка студента — build/…/.dest (пишет scripts/schema_fetch.sh или meta.json схемы).
# Готовая папка копируется в build/…/.pub, её путь — в .pubdest и .run: yadisk_upload.py заливает её на диск как есть.
# SUMMARY=0 — не писать сводку запуска.
set -euo pipefail
kind="${RUN_KIND:?нужен RUN_KIND: СХЕМА, ПЗ1 или ПЗ2}"
case "$kind" in СХЕМА|ПЗ1|ПЗ2) ;; *) echo "RUN_KIND: $kind — ждём СХЕМА, ПЗ1 или ПЗ2" >&2; exit 1 ;; esac
root="${RUN_ROOT:+$RUN_ROOT/}"

# папка студента: .dest, иначе meta.json схемы
dest_of() {
  if [ -s "$1/.dest" ]; then cat "$1/.dest"; return; fi
  [ -f "$1/meta.json" ] && python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["dest"])' "$1/meta.json" && return
  return 1
}

dirs=()
for d in "$@"; do
  d="${d%/}"
  if ! dest_of "$d" >/dev/null 2>&1; then echo "пропуск $d: неизвестно, чья это папка (нет .dest / meta.json)" >&2; continue; fi
  case "$kind" in
    СХЕМА) [ -f "$d/params.md" ] ;;
    ПЗ1) [ -d "$d/pz1" ] ;;
    ПЗ2) [ -d "$d/pz2" ] || [ -d "$d/code" ] ;;
  esac && dirs+=("$d") || echo "пропуск $d: нет результатов $kind" >&2
done
[ ${#dirs[@]} -gt 0 ] || { echo "нечего публиковать" >&2; exit 0; }
set -- "${dirs[@]}"

repo="${GITHUB_REPOSITORY:?нужен GITHUB_REPOSITORY}"
tree="https://github.com/$repo/blob/results"
summary="${GITHUB_STEP_SUMMARY:-/dev/stdout}"
keep=(params.md params.json vars.inc meta.json sheet.json schematic.png schematic.pdf schematic.kicad_sch schematic.kicad_pro ramka.kicad_wks
      erc-summary.txt erc.rpt perechen.pdf perechen-km1.pdf perechen.md perechen-1.png perechen-2.png perechen-3.png perechen-km1-1.png)
url() { printf '%s' "$1" | sed 's/ /%20/g'; }

work="$(mktemp -d)"
git config --global user.name "github-actions[bot]"
git config --global user.email "41898282+github-actions[bot]@users.noreply.github.com"

# несколько запусков КМ-1/2/3 сразу (перезапуск всех студентов) пушат в results наперегонки: повторов много, пауза случайная,
# а если так и не вышло — запуск падает (иначе карточка уходит в Issue, а в results схемы нет: 09.10.2026, Судьбинин СХЕМА-2)
published=""
for attempt in $(seq 1 12); do
  rm -rf "$work/r" "$work/idx0" && mkdir -p "$work/r"
  lease=""
  if git fetch -q origin results 2>/dev/null; then
    lease="$(git rev-parse FETCH_HEAD)"
    GIT_INDEX_FILE="$work/idx0" git --work-tree="$work/r" checkout FETCH_HEAD -- . 2>/dev/null || true
    # старые раскладки (до 01.10.2026): <группа>/<M>/… без ФИО — убрать
    find "$work/r" -mindepth 2 -maxdepth 2 -type d | grep -E '/[0-9]+$' | while read -r old; do rm -rf "$old"; done || true
  fi
  for d in "$@"; do
    dest="$root$(dest_of "$d")"
    v="$work/r/$dest"
    mkdir -p "$v"
    last="$(ls "$v" 2>/dev/null | sed -nE "s/^$kind-([0-9]+)( .*)?$/\1/p" | sort -n | tail -1 || true)"
    n=$(( ${last:-0} + 1 ))
    name="$kind-$n"
    if [ "$kind" != СХЕМА ] && [ -s "$d/.schema" ]; then name="$name по $(cat "$d/.schema")"; fi
    out="$v/$name"
    mkdir -p "$out"
    # всё, кроме исходника pandoc (*.md), но с отчётом
    flat() { find "$1" -maxdepth 1 -type f \( ! -name '*.md' -o -name report.md \) -exec cp {} "$2/" \; ; }
    case "$kind" in
      СХЕМА) for f in "${keep[@]}"; do [ -f "$d/$f" ] && cp "$d/$f" "$out/"; done ;;
      ПЗ1)   flat "$d/pz1" "$out" ;;
      ПЗ2)   [ -d "$d/pz2" ] && flat "$d/pz2" "$out"
             if [ -d "$d/code" ]; then mkdir -p "$out/программы" && flat "$d/code" "$out/программы"; fi ;;
    esac
    # история замечаний руководителя студента — копией в каждый прогон
    nick="$(cat "$d/.nick" 2>/dev/null || python3 -c 'import json,sys; print(json.load(open(sys.argv[1])).get("nick",""))' "$d/meta.json" 2>/dev/null || true)"
    [ -n "$nick" ] && [ -f "students/$nick/remarks.md" ] && cp "students/$nick/remarks.md" "$out/Замечания.md"
    # имена файлов для отправки — по ТЗ-2026 (разд. 2.1, 2.2): «Фамилия ИО Смысл-vN», N = 1 + замечаний к этому КМ в remarks.md
    case "$kind" in СХЕМА) km=1 ;; ПЗ1) km=2 ;; *) km=3 ;; esac
    ver=$(( $(grep -cE "^## .* — КМ-$km — " "students/$nick/remarks.md" 2>/dev/null || true) + 1 ))
    short="$(python3 - "$d" <<'PY'
import json, os, re, sys
d = sys.argv[1]
fio = ""
try:
    fio = json.load(open(os.path.join(d, "meta.json"), encoding="utf-8")).get("student", "")
except (OSError, ValueError):
    dest = open(os.path.join(d, ".dest"), encoding="utf-8").read().strip() if os.path.exists(os.path.join(d, ".dest")) else ""
    fio = re.sub(r"^\d+\s+", "", dest.split("/")[-1])
print(fio.replace(".", "").strip())
PY
)"
    if [ -n "$short" ]; then
      case "$kind" in
        СХЕМА) [ -f "$out/schematic.png" ] && cp "$out/schematic.png" "$out/$short Схема-v$ver.png" ;;
        ПЗ1|ПЗ2)
          for f in "$out/"*.docx "$out/"*" — просмотр.pdf"; do
            [ -f "$f" ] || continue
            b="$(basename "$f")"; nb="${b/$kind/$kind-v$ver}"
            [ "$b" != "$nb" ] && mv "$f" "$out/$nb"
          done ;;
      esac
      printf '%s\n%s\n' "$short" "$ver" > "$d/.send"
    fi
    printf '%s\nworkflow: %s, запуск %s\nкоммит: %s\nдата: %s\n' "$name" "${GITHUB_WORKFLOW:-local}" \
      "${GITHUB_SERVER_URL:-}/${repo}/actions/runs/${GITHUB_RUN_ID:-}" "${GITHUB_SHA:-}" "$(date -u '+%Y-%m-%d %H:%M UTC')" > "$d/run.txt"
    cp "$d/run.txt" "$out/"
    echo "$name" > "$d/.run"
    echo "$dest" > "$d/.pubdest"
    rm -rf "$d/.pub" && cp -R "$out" "$d/.pub"
  done
  # оглавление
  {
    echo "# Результаты генератора"
    echo
    echo "Ветка обновляется автоматически (джобы КМ-1/2/3). Руками не править. Исходники — ветка \`main\`."
    echo
    echo "Папка студента — \`<группа>/<M> <Фамилия И.О.>/\`: \`СХЕМА-N\` (КМ-1), \`ПЗ1-K по СХЕМА-N\` (КМ-2),"
    echo "\`ПЗ2-K по СХЕМА-N\` (КМ-3: ПЗ2 и программы). Прошлые прогоны не перетираются. В таблице — последние."
    echo "Пробы вариантов (не сдача) — в \`_пробы/\`."
    echo
    echo "| Группа | Вариант, студент | КМ-1: схема | Перечень | ERC | КМ-2: ПЗ1 | КМ-3: ПЗ2 | КМ-3: программы |"
    echo "| --- | --- | --- | --- | --- | --- | --- | --- |"
    (cd "$work/r" && find . -mindepth 2 -maxdepth 2 -type d ! -path './_*' ! -path './.*' | sed 's|^\./||' | sort -t/ -k1,1 -k2,2n) | while read -r n; do
      lastof() { ls "$work/r/$n" 2>/dev/null | sed -nE "s/^$1-([0-9]+)( .*)?$/\1 &/p" | sort -n | tail -1 | cut -d' ' -f2- || true; }
      sch="—" pe="—" e="—" pz1="—" pz2="—" code="—"
      k="$(lastof СХЕМА)"
      if [ -n "$k" ]; then
        p="$n/$k"
        sch="[$k]($(url "$p")): [PNG]($(url "$p/schematic.png")) · [PDF]($(url "$p/schematic.pdf")) · [KiCad]($(url "$p/schematic.kicad_sch"))"
        pe="[ПЭ3]($(url "$p/perechen.pdf")) · [КМ-1]($(url "$p/perechen-km1.pdf"))"
        e="чисто"; [ -s "$work/r/$p/erc-summary.txt" ] && e="⚠ есть замечания"
      fi
      k="$(lastof ПЗ1)"
      if [ -n "$k" ]; then
        f="$(ls "$work/r/$n/$k/"*.docx 2>/dev/null | head -1 || true)"
        pz1="[$k]($(url "$n/$k/$(basename "${f:-.}")"))"; [ -n "$f" ] || pz1="[$k]($(url "$n/$k"))"
      fi
      k="$(lastof ПЗ2)"
      if [ -n "$k" ]; then
        f="$(ls "$work/r/$n/$k/"*.docx 2>/dev/null | head -1 || true)"
        pz2="[$k]($(url "$n/$k/$(basename "${f:-.}")"))"; [ -n "$f" ] || pz2="[$k]($(url "$n/$k"))"
        r="$work/r/$n/$k/программы/report.md"
        if [ -f "$r" ]; then
          code="[отчёт]($(url "$n/$k/программы/report.md"))"; grep -q "^## ❌" "$r" && code="❌ $code"
        fi
      fi
      echo "| ${n%%/*} | ${n#*/} | $sch | $pe | $e | $pz1 | $pz2 | $code |"
    done
  } > "$work/r/README.md"

  idx="$work/index"
  rm -f "$idx"
  GIT_INDEX_FILE="$idx" git --work-tree="$work/r" add -A .
  t="$(GIT_INDEX_FILE="$idx" git write-tree)"
  c="$(git commit-tree "$t" -m "results: ${GITHUB_RUN_ID:-local} (${GITHUB_SHA:0:7})")"
  args=(--force)
  [ -n "$lease" ] && args=(--force-with-lease="refs/heads/results:$lease")
  if git push -q "${args[@]}" origin "$c:refs/heads/results"; then published=1; break; fi
  echo "results: гонка с другим запуском, повтор $attempt" >&2
  sleep $((3 + RANDOM % 13))
done
[ -n "$published" ] || { echo "::error::results: не удалось опубликовать за 12 попыток — перезапусти джобу" >&2; exit 1; }

[ "${SUMMARY:-1}" = "0" ] && exit 0
# сводка на странице запуска; картинка — по коммиту, а не по ветке (raw-CDN кэширует ветку минутами)
raw="https://raw.githubusercontent.com/$repo/$c"
for d in "$@"; do
  n="$(cat "$d/.pubdest")/$(cat "$d/.run")"
  {
    echo "## $n"
    case "$kind" in
      СХЕМА)
        if [ -s "$d/erc-summary.txt" ]; then echo "⚠ ERC:"; echo '```'; cat "$d/erc-summary.txt"; echo '```'; else echo "ERC: чисто"; fi
        echo
        echo "[Схема PDF]($(url "$tree/$n/schematic.pdf")) · [KiCad]($(url "$tree/$n/schematic.kicad_sch")) · [Перечень ПЭ3]($(url "$tree/$n/perechen.pdf")) · [Перечень для КМ-1]($(url "$tree/$n/perechen-km1.pdf")) · [vars.inc]($(url "$tree/$n/vars.inc")) · [все файлы]($(url "https://github.com/$repo/tree/results/$n"))"
        echo
        echo "[![схема]($(url "$raw/$n/schematic.png"))]($(url "$raw/$n/schematic.png"))"
        echo
        echo "<details><summary>Параметры варианта</summary>"
        echo
        cat "$d/params.md"
        echo
        echo "</details>" ;;
      *)
        echo "[все файлы]($(url "https://github.com/$repo/tree/results/$n"))" ;;
    esac
    echo
  } >> "$summary"
done
