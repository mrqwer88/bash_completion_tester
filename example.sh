_asdf() {
COMPREPLY=()
local cur prev
cur=$(_get_cword)
COMPREPLY=( $( compgen -W "one two three four five six" -- "$cur") )
return 0
}    
complete -F _asdf asdf
