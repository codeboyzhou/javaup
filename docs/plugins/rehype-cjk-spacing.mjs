const CJK =
  '\\u3400-\\u4dbf' + // CJK Extension A
  '\\u4e00-\\u9fff' + // CJK Unified Ideographs
  '\\u3000-\\u303f' + // CJK Symbols and Punctuation
  '\\uff00-\\uffef'; // Halfwidth and Fullwidth Forms

const CJK_SPACE_RE = new RegExp(`([${CJK}])\\s+(?=[${CJK}])`, 'g');

const SKIP_TAGS = new Set(['pre', 'code', 'script', 'style', 'textarea']);

export default function rehypeCjkSpacing() {
  return (tree) => {
    const visit = (node, skip) => {
      if (node.type === 'text') {
        if (!skip && typeof node.value === 'string') {
          node.value = node.value.replace(CJK_SPACE_RE, '$1');
        }
        return;
      }
      const nextSkip = skip || (node.tagName && SKIP_TAGS.has(node.tagName));
      for (const child of node.children || []) {
        visit(child, nextSkip);
      }
    };
    visit(tree, false);
  };
}
