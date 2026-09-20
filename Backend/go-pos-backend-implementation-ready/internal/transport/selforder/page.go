// The public self-order page served at /selforder (and its service worker at
// /sw.js). The cashier prints a QR whose payload is ONLY this URL, so any
// browser can open it on the customer's own phone. The page:
//   - loads the menu from the public JSON endpoint (published + in-stock only),
//   - lets the customer build a cart and place a pending order,
//   - lets the customer request a product they want (feedback) and opt into
//     browser notifications for when it comes back online.
//
// VAPID identity is embedded from config; without it the notify option hides.
package selforder

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const ordersSWJS = `'use strict';
self.addEventListener('push', function (event) {
  var data = null;
  try { data = event.data ? event.data.json() : null; } catch (e) {}
  var title = data && data.title ? data.title : 'Your store';
  var options = {
    body: (data && data.body) || '',
    tag: (data && data.tag) || '',
    data: { url: (data && data.url) || '/selforder' }
  };
  event.waitUntil(self.registration.showNotification(title, options));
});
self.addEventListener('notificationclick', function (event) {
  event.notification.close();
  var url = (event.notification.data && event.notification.data.url) || '/selforder';
  event.waitUntil(clients.matchAll({ type: 'window', includeUncontrolled: true }).then(function (list) {
    for (var i = 0; i < list.length; i++) {
      if (list[i].url.indexOf('/selforder') !== -1) { list[i].navigate(url); return list[i].focus(); }
    }
    return clients.openWindow(url);
  }));
});
`

func (h *Handler) ServiceWorker(c *gin.Context) {
	c.Data(http.StatusOK, "application/javascript; charset=utf-8", []byte(ordersSWJS))
}

