hljs.registerLanguage("rune", function (hljs) {
  const SUBST = { className: "subst", begin: /\{/, end: /\}/, keywords: {} };
  const STRING = {
    className: "string",
    variants: [
      { begin: /r"""/, end: /"""/ },
      { begin: /r"/, end: /"/ },
      { begin: /b"/, end: /"/, contains: [hljs.BACKSLASH_ESCAPE] },
      { begin: /"""/, end: /"""/, contains: [hljs.BACKSLASH_ESCAPE, SUBST] },
      { begin: /"/, end: /"/, contains: [hljs.BACKSLASH_ESCAPE, SUBST] },
    ],
  };
  const KEYWORDS = {
    keyword:
      "let mut fun return struct impl import export as match is in if else for while break continue try catch throw step self",
    literal: "true false nil",
    built_in:
      "print len type object str number int Map Set string math regex bytes json http time fs io os array module",
  };
  SUBST.keywords = KEYWORDS;
  const NUMBER = {
    className: "number",
    variants: [
      { begin: /\b0[xX][0-9a-fA-F_]+/ },
      { begin: /\b0[oO][0-7_]+/ },
      { begin: /\b0[bB][01_]+/ },
      { begin: /\b\d[\d_]*(\.\d[\d_]*)?([eE][+-]?\d[\d_]*)?/ },
    ],
    relevance: 0,
  };
  SUBST.contains = [STRING, NUMBER];
  return {
    name: "Rune",
    keywords: KEYWORDS,
    contains: [
      hljs.C_LINE_COMMENT_MODE,
      hljs.C_BLOCK_COMMENT_MODE,
      STRING,
      NUMBER,
      { className: "title", begin: /\b(?<=fun\s+)[A-Za-z_]\w*/ },
      { className: "type", begin: /\b[A-Z][A-Za-z0-9_]*[a-z][A-Za-z0-9_]*\b/ },
    ],
  };
});

document.querySelectorAll("code.language-rune").forEach(function (block) {
  block.textContent = block.textContent;
  hljs.highlightBlock(block);
});
