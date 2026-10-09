#!/usr/bin/env python3
"""Комментарий в Issue студента о результате джобы КМ-1/2/3 (если папка студента ведётся через Issue: students/<ник>/issue).

  scripts/issue_notify.py ok   build/<группа>/<M> [...]   — после публикации: что готово, ссылки на файлы и на диск
  scripts/issue_notify.py fail <ник> [...]                — джоба упала: ссылка на лог

Нужны GH_TOKEN (GITHUB_TOKEN с issues: write), GITHUB_REPOSITORY; ссылка на диск — YADISK_PUBLIC.
Папка прогона — из build/…/.pubdest и .run (publish.sh), на диске — .disk (yadisk_upload.py, если номер сдвинулся).
"""
import json
import os
import re
import subprocess
import sys
import urllib.parse

REPO = os.environ.get("GITHUB_REPOSITORY", "")
RUN = f"{os.environ.get('GITHUB_SERVER_URL', 'https://github.com')}/{REPO}/actions/runs/{os.environ.get('GITHUB_RUN_ID', '')}"
PUBLIC = os.environ.get("YADISK_PUBLIC", "").rstrip("/")
WF = os.environ.get("GITHUB_WORKFLOW", "")


def read(p):
    try:
        with open(p, encoding="utf-8") as f:
            return f.read().strip()
    except OSError:
        return ""


def issue_of(nick):
    n = read(os.path.join("students", nick, "issue"))
    return n if n.isdigit() else ""


def comment(issue, body):
    subprocess.run(["gh", "issue", "comment", issue, "--repo", REPO, "--body-file", "-"], input=body, text=True, check=False)


def q(path):
    return urllib.parse.quote(path)


GUIDE = f"https://github.com/{REPO}/blob/main"
from issue_card import card  # noqa: E402 — общий вид карточек с issue_bot.py


OWN = ("✍ **Своими словами:** когда всё дописано и это последняя сборка — откройте docx и перескажите своими словами общие абзацы "
       "(введение, описания, выводы): тексты у всех из одного шаблона. Числа, таблицы, рисунки и обозначения не трогать; "
       "новая сборка перезапишет docx — сохраните копию. Подробно — [гайд по ПЗ, «Своими словами»](" + GUIDE + "/docs/pz-guide.md).")


def warn_text(left):
    """Красная плашка: при незаполненных «ДОПИШИ» — про них; всё заполнено — только «перечитайте» и «своими словами»."""
    head = ("**Перечитайте весь отчёт целиком.** В нём ещё есть жёлтые пометки «ДОПИШИ» — места, которые нужно написать самому; "
            "отчёт с такими пометками сдавать нельзя." if left != 0 else
            "**Перечитайте весь отчёт целиком** перед сдачей: проверьте свои абзацы, рисунки и таблицы.")
    return head + "\n\n" + OWN


