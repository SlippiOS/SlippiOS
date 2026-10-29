package netpages

import (
	"html/template"

	"github.com/SlippiOS/SlippiOS/backend/internal/settings"
)

type pageData struct {
	Token      string
	Message    string
	Tone       string
	UploadOnly bool
}

var pageTemplate = template.Must(template.New("page").Parse(`<!doctype html>
<html lang="en"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>SlippiOS</title>
<style>
body{font-family:system-ui,sans-serif;background:#111;color:#eee;max-width:34rem;margin:2rem auto;padding:0 1rem;line-height:1.5}
h1{font-size:1.5rem}h2{font-size:1.1rem;margin-top:2rem}
input,button{font:inherit}input[type=text],input[type=email],input[type=password]{width:100%;padding:.6rem;margin:.3rem 0;border-radius:.4rem;border:1px solid #333;background:#1b1b1b;color:#eee}
button{background:#2b6cb0;color:#fff;border:0;border-radius:.4rem;padding:.6rem 1rem;margin-top:.4rem}
button[disabled]{opacity:.7;cursor:wait}
.spinner{display:inline-block;width:.9em;height:.9em;border:2px solid rgba(255,255,255,.35);border-top-color:#fff;border-radius:50%;animation:spin .8s linear infinite;vertical-align:-.12em;margin-right:.45em}
@keyframes spin{to{transform:rotate(360deg)}}
.ok{color:#7bd88f}.err{color:#ff8b8b}code{background:#222;padding:.1rem .3rem;border-radius:.2rem}
</style></head><body>
<h1>SlippiOS</h1>
<p>This page is only reachable on your local network.</p>
{{if .Message}}<p class="{{.Tone}}">{{.Message}}</p>{{end}}
{{if not .UploadOnly}}
<h2>Sign in to Slippi</h2>
<form method="post" action="/{{.Token}}/auth" onsubmit="busy(this,'Signing in…')">
  <input type="email" name="email" placeholder="Email" autocomplete="username" required>
  <input type="password" name="password" placeholder="Password" autocomplete="current-password" required>
  <button type="submit">Sign in</button>
</form>
{{end}}
<h2>Add a game image</h2>
<p>Select a Melee 1.02 ISO. It is copied to <code>/data/isos</code>.</p>
<input id="file" type="file" accept=".iso,application/x-iso9660-image">
<button type="button" id="upload-button" onclick="upload()">Upload</button>
<pre id="status"></pre>
<script>
function spinner() { const el = document.createElement('span'); el.className = 'spinner'; return el; }
function setStatus(text, working) {
  const el = document.getElementById('status');
  if (!el) return;
  el.replaceChildren();
  if (working) el.appendChild(spinner());
  el.appendChild(document.createTextNode(text));
}
function busy(form, label) {
  const button = form.querySelector('button[type="submit"]');
  if (!button || button.disabled) return;
  button.disabled = true;
  button.replaceChildren(spinner(), document.createTextNode(label));
}
function upload() {
  const file = document.getElementById('file').files[0];
  const button = document.getElementById('upload-button');
  if (!file) { setStatus('Choose an ISO first.', false); return; }
  if (button) { button.disabled = true; button.replaceChildren(spinner(), document.createTextNode('Uploading…')); }
  setStatus('Uploading ' + file.name + '…', true);
  const request = new XMLHttpRequest();
  request.open('POST', '/{{.Token}}/iso?name=' + encodeURIComponent(file.name));
  request.upload.onprogress = function(event) {
    if (!event.lengthComputable) return;
    setStatus('Uploading ' + file.name + '… ' + Math.round(event.loaded / event.total * 100) + '%', true);
  };
  request.onload = function() {
    if (request.status >= 200 && request.status < 300) {
      setStatus(request.responseText, false);
    } else {
      setStatus(request.responseText || 'Upload failed.', false);
      if (button) { button.disabled = false; button.textContent = 'Upload'; }
    }
  };
  request.onerror = function() {
    setStatus('Upload failed.', false);
    if (button) { button.disabled = false; button.textContent = 'Upload'; }
  };
  request.send(file);
}
</script></body></html>`))