// ordersPageHTML embeds the tenant slug + VAPID public key; the empty key
// string keeps the notify toggle hidden.
func (h *Handler) ordersPageHTML(tenant, table, lang string) string {
	const page = `<!doctype html>
<html lang="{{LANG}}" dir="{{DIR}}">
  <head>
    <meta charset="utf-8"/>
    <meta name="viewport" content="width=device-width, initial-scale=1"/>
    <meta name="theme-color" content="#08131f"/>
    <title>Self Order — {{STORE}}</title>
    <link rel="icon" type="image/svg+xml" href="data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 96 96'%3E%3Crect x='4' y='4' width='88' height='88' rx='22' fill='%230f2233' stroke='%2322c55e' stroke-width='4'/%3E%3Ctext x='48' y='68' text-anchor='middle' font-family='Verdana' font-size='52' font-weight='700' fill='%2322c55e'%3EP%3C/text%3E%3C/svg%3E"/>
    <style>
      :root { --bg:#08131f; --card:#0f2233; --border:#1d3a52; --text:#e6eef5; --muted:#93a8ba; --accent:#22c55e; --accent-dark:#15803d; --warn:#f59e0b; }
      * { box-sizing:border-box; }
      body { font-family:'Segoe UI',system-ui,-apple-system,Roboto,sans-serif; margin:0; background:var(--bg); color:var(--text); min-height:100vh; }
      header { position:sticky; top:0; z-index:20; background:linear-gradient(180deg,#0c1d2c,#08131f); border-bottom:1px solid var(--border); padding:14px 16px; display:flex; align-items:center; justify-content:space-between; gap:10px; }
      header h1 { font-size:18px; margin:0; font-weight:700; }
      header .table { color:var(--muted); font-size:13px; }
      main { max-width:820px; margin:0 auto; padding:16px; }
      .bar { display:flex; justify-content:space-between; align-items:center; gap:10px; margin-bottom:14px; flex-wrap:wrap; }
      .bar .lang { color:var(--accent); background:none; border:1px solid var(--border); border-radius:10px; padding:6px 12px; cursor:pointer; font-size:13px; }
      .group { margin-bottom:18px; }
      .group h2 { font-size:15px; color:var(--muted); margin:0 0 10px; font-weight:600; }
      .grid { display:grid; grid-template-columns:repeat(auto-fill,minmax(230px,1fr)); gap:12px; }
      .card { background:var(--card); border:1px solid var(--border); border-radius:14px; padding:14px; display:flex; flex-direction:column; gap:8px; }
      .card .name { font-weight:700; font-size:15px; }
      .card .desc { color:var(--muted); font-size:12px; line-height:1.4; min-height:34px; }
      .card .foot { display:flex; justify-content:space-between; align-items:center; gap:8px; margin-top:auto; }
      .price { color:var(--accent); font-weight:700; }
      .bump { display:inline-flex; align-items:center; gap:8px; }
      .bump button { width:28px; height:28px; border-radius:8px; border:1px solid var(--border); background:#0a1a2a; color:var(--text); cursor:pointer; font-size:15px; }
      .bump .qty { width:22px; text-align:center; }
      .bell { background:none; border:none; cursor:pointer; color:var(--muted); font-size:16px; padding:2px; }
      .bell:hover { color:var(--accent); }
      .fab { position:fixed; right:18px; bottom:18px; z-index:30; background:var(--accent); color:#052e12; border:none; border-radius:999px; padding:14px 18px; font-weight:700; font-size:15px; box-shadow:0 8px 24px rgba(0,0,0,.4); cursor:pointer; }
      .fab .count { background:#052e12; color:#fff; border-radius:999px; min-width:20px; display:inline-block; text-align:center; padding:2px 5px; margin-left:8px; font-size:12px; }
      .drawer { position:fixed; inset:0 0 0 auto; width:min(380px,92vw); background:var(--card); border-left:1px solid var(--border); z-index:40; transform:translateX(100%); transition:transform .2s ease; display:flex; flex-direction:column; }
      html[dir="rtl"] .drawer { left:0; right:auto; border-left:none; border-right:1px solid var(--border); transform:translateX(-100%); }
      .drawer.open { transform:translateX(0); }
      .drawer .hd { padding:16px; border-bottom:1px solid var(--border); font-weight:700; display:flex; justify-content:space-between; align-items:center; }
      .drawer .bd { padding:16px; overflow-y:auto; flex:1; }
      .line { display:flex; align-items:center; justify-content:space-between; gap:10px; padding:8px 0; border-bottom:1px dashed var(--border); }
      .line .meta { font-size:13px; }
      .drawer .ft { padding:16px; border-top:1px solid var(--border); }
      .total { display:flex; justify-content:space-between; font-weight:700; font-size:18px; margin-bottom:12px; }
      button.go { width:100%; background:var(--accent); color:#052e12; border:none; border-radius:12px; padding:14px; font-weight:700; font-size:15px; cursor:pointer; }
      button.go:disabled { opacity:.5; cursor:default; }
      .close { background:none; border:none; color:var(--muted); font-size:20px; cursor:pointer; }
      .modal { position:fixed; inset:0; background:rgba(0,0,0,.6); display:none; align-items:center; justify-content:center; z-index:50; padding:16px; }
      .modal.open { display:flex; }
      .panel { background:var(--card); border:1px solid var(--border); border-radius:16px; padding:20px; width:min(420px,90vw); }
      .panel h3 { margin:0 0 12px; }
      label { display:block; font-size:12px; color:var(--muted); margin:10px 0 4px; }
      input, textarea { width:100%; background:#0a1a2a; border:1px solid var(--border); border-radius:10px; color:var(--text); padding:10px; font-size:14px; }
      .row { display:flex; align-items:center; gap:8px; margin-top:12px; }
      .row input[type=checkbox] { width:auto; }
      .actions { display:flex; gap:10px; margin-top:16px; }
      .actions button { flex:1; padding:11px; border-radius:10px; font-weight:600; cursor:pointer; border:1px solid var(--border); background:#0a1a2a; color:var(--text); }
      .actions button.primary { background:var(--accent); color:#052e12; }
      .msg { margin-top:10px; font-size:13px; color:var(--warn); min-height:18px; }
      .ok { color:var(--accent); }
      .empty { color:var(--muted); text-align:center; padding:40px 0; }
    </style>
  </head>
  <body>
    <header>
      <h1>{{STORE}}<span class="table" id="tableName"></span></h1>
      <button class="lang" id="langBtn"></button>
    </header>
    <main>
      <div class="bar">
        <div class="table" id="storeMeta"></div>
      </div>
      <div id="sections"></div>
      <div class="empty" id="empty" style="display:none"></div>
      <div class="empty" id="requestEntry" style="display:none;border-top:1px solid var(--border);margin-top:24px;padding-top:24px">
        <button class="lang" id="wantBtn" style="color:var(--accent);border:1px solid var(--accent)"></button>
      </div>
    </main>
    <button class="fab" id="cartBtn" style="display:none"><span id="cartLabel"></span><span class="count" id="cartCount">0</span></button>

    <div class="drawer" id="drawer">
      <div class="hd"><span id="drawerTitle"></span><button class="close" id="closeBtn">&times;</button></div>
      <div class="bd" id="drawerBody"></div>
      <div class="ft">
        <div class="total"><span id="totalLabel"></span><span id="totalVal"></span></div>
        <button class="go" id="orderBtn"></button>
        <div class="msg" id="orderMsg"></div>
      </div>
    </div>

    <div class="modal" id="requestModal">
      <div class="panel">
        <h3 id="reqTitle"></h3>
        <label id="reqNameLabel"></label>
        <input id="reqName" type="text"/>
        <label id="reqNoteLabel"></label>
        <textarea id="reqNote" rows="2"></textarea>
        <label id="reqContactLabel"></label>
        <input id="reqContact" type="text"/>
        <div class="row" id="notifyRow" style="display:none">
          <input id="notifyCheck" type="checkbox"/>
          <label for="notifyCheck" style="margin:0" id="notifyLabel"></label>
        </div>
        <div class="actions">
          <button id="reqCancel"></button>
          <button class="primary" id="reqSend"></button>
        </div>
        <div class="msg" id="reqMsg"></div>
      </div>
    </div>

    <div class="modal" id="doneModal">
      <div class="panel" style="text-align:center">
        <h3 id="doneTitle"></h3>
        <p id="doneRef" class="ok"></p>
        <div class="actions"><button class="primary" id="doneClose"></button></div>
      </div>
    </div>

    <script>
      var T = new URLSearchParams(location.search).get('tenant') || '';
      var TABLE = new URLSearchParams(location.search).get('table') || '';
      var lang = new URLSearchParams(location.search).get('lang') || '{{LANG}}';
      var dir = lang === 'en' ? 'ltr' : 'rtl';
      document.documentElement.setAttribute('dir', dir);
      document.documentElement.setAttribute('lang', lang);
      var VAPID_PUBLIC = {{VAPID_PUBLIC}};
      var I = {
        en: { langBtn:'العربية', cart:'Cart', total:'Total', order:'Place order', ordering:'Ordering…',
          placed:'Order sent to the counter!', ref:'Reference', want:'Not seeing what you need? Request a product',
          reqTitle:'Request a product', reqName:'Product name', reqNote:'Note (size, brand, …)', reqContact:'Your phone (optional)',
          notify:'Notify me in the browser when it is back in stock', cancel:'Cancel', send:'Send', sending:'Sending…',
          sent:'Request sent. We will let you know!', err:'Something went wrong. Try again.', empty:'Nothing on the menu right now.',
          add:'Add to order', noNotify:'Notifications are blocked', qty:'qty', table:'Table' },
        ar: { langBtn:'English', cart:'السلة', total:'الإجمالي', order:'إرسال الطلب', ordering:'جارٍ الإرسال…',
          placed:'تم إرسال طلبك إلى الكاشير!', ref:'رقم الطلب', want:'لا تجد ما تريد؟ اطلب منتجاً',
          reqTitle:'اطلب منتجاً', reqName:'اسم المنتج', reqNote:'ملاحظة (مقاس، ماركة، …)', reqContact:'هاتفك (اختياري)',
          notify:'أخطرني في المتصفح عندما يتوفر مرة أخرى', cancel:'إلغاء', send:'إرسال', sending:'جارٍ الإرسال…',
          sent:'تم إرسال طلبك وسنخطرك!', err:'حدث خطأ، حاول مرة أخرى.', empty:'لا توجد منتجات في القائمة حالياً.',
          add:'أضف للطلب', noNotify:'الإشعارات محظورة', qty:'الكمية', table:'طاولة' }
      };
      var tr = function(k){ return (I[lang] && I[lang][k]) || I.ar[k]; };
      var fmt = function(minor){ return 'L.E ' + (minor / 100).toFixed(2); };
      var cart = {}; var store = null;

      function b64url(bytes){
        var s = '';
        bytes.forEach(function(b){ s += String.fromCharCode(b); });
        return btoa(s).replace(/\+/g,'-').replace(/\//g,'_').replace(/=+$/,'');
      }

      fetch('/v1/selforder/menu/' + encodeURIComponent(T)).then(function(r){
        if (!r.ok) throw new Error('menu');
        return r.json();
      }).then(function(j){
        store = j.data;
        document.title = 'Self Order — ' + store.tenant.name;
        document.getElementById('storeMeta').textContent = store.tenant.address || '';
        var table = document.getElementById('tableName');
        if (TABLE) { table.textContent = ' — ' + tr('table') + ' ' + TABLE; }
        render(store);
        document.getElementById('cartBtn').style.display = 'inline-flex';
        document.getElementById('requestEntry').style.display = '';
        document.getElementById('wantBtn').textContent = tr('want');
      }).catch(function(){
        var e = document.getElementById('empty');
        e.style.display = '';
        e.textContent = tr('err');
      });

      function render(store){
        var wrap = document.getElementById('sections');
        wrap.innerHTML = '';
        var placed = 0;
        store.categories.forEach(function(cat){
          if (!cat.products.length) return;
          placed++;
          var g = document.createElement('div'); g.className = 'group';
          g.innerHTML = '<h2>' + cat.name + '</h2><div class="grid"></div>';
          var grid = g.querySelector('.grid');
          cat.products.forEach(function(p){ grid.appendChild(productCard(p)); });
          wrap.appendChild(g);
        });
        if (store.uncategorized && store.uncategorized.length) {
          placed++;
          var g2 = document.createElement('div'); g2.className = 'group';
          g2.innerHTML = '<h2>' + (lang==='en'?'Other':'أخرى') + '</h2><div class="grid"></div>';
          var grid2 = g2.querySelector('.grid');
          store.uncategorized.forEach(function(p){ grid2.appendChild(productCard(p)); });
          wrap.appendChild(g2);
        }
        if (!placed) { var e = document.getElementById('empty'); e.style.display=''; e.textContent = tr('empty'); }
      }

      function productCard(p){
        var d = document.createElement('div'); d.className = 'card';
        var desc = p.description || '';
        var bell = '<button class="bell" title="' + tr('want') + '" onclick="askProduct(\'' + p.id + '\',\'' + p.name.replace(/'/g,"\\'") + '\')">&#128276;</button>';
        d.innerHTML =
          '<div class="name">' + p.name + ' ' + bell + '</div>' +
          '<div class="desc">' + desc + '</div>' +
          '<div class="foot"><span class="price">' + fmt(p.price_minor) + '</span>' +
          '<span class="bump"><button onclick="dec(\'' + p.id + '\')">-</button><span class="qty" id="q_' + p.id + '">0</span><button onclick="inc(\'' + p.id + '\')">+</button></span></div>';
        return d;
      }

      function syncQty(p){ var el = document.getElementById('q_' + p); if (el) el.textContent = cart[p] || 0; }
      function count(){ var n = 0; for (var k in cart) n += cart[k]; return n; }
      window.inc = function(p){ cart[p] = (cart[p] || 0) + 1; syncQty(p); refreshFab(); };
      window.dec = function(p){ if (!cart[p]) return; cart[p]--; if (!cart[p]) delete cart[p]; syncQty(p); refreshFab(); };
      function refreshFab(){
        var n = count();
        document.getElementById('cartCount').textContent = n;
        document.getElementById('cartLabel').textContent = tr('cart');
      }

      var orderBtn = document.getElementById('orderBtn');
      document.getElementById('closeBtn').onclick = function(){ drawer(false); };
      document.getElementById('cartBtn').onclick = function(){ renderDrawer(); drawer(true); };
      function drawer(open){ document.getElementById('drawer').classList.toggle('open', open); }

      function lineItems(){
        var items = []; var sub = 0;
        for (var k in cart) {
          var p = findProduct(k); if (!p) continue;
          items.push({product_id: k, quantity: cart[k]});
          sub += p.price_minor * cart[k];
        }
        return {items: items, sub: sub};
      }
      function findProduct(id){
        var walk = function(list){ for (var i=0;i<list.length;i++) if (list[i].id === id) return list[i]; return null; };
        var p = null;
        (store.categories || []).forEach(function(c){ if (!p) p = walk(c.products); });
        if (!p) p = walk(store.uncategorized || []);
        return p;
      }
      function renderDrawer(){
        var body = document.getElementById('drawerBody'); body.innerHTML = '';
        var rows = 0;
        for (var k in cart) {
          var p = findProduct(k); if (!p) continue; rows++;
          var l = document.createElement('div'); l.className = 'line';
          l.innerHTML = '<div class="meta">' + p.name + '<br/><span class="price">' + fmt(p.price_minor) + '</span></div>' +
            '<span class="bump"><button onclick="dec(\'' + k + '\')">-</button><span class="qty">' + cart[k] + '</span><button onclick="inc(\'' + k + '\')">+</button></span>';
          body.appendChild(l);
        }
        if (!rows) body.innerHTML = '<div class="empty">' + tr('empty') + '</div>';
        var li = lineItems();
        document.getElementById('totalVal').textContent = fmt(li.sub);
        orderBtn.disabled = !rows;
        orderBtn.textContent = tr('order');
        orderBtn.onclick = function(){
          orderBtn.disabled = true; orderBtn.textContent = tr('ordering');
          document.getElementById('orderMsg').textContent = '';
          fetch('/v1/selforder/orders', {method:'POST', headers:{'Content-Type':'application/json'},
            body: JSON.stringify({tenant: T, table_name: TABLE, items: li.items})})
          .then(function(r){ return r.json().then(function(j){ return {ok:r.ok, j:j}; }); })
          .then(function(res){
            if (!res.ok) throw new Error((res.j && res.j.error && res.j.error.message) || tr('err'));
            cart = {}; store.categories.forEach(function(c){ c.products.forEach(function(p){ syncQty(p.id); }); });
            (store.uncategorized||[]).forEach(function(p){ syncQty(p.id); }); refreshFab(); drawer(false);
            document.getElementById('doneRef').textContent = tr('ref') + ': ' + res.j.data.reference;
            document.getElementById('doneModal').classList.add('open');
          }).catch(function(e){ document.getElementById('orderMsg').textContent = e.message || tr('err'); orderBtn.disabled = false; orderBtn.textContent = tr('order'); });
        };
      }

      var wantNode = document.getElementById('requestModal');
      document.getElementById('wantBtn').onclick = function(){ openRequest('', ''); };
      window.askProduct = function(id, name){ openRequest(id, name); };
      function openRequest(id, name){
        document.getElementById('reqName').value = name || '';
        document.getElementById('reqNote').value = '';
        document.getElementById('reqContact').value = '';
        document.getElementById('reqMsg').textContent = '';
        document.getElementById('notifyRow').style.display = VAPID_PUBLIC ? '' : 'none';
        document.getElementById('notifyCheck').checked = false;
        document.getElementById('reqTitle').textContent = tr('reqTitle');
        document.getElementById('reqNameLabel').textContent = tr('reqName');
        document.getElementById('reqNoteLabel').textContent = tr('reqNote');
        document.getElementById('reqContactLabel').textContent = tr('reqContact');
        document.getElementById('notifyLabel').textContent = tr('notify');
        document.getElementById('reqCancel').textContent = tr('cancel');
        document.getElementById('reqSend').textContent = tr('send');
        wantNode.__product = id;
        wantNode.classList.add('open');
      }
      document.getElementById('reqCancel').onclick = function(){ wantNode.classList.remove('open'); };
      document.getElementById('doneClose').onclick = function(){ document.getElementById('doneModal').classList.remove('open'); };

      function urlB64ToUint8Array(b64){
        var pad = b64.replace(/=+$/,'');
        var raw = atob(pad);
        var arr = new Uint8Array(raw.length);
        for (var i=0;i<raw.length;i++) arr[i] = raw.charCodeAt(i);
        return arr;
      }
      document.getElementById('reqSend').onclick = function(){
        var btn = document.getElementById('reqSend');
        btn.disabled = true; btn.textContent = tr('sending');
        document.getElementById('reqMsg').textContent = '';
        var pid = wantNode.__product || '';
        var body = {tenant: T, product_id: pid, product_name: pid ? '' : document.getElementById('reqName').value,
          note: document.getElementById('reqNote').value, contact: document.getElementById('reqContact').value};
        var doSend = function(sub){
          if (sub) body.webpush = sub;
          return fetch('/v1/selforder/requests', {method:'POST', headers:{'Content-Type':'application/json'}, body: JSON.stringify(body)})
          .then(function(r){ return r.json().then(function(j){ return {ok:r.ok, j:j}; }); });
        };
        (function(){
          if (!document.getElementById('notifyCheck').checked || !VAPID_PUBLIC) return Promise.resolve(null);
          return Notification.requestPermission().then(function(perm){
            if (perm !== 'granted') { document.getElementById('reqMsg').textContent = tr('noNotify'); return null; }
            return navigator.serviceWorker.register('/sw.js').then(function(reg){
              return reg.pushManager.subscribe({userVisibleOnly:true, applicationServerKey: urlB64ToUint8Array(VAPID_PUBLIC)});
            }).then(function(sub){
              return {endpoint: sub.endpoint, keys: {p256dh: b64url(new Uint8Array(sub.getKey('p256dh'))),
                auth: b64url(new Uint8Array(sub.getKey('auth')))}};
            });
          });
        })().then(doSend).then(function(res){
          if (!res.ok) throw new Error((res.j && res.j.error && res.j.error.message) || tr('err'));
          document.getElementById('reqMsg').textContent = tr('sent');
          document.getElementById('reqMsg').className = 'msg ok';
          setTimeout(function(){ wantNode.classList.remove('open'); }, 1200);
        }).catch(function(e){
          document.getElementById('reqMsg').textContent = e.message || tr('err');
          btn.disabled = false; btn.textContent = tr('send');
        });
      };

      document.getElementById('langBtn').textContent = tr('langBtn');
      document.getElementById('langBtn').onclick = function(){
        var next = lang === 'en' ? 'ar' : 'en';
        location.href = '/selforder?tenant=' + encodeURIComponent(T) + (TABLE ? '&table=' + encodeURIComponent(TABLE) : '') + '&lang=' + next;
      };
    </script>
  </body>
</html>`

	dir := "ltr"
	if lang == "ar" {
		dir = "rtl"
	}
	store := tenant
	// {{LANG}}/{{DIR}}+store placeholders:
	html := page
	html = strings.ReplaceAll(html, "{{LANG}}", lang)
	html = strings.ReplaceAll(html, "{{DIR}}", dir)
	html = strings.ReplaceAll(html, "{{STORE}}", store)
	if h.vapid.PublicKey != "" {
		html = strings.ReplaceAll(html, "{{VAPID_PUBLIC}}", `'`+h.vapid.PublicKey+`'`)
	} else {
		html = strings.ReplaceAll(html, "{{VAPID_PUBLIC}}", `''`)
	}
	return html
}

// Page serves the self-order web app. tenant comes from the ?tenant= query
// parameter (slug or UUID); lang from ?lang= (ar/en). The page itself fetches
// the JSON menu from /v1/selforder/menu/:tenant.
func (h *Handler) Page(c *gin.Context) {
	tenant := c.Query("tenant")
	if tenant == "" {
		c.Data(http.StatusBadRequest, "text/html; charset=utf-8", []byte(`<!doctype html><html><body><h1>Missing ?tenant=</h1><p>Open the QR from inside the store, or ask the cashier for the menu link.</p></body></html>`))
		return
	}
	lang := c.Query("lang")
	if lang != "en" && lang != "ar" {
		lang = "ar"
	}
	table := c.Query("table")
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(h.ordersPageHTML(tenant, table, lang)))
}
