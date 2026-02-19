#!/usr/bin/env bash
# Claude Code statusline - Catppuccin Mocha + Powerline (2 lines)

INPUT=$(cat)

# Catppuccin Mocha palette (R;G;B for ANSI true-color)
BASE="30;30;46"       # #1e1e2e
SURFACE0="49;50;68"   # #313244
SURFACE1="69;71;90"   # #45475a
BLUE="137;180;250"    # #89b4fa
GREEN="166;227;161"   # #a6e3a1
YELLOW="249;226;175"  # #f9e2af
RED="243;139;168"     # #f38ba8
PINK="245;194;231"    # #f5c2e7
TEAL="148;226;213"    # #94e2d5
SURFACE2="88;91;112"  # #585b70
MAUVE="203;166;247"   # #cba6f7

SEP_R=$'\ue0b0'   # filled right arrow
SEP_L=$'\ue0b2'   # filled left arrow

fmt_tokens() {
  local n=$1
  if [ "$n" -ge 1000 ]; then
    awk "BEGIN { printf \"%.1fk\", $n / 1000 }"
  else
    printf '%d' "$n"
  fi
}

# Render a left-to-right powerline array to stdout.
# Usage: render_left BG_ARRAY_NAME TEXT_ARRAY_NAME
render_left() {
  eval "local bgs=(\"\${${1}[@]}\")"
  eval "local texts=(\"\${${2}[@]}\")"
  local num="${#bgs[@]}"
  for ((i=0; i<num; i++)); do
    printf "\033[48;2;%sm%b" "${bgs[$i]}" "${texts[$i]}"
    if [ $((i+1)) -lt "$num" ]; then
      printf "\033[0m\033[38;2;%sm\033[48;2;%sm%s" "${bgs[$i]}" "${bgs[$((i+1))]}" "$SEP_R"
    else
      printf "\033[0m\033[38;2;%sm%s\033[0m" "${bgs[$i]}" "$SEP_R"
    fi
  done
}

# ═══════════════════════════════════════════════════════════════════════════════
# LINE 1: model | tokens | context bar | compact warning
# ═══════════════════════════════════════════════════════════════════════════════
L1_BG=(); L1_TEXT=()

# ── Model ────────────────────────────────────────────────────────────────────
MODEL_DISPLAY=$(printf '%s' "$INPUT" | jq -r '.model.display_name // empty')
SHORT_MODEL=$([ -n "$MODEL_DISPLAY" ] \
  && printf '%s' "$MODEL_DISPLAY" | sed 's/^Claude //' \
  || echo "Claude")
L1_BG+=("$SURFACE1")
L1_TEXT+=("\033[38;2;${BLUE}m 󰧑 ${SHORT_MODEL} ")

# ── Tokens ───────────────────────────────────────────────────────────────────
IN_T=$(printf '%s' "$INPUT" | jq -r '.context_window.current_usage.input_tokens // 0')
OUT_T=$(printf '%s' "$INPUT" | jq -r '.context_window.current_usage.output_tokens // 0')
CACHE_R=$(printf '%s' "$INPUT" | jq -r '.context_window.current_usage.cache_read_input_tokens // 0')
CACHE_C=$(printf '%s' "$INPUT" | jq -r '.context_window.current_usage.cache_creation_input_tokens // 0')
CACHE_T=$(( CACHE_R + CACHE_C ))
FIN=$(fmt_tokens "$IN_T"); FOUT=$(fmt_tokens "$OUT_T"); FCACHE=$(fmt_tokens "$CACHE_T")
L1_BG+=("$SURFACE0")
L1_TEXT+=("\033[38;2;${GREEN}m ↑${FIN} \033[38;2;${YELLOW}m↓${FOUT} \033[38;2;${TEAL}m~${FCACHE} ")

# ── Context bar ───────────────────────────────────────────────────────────────
USED_PCT=$(printf '%s' "$INPUT" | jq -r '.context_window.used_percentage // empty')
CTX_SIZE=$(printf '%s' "$INPUT" | jq -r '.context_window.context_window_size // empty')
PCT=0
[ -n "$USED_PCT" ] && [ "$USED_PCT" != "null" ] && PCT=$(printf '%.0f' "$USED_PCT")

