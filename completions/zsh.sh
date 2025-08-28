#!/usr/bin/env zsh

_eol_get_products() {
    local products
    products=(${(f)"$(eol products -f json 2>/dev/null | jq -r '.result[]?.name // empty' 2>/dev/null)"})
    print -l $products
}

_eol_get_categories() {
    local categories
    categories=(${(f)"$(eol categories -f json 2>/dev/null | jq -r '.result[]?.name // empty' 2>/dev/null)"})
    print -l $categories
}

_eol_get_tags() {
    local tags
    tags=(${(f)"$(eol tags -f json 2>/dev/null | jq -r '.result[]?.name // empty' 2>/dev/null)"})
    print -l $tags
}

_eol_get_identifier_types() {
    local types
    types=(${(f)"$(eol identifiers -f json 2>/dev/null | jq -r '.result[]?.name // empty' 2>/dev/null)"})
    print -l $types
}

_eol_get_product_versions() {
    local product=$1
    [[ -z "$product" ]] && return
    local versions
    versions=(${(f)"$(eol product "$product" -f json 2>/dev/null | jq -r '.result?.releases[]?.name // empty' 2>/dev/null)"})
    print -l $versions
}

_eol_products() {
    local products
    products=($(_eol_get_products))
    _describe 'products' products
}

_eol_categories() {
    local categories
    categories=($(_eol_get_categories))
    _describe 'categories' categories
}

_eol_tags() {
    local tags
    tags=($(_eol_get_tags))
    _describe 'tags' tags
}

_eol_identifier_types() {
    local types
    types=($(_eol_get_identifier_types))
    _describe 'identifier types' types
}

_eol_product_versions() {
    local product=$1
    [[ -z "$product" ]] && return
    local versions
    versions=($(_eol_get_product_versions "$product"))
    _describe 'versions' versions
}

_eol() {
    local context state state_descr line
    typeset -A opt_args

    _arguments -C \
        '(-f --format)'{-f,--format}'[Output format]:format:(text json)' \
        '(-t --template)'{-t,--template}'[Inline template]:template:' \
        '--templates-dir[Template directory]:directory:_directories' \
        '(-h --help)'{-h,--help}'[Show help]' \
        '1: :_eol_commands' \
        '*:: :->args'

    case $state in
        args)
            case $words[1] in
                product|latest)
                    _eol_products
                    ;;
                release|release-badge)
                    case $CURRENT in
                        2)
                            _eol_products
                            ;;
                        3)
                            _eol_product_versions $words[2]
                            ;;
                    esac
                    ;;
                category)
                    case $CURRENT in
                        2)
                            _eol_categories
                            ;;
                    esac
                    ;;
                tag)
                    case $CURRENT in
                        2)
                            _eol_tags
                            ;;
                    esac
                    ;;
                identifier)
                    case $CURRENT in
                        2)
                            _eol_identifier_types
                            ;;
                    esac
                    ;;
            esac
            ;;
    esac
}

_eol_commands() {
    local commands=(
        'index:Show API endpoints'
        'products:List all products'
        'products-full:List all products with detailed information'
        'product:Get details for a specific product'
        'release:Get specific release information'
        'release-badge:Generate SVG badge for specific release'
        'latest:Get latest release information'
        'categories:List all categories'
        'category:List products in a specific category'
        'tags:List all tags'
        'tag:List products with a specific tag'
        'identifiers:List all identifier types'
        'identifier:List identifiers by type'
        'templates-export:Export templates to default location or specified directory'
        'completion:Generate shell completion scripts (auto-detects shell)'
        'completion-bash:Generate bash completion script'
        'completion-zsh:Generate zsh completion script'
        'version:Show version information'
        'help:Show help message'
    )
    _describe 'commands' commands
}
