#!/usr/bin/env zsh

#compdef eol

# Zsh completion for eol CLI tool
# Uses eol's own JSON API and caching for dynamic completion

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
        '--disable-cache[Disable caching]' \
        '--cache-dir[Cache directory]:directory:_directories' \
        '--cache-for[Cache TTL]:duration:' \
        '--template-dir[Template directory]:directory:_directories' \
        '(-h --help)'{-h,--help}'[Show help]' \
        '1: :_eol_commands' \
        '*:: :->args'

    case $state in
        args)
            case $words[1] in
                product|latest)
                    _eol_products
                    ;;
                release)
                    case $CURRENT in
                        2)
                            _eol_products
                            ;;
                        3)
                            _eol_product_versions $words[2]
                            ;;
                    esac
                    ;;
                categories)
                    case $CURRENT in
                        2)
                            _eol_categories
                            ;;
                    esac
                    ;;
                tags)
                    case $CURRENT in
                        2)
                            _eol_tags
                            ;;
                    esac
                    ;;
                identifiers)
                    case $CURRENT in
                        2)
                            _eol_identifier_types
                            ;;
                    esac
                    ;;
                cache)
                    _arguments \
                        '1:cache command:(stats clear)'
                    ;;
                templates)
                    case $CURRENT in
                        2)
                            _arguments \
                                '1:template command:(export)'
                            ;;
                        3)
                            case $words[2] in
                                export)
                                    _directories
                                    ;;
                            esac
                            ;;
                    esac
                    ;;
                completion)
                    _arguments \
                        '1:shell:(bash zsh)'
                    ;;
            esac
            ;;
    esac
}

_eol_commands() {
    local commands=(
        'index:Show API endpoints'
        'products:List all products'
        'product:Get details for a specific product'
        'release:Get specific release information'
        'latest:Get latest release information'
        'categories:List categories or products in a category'
        'tags:List tags or products with a tag'
        'identifiers:List identifier types or identifiers by type'
        'cache:Cache management commands'
        'templates:Template management commands'
        'completion:Shell completion scripts'
        'help:Show help message'
    )
    _describe 'commands' commands
}

_eol "$@"