if [ -n "$CTX_SIZE" ] && [ "$CTX_SIZE" != "null" ] && [ "$CTX_SIZE" -gt 0 ] 2>/dev/null; then
  [ "$CTX_SIZE" -ge 1000000 ] \
    && CTX_LABEL="$(( CTX_SIZE / 1000000 ))M" \
    || CTX_LABEL="$(( CTX_SIZE / 1000 ))k"
else
  CTX_LABEL=""
fi

if   [ "$PCT" -lt 70 ]; then BAR_COLOR="$GREEN"
elif [ "$PCT" -lt 90 ]; then BAR_COLOR="$YELLOW"
else                          BAR_COLOR="$RED"
fi

SIZE_SUFFIX=""; [ -n "$CTX_LABEL" ] && SIZE_SUFFIX="/${CTX_LABEL}"

if [ "$PCT" -gt 0 ]; then
  FILLED=$(( PCT * 10 / 100 )); [ "$FILLED" -gt 10 ] && FILLED=10; EMPTY=$(( 10 - FILLED ))
  BAR_F=""; BAR_E=""
  for ((i=0; i<FILLED; i++)); do BAR_F+="█"; done
  for ((i=0; i<EMPTY;  i++)); do BAR_E+="░"; done
  CTX_TEXT="\033[38;2;${BAR_COLOR}m ${BAR_F}\033[38;2;${SURFACE2}m${BAR_E}\033[38;2;${BAR_COLOR}m ${PCT}%${SIZE_SUFFIX} "
else
  CTX_TEXT="\033[38;2;${SURFACE2}m ░░░░░░░░░░ \033[38;2;${GREEN}m--%${SIZE_SUFFIX} "
fi
L1_BG+=("$BASE"); L1_TEXT+=("$CTX_TEXT")

# ── Compact warning ───────────────────────────────────────────────────────────
EXCEEDS=$(printf '%s' "$INPUT" | jq -r '.exceeds_200k_tokens // false')
if [ "$EXCEEDS" = "true" ] || [ "$PCT" -ge 85 ]; then
  L1_BG+=("$RED"); L1_TEXT+=("\033[38;2;${BASE}m ⚡ COMPACT ")
fi

# ═══════════════════════════════════════════════════════════════════════════════
# Git info — cached 5s
# Cache: 4 lines: branch / dirty / ahead / behind
# ═══════════════════════════════════════════════════════════════════════════════
GIT_CACHE="/tmp/claude-statusline-git-cache"
NOW=$(date +%s)
GIT_BRANCH=""; GIT_DIRTY=""; GIT_AHEAD=0; GIT_BEHIND=0
USE_CACHE=false

if [ -f "$GIT_CACHE" ]; then
  CACHE_TIME=$(stat -c %Y "$GIT_CACHE" 2>/dev/null || echo 0)
  [ $(( NOW - CACHE_TIME )) -lt 5 ] && USE_CACHE=true
fi

if $USE_CACHE; then
  GIT_BRANCH=$(sed -n '1p' "$GIT_CACHE")
  GIT_DIRTY=$(sed -n '2p' "$GIT_CACHE")
  GIT_AHEAD=$(sed -n '3p' "$GIT_CACHE")
  GIT_BEHIND=$(sed -n '4p' "$GIT_CACHE")
