#!/bin/bash

# Bash completion for eol CLI tool
# Uses eol's own JSON API and caching for dynamic completion

_eol_get_products() {
    eol products -f json 2>/dev/null | jq -r '.result[]?.name // empty' 2>/dev/null || true
}

_eol_get_categories() {
    eol categories -f json 2>/dev/null | jq -r '.result[]?.name // empty' 2>/dev/null || true
}

_eol_get_tags() {
    eol tags -f json 2>/dev/null | jq -r '.result[]?.name // empty' 2>/dev/null || true
}

_eol_get_identifier_types() {
    eol identifiers -f json 2>/dev/null | jq -r '.result[]?.name // empty' 2>/dev/null || true
}

_eol_get_product_versions() {
    local product="$1"
    [[ -z "${product}" ]] && return 0
    eol product "${product}" -f json 2>/dev/null | jq -r '.result?.releases[]?.name // empty' 2>/dev/null || true
}

_eol_completion() {
    local cur prev words cword
    _init_completion || return

    # Main commands
    local commands="index products product release latest categories tags identifiers cache templates completion help"

    # Cache subcommands
    local cache_cmds="stats clear"

    # Template subcommands
    local template_cmds="export"

    # Completion subcommands
    local completion_cmds="bash zsh"

    # Global flags
    local global_flags="-f --format -t --template --disable-cache --cache-dir --cache-for --template-dir -h --help"

    case ${cword} in
        1)
            # Complete main commands
            local compgen_output
            compgen_output=$(compgen -W "${commands}" -- "${cur}") || true
            mapfile -t COMPREPLY <<< "${compgen_output}"
            ;;
        2)
            case "${prev}" in
                cache)
                    local compgen_output
                    compgen_output=$(compgen -W "${cache_cmds}" -- "${cur}") || true
                    mapfile -t COMPREPLY <<< "${compgen_output}"
                    ;;
                templates)
                    local compgen_output
                    compgen_output=$(compgen -W "${template_cmds}" -- "${cur}") || true
                    mapfile -t COMPREPLY <<< "${compgen_output}"
                    ;;
                completion)
                    local compgen_output
                    compgen_output=$(compgen -W "${completion_cmds}" -- "${cur}") || true
                    mapfile -t COMPREPLY <<< "${compgen_output}"
                    ;;
                product|latest)
                    # Complete with actual product names
                    local products
                    local products_output
                    products_output=$(_eol_get_products) || true
                    mapfile -t products <<< "${products_output}"
                    local compgen_output
                    compgen_output=$(compgen -W "${products[*]}" -- "${cur}") || true
                    mapfile -t COMPREPLY <<< "${compgen_output}"
                    ;;
                categories)
                    # Complete with actual category names
                    local categories
                    local categories_output
                    categories_output=$(_eol_get_categories) || true
                    mapfile -t categories <<< "${categories_output}"
                    local compgen_output
                    compgen_output=$(compgen -W "${categories[*]}" -- "${cur}") || true
                    mapfile -t COMPREPLY <<< "${compgen_output}"
                    ;;
                tags)
                    # Complete with actual tag names
                    local tags
                    local tags_output
                    tags_output=$(_eol_get_tags) || true
                    mapfile -t tags <<< "${tags_output}"
                    local compgen_output
                    compgen_output=$(compgen -W "${tags[*]}" -- "${cur}") || true
                    mapfile -t COMPREPLY <<< "${compgen_output}"
                    ;;
                identifiers)
                    # Complete with identifier types
                    local types
                    local types_output
                    types_output=$(_eol_get_identifier_types) || true
                    mapfile -t types <<< "${types_output}"
                    local compgen_output
                    compgen_output=$(compgen -W "${types[*]}" -- "${cur}") || true
                    mapfile -t COMPREPLY <<< "${compgen_output}"
                    ;;
                release)
                    # Complete with product names for release command
                    local products
                    local products_output
                    products_output=$(_eol_get_products) || true
                    mapfile -t products <<< "${products_output}"
                    local compgen_output
                    compgen_output=$(compgen -W "${products[*]}" -- "${cur}") || true
                    mapfile -t COMPREPLY <<< "${compgen_output}"
                    ;;
                -f|--format)
                    local compgen_output
                    compgen_output=$(compgen -W "text json" -- "${cur}") || true
                    mapfile -t COMPREPLY <<< "${compgen_output}"
                    ;;
                --cache-dir|--template-dir)
                    # Complete directories
                    local compgen_output
                    compgen_output=$(compgen -d -- "${cur}") || true
                    mapfile -t COMPREPLY <<< "${compgen_output}"
                    ;;
                *)
                    local compgen_output
                    compgen_output=$(compgen -W "${global_flags}" -- "${cur}") || true
                    mapfile -t COMPREPLY <<< "${compgen_output}"
                    ;;
            esac
            ;;
        3)
            case "${words[1]}" in
                release)
                    # Third argument for release: complete with versions for the product
                    local product="${words[2]}"
                    if [[ -n "${product}" ]]; then
                        local versions
                        local versions_output
                        versions_output=$(_eol_get_product_versions "${product}") || true
                        mapfile -t versions <<< "${versions_output}"
                        local compgen_output
                        compgen_output=$(compgen -W "${versions[*]}" -- "${cur}") || true
                        mapfile -t COMPREPLY <<< "${compgen_output}"
                    fi
                    ;;
                templates)
                    case "${words[2]}" in
                        export)
                            # Complete directories for export destination
                            local compgen_output
                            compgen_output=$(compgen -d -- "${cur}") || true
                            mapfile -t COMPREPLY <<< "${compgen_output}"
                            ;;
                        *)
                            # Default case for unknown template subcommands
                            ;;
                    esac
                    ;;
                *)
                    # For other commands, offer global flags
                    local compgen_output
                    compgen_output=$(compgen -W "${global_flags}" -- "${cur}") || true
                    mapfile -t COMPREPLY <<< "${compgen_output}"
                    ;;
            esac
            ;;
        *)
            # For remaining positions, offer global flags
            local compgen_output
            compgen_output=$(compgen -W "${global_flags}" -- "${cur}") || true
            mapfile -t COMPREPLY <<< "${compgen_output}"
            ;;
    esac
}

complete -F _eol_completion eol
