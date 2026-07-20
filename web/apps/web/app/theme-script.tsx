"use client";

import { useRef } from "react";
import { useServerInsertedHTML } from "next/navigation";

// FOUC-prevention theme bootstrap. Runs during HTML parse, before hydration.
//
// We inject it via useServerInsertedHTML rather than rendering a <script> in
// the layout's JSX. React 19's resource hoisting re-creates head <script>
// elements on the client, which trips its dev-only "scripts inside React
// components are never executed" warning for ANY executable inline script
// (raw <script>, dangerouslySetInnerHTML, or next/script alike). Inserting it
// as a raw string keeps it out of the reconciled tree, so the client renderer
// never sees a <script> element — no warning, and the script still runs from
// the server-rendered HTML before first paint.
const themeScript = `(function(){try{var t=localStorage.getItem('theme')||'system';var d=t==='dark'||(t==='system'&&window.matchMedia('(prefers-color-scheme: dark)').matches);document.documentElement.classList.toggle('dark',d);}catch(e){}})();`;

export function ThemeScript() {
  // useServerInsertedHTML's callback fires on every stream flush (once per
  // suspense boundary), so guard with a per-request ref to inject exactly once
  // — into the first flush, which lands in <head> before any body content.
  const injected = useRef(false);
  useServerInsertedHTML(() => {
    if (injected.current) return null;
    injected.current = true;
    return (
      <script id="theme-init" dangerouslySetInnerHTML={{ __html: themeScript }} />
    );
  });
  return null;
}