type setupPage struct {
	Token        string
	GameReady    bool
	AccountReady bool
	Done         bool
	GameName     string
	Account      string
	ISOs         []settings.ISO
	Message      string
	Error        string
}

var setupTemplate = template.Must(template.New("setup").Parse(`<!doctype html>
<html lang="en"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>Set up SlippiOS</title>
<style>
:root{color-scheme:dark}
*{box-sizing:border-box}
body{margin:0;min-height:100vh;background:radial-gradient(circle at 50% -10%,#1d2738 0,#0b0b10 55%);color:#eef1f7;font-family:system-ui,-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;line-height:1.5}
main{width:min(36rem,calc(100% - 2rem));margin:0 auto;padding:3rem 0 4rem}
.brand{font-size:.85rem;font-weight:700;letter-spacing:.22em;text-transform:uppercase;color:#7db5ff}
h1{font-size:2rem;margin:.4rem 0 .2rem}
.sub{color:#9aa4b5;margin:0 0 2rem}
.card{background:#141821;border:1px solid #252c3a;border-radius:1rem;padding:1.4rem;box-shadow:0 20px 60px rgb(0 0 0 / .3)}
.steps{display:flex;align-items:center;gap:.6rem;margin:0 0 1.5rem}
.step{display:flex;align-items:center;gap:.45rem;color:#6f7a8c;font-size:.85rem;white-space:nowrap}
.step b{display:grid;place-items:center;width:1.6rem;height:1.6rem;border-radius:50%;background:#252c3a;font-size:.75rem}
.step.active{color:#eef1f7}.step.active b{background:#2b6cb0;color:#fff}
.step.done{color:#79d99a}.step.done b{background:#1f5c3a;color:#b9f5cf}
.line{height:1px;flex:1;background:#252c3a}
h2{font-size:1.25rem;margin:0 0 .35rem}
p{margin:.4rem 0}
.muted{color:#9aa4b5;font-size:.92rem}
.chosen{background:#0f131b;border:1px solid #252c3a;border-radius:.6rem;padding:.65rem .8rem;margin:.9rem 0;color:#cdd5e2}
label{display:block;font-size:.82rem;color:#9aa4b5;margin:.75rem 0 .25rem}
input,select,button{font:inherit}
input[type=text],input[type=email],input[type=password],select{width:100%;padding:.7rem .8rem;border-radius:.55rem;border:1px solid #30394a;background:#0f131b;color:#eef1f7}
button{width:100%;border:0;border-radius:.55rem;padding:.8rem 1rem;background:#2b6cb0;color:#fff;font-weight:650;cursor:pointer;margin-top:.7rem}
button:hover{background:#347cc7}
button.secondary{background:#222a37;color:#cbd4e2;font-weight:500}
button.secondary:hover{background:#2c3646}
button[disabled]{opacity:.7;cursor:wait}
.spinner{display:inline-block;width:.9em;height:.9em;border:2px solid rgba(255,255,255,.35);border-top-color:#fff;border-radius:50%;animation:spin .8s linear infinite;vertical-align:-.12em;margin-right:.45em}
@keyframes spin{to{transform:rotate(360deg)}}
.or{text-align:center;color:#6f7a8c;font-size:.82rem;margin:1rem 0 .2rem}
.status{min-height:1.4rem;color:#9aa4b5;font-size:.88rem;word-break:break-word}
.err{color:#ff8b8b}.ok{color:#7bd88f}
.done-title{color:#79d99a}
.small{font-size:.82rem;color:#6f7a8c}
</style></head><body><main>
<div class="brand">SlippiOS</div>
<h1>{{if .Done}}You're all set{{else}}Set up your console{{end}}</h1>
<p class="sub">This page only works on your local network.</p>
<section class="card">
  <div class="steps">
    <div class="step {{if .GameReady}}done{{else}}active{{end}}"><b>1</b> Game</div>
    <div class="line"></div>
    <div class="step {{if .AccountReady}}done{{else if .GameReady}}active{{end}}"><b>2</b> Account</div>
  </div>

  {{if .Error}}<p class="err">{{.Error}}</p>{{end}}
  {{if .Message}}<p class="ok">{{.Message}}</p>{{end}}

  {{if not .GameReady}}
    {{if .ISOs}}
    <h2>Choose your default Melee image</h2>
    <p class="muted">This starts when you power on. You can change it later from Melee ROMs in the shell.</p>
    <form method="post" action="/{{.Token}}/select" onsubmit="busy(this,'Saving…')">
      <label for="game">Game image</label>
      <select id="game" name="path">
        {{range .ISOs}}<option value="{{.Path}}">{{.Name}}</option>{{end}}
      </select>
      <button type="submit">Use this game</button>
    </form>
    <p class="or">or upload a new image</p>
    {{else}}
    <h2>Add your game</h2>
    <p class="muted">Upload a Melee 1.02 ISO. It will become your default image.</p>
    {{end}}
    <div>
      <input id="file" type="file" accept=".iso,application/x-iso9660-image">
      <button type="button" id="upload-button" onclick="upload()">Upload ISO</button>
      <p id="status" class="status"></p>
    </div>
  {{else if not .AccountReady}}
    <h2>Sign in to Slippi</h2>
    <div class="chosen">Game: {{.GameName}}</div>
    <p class="muted">Sign in for online play, or skip to play offline.</p>
    <form method="post" action="/{{.Token}}/auth" onsubmit="busy(this,'Signing in…')">
      <label for="email">Email</label>
      <input id="email" type="email" name="email" autocomplete="username" required>
      <label for="password">Password</label>
      <input id="password" type="password" name="password" autocomplete="current-password" required>
      <button type="submit">Sign in</button>
    </form>
    <form method="post" action="/{{.Token}}/skip" onsubmit="busy(this,'Skipping…')">
      <button type="submit" class="secondary">Skip for offline play</button>
    </form>
  {{else}}
    <h2 class="done-title">Melee is starting on your TV</h2>
    <p class="muted">You can close this page.</p>
    {{if .Account}}<p class="small">Slippi: {{.Account}}</p>{{end}}
  {{end}}
</section>
</main>
<script>
function spinner() { const el = document.createElement('span'); el.className = 'spinner'; return el; }
function setStatus(text, working) {
  const el = document.getElementById('status');
  if (!el) return;
  el.replaceChildren();
  if (working) el.appendChild(spinner());
  el.appendChild(document.createTextNode(text));
}
function busy(form, label) {
  const button = form.querySelector('button[type="submit"]');
  if (!button || button.disabled) return;
  button.disabled = true;
  button.replaceChildren(spinner(), document.createTextNode(label));
}
function upload() {
  const file = document.getElementById('file').files[0];
  const button = document.getElementById('upload-button');
  if (!file) { setStatus('Choose an ISO first.', false); return; }
  if (button) { button.disabled = true; button.replaceChildren(spinner(), document.createTextNode('Uploading…')); }
  setStatus('Uploading ' + file.name + '…', true);
  const request = new XMLHttpRequest();
  request.open('POST', '/{{.Token}}/iso?name=' + encodeURIComponent(file.name));
  request.upload.onprogress = function(event) {
    if (!event.lengthComputable) return;
    setStatus('Uploading ' + file.name + '… ' + Math.round(event.loaded / event.total * 100) + '%', true);
  };
  request.onload = function() {
    if (request.status >= 200 && request.status < 300) {
      window.location.href = '/{{.Token}}/';
    } else {
      setStatus(request.responseText || 'Upload failed.', false);
      if (button) { button.disabled = false; button.textContent = 'Upload ISO'; }
    }
  };
  request.onerror = function() {
    setStatus('Upload failed.', false);
    if (button) { button.disabled = false; button.textContent = 'Upload ISO'; }
  };
  request.send(file);
}
</script></body></html>`))
