// Target of every data-changing <form> when its JavaScript handler never
// ran (blocked script, very old browser). Forms post here instead of
// submitting natively as GET, so passwords never land in a URL; the body
// is never read, logged or forwarded.
const page = `<!doctype html>
<html lang="pt-BR"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Terminus precisa de JavaScript</title>
<style>body{font:16px/1.5 system-ui,sans-serif;max-width:32rem;margin:15vh auto;padding:0 1rem;color:#111a16;background:#f2f5f3}a{color:#0b6b4b;font-weight:600}</style>
</head><body><h1>Nada foi enviado</h1>
<p>O Terminus precisa de JavaScript para salvar formulários, e ele não carregou nesta página.
Recarregue a página ou abra o endereço <strong>localhost</strong> em vez do IP.</p>
<p><a href="/">Voltar ao Terminus</a></p></body></html>`;

export function POST() {
  return new Response(page, {
    status: 400,
    headers: { "Content-Type": "text/html; charset=utf-8", "Cache-Control": "no-store" },
  });
}