else
  CWD_GIT=$(printf '%s' "$INPUT" | jq -r '.cwd // empty')
  [ -z "$CWD_GIT" ] && CWD_GIT="$PWD"
  GIT_BRANCH=$(git -C "$CWD_GIT" --no-optional-locks rev-parse --abbrev-ref HEAD 2>/dev/null)
  if [ -n "$GIT_BRANCH" ]; then
    if ! git -C "$CWD_GIT" --no-optional-locks diff --quiet 2>/dev/null || \
       ! git -C "$CWD_GIT" --no-optional-locks diff --cached --quiet 2>/dev/null; then
      GIT_DIRTY=" ✦"
    fi
    GIT_AHEAD=$(git  -C "$CWD_GIT" --no-optional-locks rev-list --count @{u}..HEAD 2>/dev/null || echo 0)
    GIT_BEHIND=$(git -C "$CWD_GIT" --no-optional-locks rev-list --count HEAD..@{u} 2>/dev/null || echo 0)
  fi
  printf '%s\n%s\n%s\n%s\n' \
    "$GIT_BRANCH" "$GIT_DIRTY" "${GIT_AHEAD:-0}" "${GIT_BEHIND:-0}" > "$GIT_CACHE"
fi

# ═══════════════════════════════════════════════════════════════════════════════
# LINE 2: git branch (left) + CWD (right-aligned)
# ═══════════════════════════════════════════════════════════════════════════════
L2_BG=(); L2_TEXT=(); L2_VIS=()

if [ -n "$GIT_BRANCH" ]; then
  GIT_LABEL="${GIT_BRANCH}${GIT_DIRTY}"
  [ "${GIT_AHEAD:-0}"  -gt 0 ] && GIT_LABEL+=" ↑${GIT_AHEAD}"
  [ "${GIT_BEHIND:-0}" -gt 0 ] && GIT_LABEL+=" ↓${GIT_BEHIND}"
  L2_BG+=("$PINK")
  L2_TEXT+=("\033[38;2;${BASE}m  ${GIT_LABEL} ")
  L2_VIS+=("  ${GIT_LABEL} ")
fi

# CWD — replace $HOME with ~, truncate if long
CWD_RAW=$(printf '%s' "$INPUT" | jq -r '.cwd // empty')
[ -z "$CWD_RAW" ] && CWD_RAW="$PWD"
if   [[ "$CWD_RAW" == "$HOME" ]];    then CWD_DISP="~"
elif [[ "$CWD_RAW" == "$HOME/"* ]];  then CWD_DISP="~/${CWD_RAW#"$HOME/"}"
else                                       CWD_DISP="$CWD_RAW"
fi
if [ "${#CWD_DISP}" -gt 30 ]; then
  LAST=$(basename "$CWD_RAW"); PARENT=$(basename "$(dirname "$CWD_RAW")")
  CWD_DISP="…/${PARENT}/${LAST}"
  [ "${#CWD_DISP}" -gt 30 ] && CWD_DISP="…/${LAST}"
fi
CWD_TEXT=" ${CWD_DISP} "
CWD_W=$(( ${#CWD_TEXT} + 1 ))   # +1 for SEP_L arrow

# ── Output line 1 ────────────────────────────────────────────────────────────
render_left L1_BG L1_TEXT
printf '\n'

# ── Output line 2 ────────────────────────────────────────────────────────────
TERM_W=${COLUMNS:-0}
[ "$TERM_W" -gt 0 ] 2>/dev/null || TERM_W=$(tput cols 2>/dev/null)
[ "$TERM_W" -gt 0 ] 2>/dev/null || TERM_W=80

# Measure left side visible width
L2_LEFT_W=0
for vis in "${L2_VIS[@]}"; do L2_LEFT_W=$(( L2_LEFT_W + ${#vis} )); done
L2_LEFT_W=$(( L2_LEFT_W + ${#L2_BG[@]} ))   # +1 per arrow

PAD=$(( TERM_W - L2_LEFT_W - CWD_W ))
[ "$PAD" -lt 0 ] && PAD=0

[ ${#L2_BG[@]} -gt 0 ] && render_left L2_BG L2_TEXT

printf '%*s' "$PAD" ''

# CWD: left-pointing arrow in MAUVE, then segment
printf "\033[0m\033[38;2;%sm%s\033[48;2;%sm\033[38;2;%sm%s\033[0m\n" \
  "$MAUVE" "$SEP_L" "$MAUVE" "$BASE" "$CWD_TEXT"