def ok(d):
    nick = read(os.path.join(d, ".nick"))
    if not nick:
        try:
            nick = json.load(open(os.path.join(d, "meta.json"), encoding="utf-8")).get("nick", "")
        except (OSError, ValueError):
            nick = ""
    issue = issue_of(nick) if nick else ""
    dest, run = read(os.path.join(d, ".pubdest")), read(os.path.join(d, ".run"))
    if not issue or not dest or not run:
        return
    path = f"{dest}/{run}"
    blob = f"https://github.com/{REPO}/blob/results/{q(path)}"
    tree = f"https://github.com/{REPO}/tree/results/{q(path)}"
    rawr = f"https://raw.githubusercontent.com/{REPO}/results/{q(path)}"
    files = sorted(os.listdir(os.path.join(d, ".pub"))) if os.path.isdir(os.path.join(d, ".pub")) else []
    link = lambda f, t=None: f"[{t or f}]({blob}/{q(f)})"
    disk = read(os.path.join(d, ".disk")) or path
    where = f"📁 [Яндекс-диск: {disk}]({PUBLIC}/{q(disk)}) · [все файлы прогона]({tree})" if PUBLIC else f"📁 [все файлы прогона]({tree})"
    foot = f"<sub>запуск: {RUN}</sub>"
    sv = read(os.path.join(d, ".send")).splitlines()
    short, ver = (sv[0], sv[1]) if len(sv) >= 2 else ("Фамилия ИО", "1")
    grp, rest = (dest.split("/", 1) + [""])[:2]
    var = rest.split(" ", 1)[0]
    body = f"в теле письма — «группа {grp}, вариант {var}»"
    has = lambda rel: os.path.exists(os.path.join("students", nick, rel))

    if run.startswith("СХЕМА"):
        erc = read(os.path.join(d, ".pub", "erc-summary.txt"))
        png = f"{short} Схема-v{ver}.png"
        comment(issue, card(1, f"✅ Шаг 1 · КМ-1 — схема готова ({run})",
            files=f"[![схема]({rawr}/schematic.png)]({rawr}/schematic.png)\n\n" +
                  " · ".join(link(f, t) for f, t in [(png, f"📤 {png}"), ("schematic.pdf", "схема PDF"), ("perechen-km1.pdf", "перечень для КМ-1"),
                                                     ("perechen.pdf", "перечень ПЭ3"), ("schematic.kicad_pro", "проект KiCad"),
                                                     ("schematic.kicad_sch", "схема KiCad")] if f in files) + f"\n\n{where}",
            student=("ERC: чисто ✅" if not erc else "⚠ ERC — есть замечания, см. `erc-summary.txt`") +
                    f"\n\n✉️ **Сдать КМ-1:** на почту руководителя, тема «МПС-Схема», вложения: **{png}** (ровно этот файл — "
                    "чёрно-белый; не скриншот и не экспорт из KiCad, там цвет), **полный перечень** `perechen.pdf` "
                    "(ТЗ разрешает черновик только из микросхем и разъёмов, но руководители просят весь) и **задание** — "
                    f"ТЗ-2026 и {link('params.md', 'params.md')} с параметрами своего варианта; {body}.\n\n"
                    "Открыть в KiCad: скачать `schematic.kicad_pro`, `schematic.kicad_sch`, `ramka.kicad_wks` в одну папку и открыть проект "
                    f"([подробно]({GUIDE}/README.md#для-студентов)).",
            ai=f"Данные варианта для следующих шагов: {link('params.md')} (адреса, программы, таймеры) · {link('vars.inc')} "
               "(EQU/BIT для ассемблера). Числа не пересчитывай — бери отсюда.",
            nxt=("ПЗ1 и ПЗ2 пересоберутся по этой схеме сами — ждите их карточки." if has("pz/pz1.md") and has("code/prog1.a51") else
                 "ПЗ1 пересоберётся по этой схеме сама — ждите её карточку; дальше — **Шаг 3 · КМ-3, `/км3`**." if has("pz/pz1.md") else
                 "**Шаг 2 · КМ-2 — напишите `/км2`**: появится заготовка ПЗ1, в ней дописать 2 абзаца."),
            foot=foot))
        return

    if run.startswith("ПЗ1"):
        m = re.search(r"Мест «ДОПИШИ»: \d+, осталось заполнить: (\d+)", read(os.path.join(d, "pz1", "report.md")))
        left = int(m.group(1)) if m else -1
        docs = [f for f in files if f.endswith(".docx")] + [f for f in files if f.endswith(".pdf")]
        comment(issue, card(2, f"{'✅' if left == 0 else '📝'} Шаг 2 · КМ-2 — ПЗ1 ({run})",
            files=" · ".join(link(f, ("📄 " if f.endswith(".docx") else "👁 ") + f) for f in docs) + f"\n\n{where}",
            student=("✅ Все места «ДОПИШИ» заполнены." if left == 0 else
                     f"📝 Осталось мест «ДОПИШИ»: **{left}** — их дописывает нейронка или вы (`pz1.md`, команда `/км2`).") +
                    f"\n\n✉️ **Сдать КМ-2:** docx → «Сохранить как PDF» → `{short} ПЗ1-v{ver}.pdf` → на почту руководителя, тема «МПС-ПЗ1»; {body}.",
            ai=(f"Возьми текущий [`pz1.md`](https://raw.githubusercontent.com/{REPO}/main/students/{nick}/pz/pz1.md), заполни разделы под "
                "подсказками «✍» своими словами от лица студента и пришли файл целиком: `/км2` + блок ```` ```pz1.md ````. "
                if left != 0 else "Всё заполнено. ") +
               "Перед сдачей предложи студенту пересказ общих абзацев (факты и числа не меняй) — что можно и что нельзя трогать: "
               f"[docs/finish-guide.md]({GUIDE}/docs/finish-guide.md), разд. 2.",
            nxt="**Шаг 3 · КМ-3 — напишите `/км3`**: заготовки программ, ПЗ2 и схем алгоритмов, гайд по коду.",
            warn=warn_text(left), foot=foot))
        return

    # ПЗ2 + программы (КМ-3)
    m = re.search(r"Мест «ДОПИШИ»: \d+, осталось заполнить: (\d+)", read(os.path.join(d, "pz2", "report.md")))
    left = int(m.group(1)) if m else -1
    code = read(os.path.join(d, ".pub", "программы", "report.md"))
    bad = len(re.findall(r"^## ❌", code, re.M)) if code else 0
    pd = os.path.join(d, ".pub", "программы")
    robot = sorted(f for f in os.listdir(pd) if f.endswith(".txt")) if os.path.isdir(pd) else []
    src = sorted(f for f in os.listdir(pd) if f.endswith(".a51")) if os.path.isdir(pd) else []
    docs = [f for f in files if f.endswith(".docx")] + [f for f in files if f.endswith(".pdf")]
    fl = []
    if code:
        fl.append(f"{'❌' if bad else '✅'} {link('программы/report.md', 'отчёт проверки программ')}")
    if robot:
        fl.append("📤 файлы для робота (`vars.inc` вклеен): " + " · ".join(link('программы/' + f, f) for f in robot))
    if src:
        fl.append("исходники: " + " · ".join(link('программы/' + f, f) for f in src))
    if docs:
        fl.append(" · ".join(link(f, ("📄 " if f.endswith(".docx") else "👁 ") + f) for f in docs))
    st = ["❌ **В программах есть ошибки** — что именно, в отчёте проверки; их исправляет нейронка (или вы) и присылает заново `/км3`."
          if bad else "✅ Программы проходят нашу проверку в эмуляторе на модели вашей схемы."]
    if left > 0:
        st.append(f"📝 В ПЗ2 осталось мест «ДОПИШИ»: **{left}** (`pz2.md`, команда `/км3`).")
    elif left == 0:
        st.append("✅ В ПЗ2 все места «ДОПИШИ» заполнены.")
    w = re.findall(r"^- (.+)$", read(os.path.join(d, "pz2", "report.md")).split("## Предупреждения")[-1], re.M) \
        if "## Предупреждения" in read(os.path.join(d, "pz2", "report.md")) else []
    if w:
        st.append("⚠️ **Схемы алгоритмов:**\n" + "\n".join(f"- {x}" for x in w))
    st.append(f"✉️ **Сдать:** 1) **ПЗ2** — docx → «Сохранить как PDF» → `{short} ПЗ2-v{ver}.pdf` → на почту ОСЭП руководителя, тема «МПС-ПЗ2»; {body}.\n"
              "2) **После рассмотрения ПЗ2** — программы: " + (", ".join(f"`{c}`" for c in robot) if robot else "файлы `Фамилия ИО-код-n.txt`") +
              f" (имена не менять — по ним проверяет робот), тема «МПС-код», на почту ОСЭП руководителя; {body}.")
    ai = (f"Исправь ошибки по [отчёту]({blob}/{q('программы/report.md')}) и пришли исправленные файлы `/км3` "
          "(каждый — блоком с именем файла в первой строке)." if bad else "Программы проходят.")
    if left > 0:
        ai += (f" Допиши «ДОПИШИ» в [`pz2.md`](https://raw.githubusercontent.com/{REPO}/main/students/{nick}/pz/pz2.md) "
               "и пришли `/км3` + блок ```` ```pz2.md ````.")
    ai += f" Гайд по коду — [docs/code-guide.md]({GUIDE}/docs/code-guide.md)."
    if not bad and left == 0:
        ai += (f"\n\n**Всё ✅ — не заканчивай сессию.** По [docs/finish-guide.md]({GUIDE}/docs/finish-guide.md) предложи студенту "
               "продолжить: пояснения к алгоритмам ПЗ2 своими словами (`flow.*` в `pz2.md`), пересказ общих абзацев ПЗ, "
               "разбор схемы и кода к защите, замечания руководителя по мере прихода.")
    comment(issue, card(3, f"{'❌' if bad else '✅'} Шаг 3 · КМ-3 — программы и ПЗ2 ({run})",
        files="\n".join(f"- {x}" for x in fl) + f"\n\n{where}",
        student="\n\n".join(st), ai=ai, warn=warn_text(left), extra=km3_guide(d, var),
        nxt=("Исправить программы и прислать `/км3`." if bad else
             "Сдать ПЗ2, после её рассмотрения — программы. Ответ руководителя или робота — `/замечание КМ-3 <дословно>`."),
        foot=foot))


