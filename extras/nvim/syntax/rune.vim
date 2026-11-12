if exists("b:current_syntax")
  finish
endif

syn keyword runeKeyword let mut fun return struct impl import export match is in
syn keyword runeConditional if else
syn keyword runeRepeat for while break continue
syn keyword runeException try catch throw
syn keyword runeOperatorWord and or
syn keyword runeStatement print
syn keyword runeBoolean true false
syn keyword runeNil nil
syn keyword runeSelf self
syn match runeAs "\<as\>"

syn keyword runeNamespace string math regex bytes json http time fs io os array
syn keyword runeBuiltin len type object
syn keyword runeType Map Set

syn match runeFuncDecl "\<fun\s\+\zs\h\w*"
syn match runeFuncCall "\h\w*\ze\s*("
syn match runeStructName "\<\(struct\|impl\)\s\+\zs\h\w*"
syn match runeConst "\<\u[A-Z0-9_]\+\>"

syn match runeNumber "\<\d[0-9_]*\(\.\d[0-9_]*\)\=\([eE][+-]\=\d[0-9_]*\)\=\>"
syn match runeNumber "\<0[xX][0-9a-fA-F_]\+\>"
syn match runeNumber "\<0[oO][0-7_]\+\>"
syn match runeNumber "\<0[bB][01_]\+\>"

syn match runeEscape contained "\\[ntr"\\{}]"
syn match runeByteEscape contained "\\x\x\x"
syn region runeInterp contained matchgroup=runeInterpDelim start="{" end="}" contains=TOP,runeInterpBrace
syn region runeInterpBrace contained transparent start="{" end="}" contains=TOP,runeInterpBrace
syn region runeString start=+"+ skip=+\\\\\|\\"+ end=+"+ contains=runeEscape,runeInterp
syn region runeBytes matchgroup=runeBytesPrefix start=+\<b"+ skip=+\\\\\|\\"+ end=+"+ contains=runeEscape,runeByteEscape
syn region runeRawString matchgroup=runeBytesPrefix start=+\<r"+ end=+"+

syn match runeOperator "=>\|\.\.\.\|\.\.=\|\.\.\|?\.\|??\|?\|[-+*/%=!<>&|^~]=\="

syn keyword runeTodo contained TODO FIXME XXX NOTE
syn match runeComment "//.*$" contains=runeTodo,@Spell

hi def link runeKeyword Keyword
hi def link runeConditional Conditional
hi def link runeRepeat Repeat
hi def link runeException Exception
hi def link runeOperatorWord Keyword
hi def link runeStatement Statement
hi def link runeAs Keyword
hi def link runeBoolean Boolean
hi def link runeNil Constant
hi def link runeSelf Special
hi def link runeNamespace Include
hi def link runeBuiltin Function
hi def link runeType Type
hi def link runeFuncDecl Function
hi def link runeFuncCall Function
hi def link runeStructName Type
hi def link runeConst Constant
hi def link runeNumber Number
hi def link runeString String
hi def link runeBytes String
hi def link runeRawString String
hi def link runeBytesPrefix Special
hi def link runeEscape SpecialChar
hi def link runeByteEscape SpecialChar
hi def link runeInterpDelim Delimiter
hi def link runeOperator Operator
hi def link runeComment Comment
hi def link runeTodo Todo

let b:current_syntax = "rune"
