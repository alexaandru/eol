#!/bin/sh

(echo "Product\tCategory"; eol products -t '{{ range . }}{{.name}}{{"\t"}}{{.category}}{{"\n"}}{{ end }}')| column -t| \
  fzf --preview-window="right:70%" --preview "eol product {1}" --header-lines=1 --footer="EndOfLife.date cli v1.0.0" \
      --border-label " Categories: $(eol categories -f json|jq -r '[.result[].name]|join(", ")') " --preview-label="" \
      --bind 'result:bg-transform-footer: [[ $FZF_MATCH_COUNT -gt 0 ]] && sort {*f2} | uniq -c | sort -nrk2' \
      --bind 'focus:bg-transform-preview-label: echo " $(eol product {1} -f json | jq -r .result.label) "' \
      --footer-label " EndOfLife.date cli "