def km3_guide(d, var):
    """Памятка КМ-3: что сдаётся, какие программы у варианта (params.md), как проверяем мы и как сдавать (ТЗ-2026, разд. 2.2)."""
    progs = []
    for line in read(os.path.join(d, "params.md")).splitlines():
        m = re.match(r"^\|\s*([123])\s*\|\s*(.+?)\s*\|\s*$", line)
        if m:
            progs.append(f"   {m.group(1)}. {m.group(2)}")
    lines = ["\n<details><summary><b>📘 КМ-3 — что это и как сдавать (раскрыть)</b></summary>\n",
             "**Что сдаётся:** три программы на ассемблере 8051 по табл. 1 ТЗ и ПЗ2 — описание программной части (по этим программам)."]
    if progs:
        lines.append(f"\n**Твои программы (вариант {var}):**\n" + "\n".join(progs))
    lines += [
        "\n**Что даётся:** заготовки по рис. 7 ТЗ под вариант (`/км3` → `code/prog1-3.a51`), `vars.inc` с адресами, битами и "
        "константами варианта (в файлы для робота вклеивается сам), гайд — `docs/code-guide.md` (правила робота, устройства схемы глазами программы).",
        "\n**Требования ТЗ к каждой программе:** заглушки `nop` + `reti` на всех неиспользуемых прерываниях; инициализация сразу после Reset; "
        "процедура задачи с комментарием `%proc%`, вызванная `call` (без бесконечных циклов внутри); последняя команда `jmp $ ; %stop%`; "
        "комментарии — по смыслу, а не перевод мнемоник; у каждой процедуры — комментарий о назначении, входах и выходах.",
        "\n**Как проверяем мы** (каждый прогон): свой ассемблер (байты = MCU 8051 IDE) и эмулятор 8051 с моделью **твоей** схемы "
        "(клавиатура, буфер IDT7005, индикатор, регистр Y2, стробы, дешифратор): сценарии из ТЗ, такты таймеров, правила рис. 7 → отчёт ✅/❌ выше. "
        "Это наша проверка, не робот кафедры: как именно проверяет робот, неизвестно — его ответ главный.",
        "\n**Как прислать код сюда:** комментарий `/км3` и под ним файлы блоками, в первой строке блока — имя файла "
        "(```` ```prog1.a51 ````, ```` ```prog2.a51 ````, ```` ```pz2.md ````) — проверка и ПЗ2 пересоберутся сами.",
        "\n**Как сдавать (ТЗ, разд. 2.2):**\n"
        "   1. Сначала **ПЗ2** — PDF на почту ОСЭП руководителя, тема «МПС-ПЗ2».\n"
        "   2. **После рассмотрения ПЗ2** (с учётом замечаний) — **программы**: файлы для робота (ссылки выше, `.txt`, UTF-8) на почту ОСЭП "
        "руководителя, тема «МПС-код», имена «Фамилия ИО-код-n» не менять, в теле — группа и вариант.\n"
        "   3. Проверка автоматическая — робот кафедры в сети МЭИ `10.3.170.2:8051` (работает после 7-й недели); ответ «принято» или ошибка. "
        "Ответ робота запишите сюда: `/замечание КМ-3 <что ответил робот>` — сверим с нашей проверкой.",
        "\n</details>"]
    return "\n".join(lines)


def fail(nick):
    issue = issue_of(nick)
    if issue:
        comment(issue, f"## ❌ {WF}: не получилось\n\nЧто именно — [в логе]({RUN}) (раскройте шаг с красным крестиком).\n\n"
                       "**Частое:** «схемы ещё нет» → `/км1`; «схема устарела» (меняли форму или правки схемы) → `/км1`, потом `/км2`, `/км3`.")


if __name__ == "__main__":
    mode, args = sys.argv[1], sys.argv[2:]
    for a in args:
        (ok if mode == "ok" else fail)(a.rstrip("/"))
